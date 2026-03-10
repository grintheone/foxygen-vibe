package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Status   string `json:"status"`
		Database struct {
			Configured bool `json:"configured"`
			Connected  bool `json:"connected"`
		} `json:"database"`
		Storage struct {
			Configured bool   `json:"configured"`
			Connected  bool   `json:"connected"`
			Bucket     string `json:"bucket,omitempty"`
		} `json:"storage"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := response{Status: "ok"}
	payload.Database.Configured = s.databaseConfigured
	payload.Database.Connected = s.db != nil
	payload.Storage.Configured = s.storageConfigured
	payload.Storage.Connected = s.storage != nil
	if s.storage != nil {
		payload.Storage.Bucket = s.storage.Bucket()
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	type createAccountRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type createAccountResponse struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Disabled bool   `json:"disabled"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.queries == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	defer r.Body.Close()

	var input createAccountRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)

	switch {
	case input.Username == "":
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	case input.Password == "":
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		log.Printf("password hash failed: %v", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	store := s.queries
	var tx pgx.Tx

	if s.db != nil {
		tx, err = s.db.Begin(ctx)
		if err != nil {
			log.Printf("begin create account transaction failed: %v", err)
			http.Error(w, "failed to create account", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		store = appdb.New(tx)
	}

	account, err := store.CreateAccount(ctx, appdb.CreateAccountParams{
		Username:     input.Username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}

		log.Printf("create account failed: %v", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	if _, err := store.CreateUserProfile(ctx, account.UserID); err != nil {
		log.Printf("create user profile failed: %v", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("commit create account transaction failed: %v", err)
			http.Error(w, "failed to create account", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusCreated, createAccountResponse{
		UserID:   account.UserID.String(),
		Username: account.Username,
		Disabled: account.Disabled,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	type loginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var input loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)

	switch {
	case input.Username == "":
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	case input.Password == "":
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	if s.queries == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	store := s.queries
	var tx pgx.Tx
	var err error

	if s.db != nil {
		tx, err = s.db.Begin(ctx)
		if err != nil {
			log.Printf("begin login transaction failed: %v", err)
			http.Error(w, "failed to authenticate", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		store = appdb.New(tx)
	}

	account, err := store.GetAccountByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Printf("load account failed: %v", err)
		http.Error(w, "failed to authenticate", http.StatusInternalServerError)
		return
	}

	if account.Disabled {
		http.Error(w, "account is disabled", http.StatusForbidden)
		return
	}

	if !verifyPassword(input.Password, account.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	response, err := s.issueTokenPair(ctx, store, account)
	if err != nil {
		log.Printf("issue token pair failed: %v", err)
		http.Error(w, "failed to authenticate", http.StatusInternalServerError)
		return
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("commit login transaction failed: %v", err)
			http.Error(w, "failed to authenticate", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	type refreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var input refreshRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.RefreshToken == "" {
		http.Error(w, "refresh_token is required", http.StatusBadRequest)
		return
	}

	if s.queries == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	store := s.queries
	var tx pgx.Tx
	var err error

	if s.db != nil {
		tx, err = s.db.Begin(ctx)
		if err != nil {
			log.Printf("begin refresh transaction failed: %v", err)
			http.Error(w, "failed to refresh session", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		store = appdb.New(tx)
	}

	current, err := store.GetRefreshTokenByHash(ctx, hashOpaqueToken(input.RefreshToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}

		log.Printf("load refresh token failed: %v", err)
		http.Error(w, "failed to refresh session", http.StatusInternalServerError)
		return
	}

	if err := validateStoredRefreshToken(current); err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	account, err := store.GetAccountByUserID(ctx, current.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid refresh token", http.StatusUnauthorized)
			return
		}

		log.Printf("load refresh token account failed: %v", err)
		http.Error(w, "failed to refresh session", http.StatusInternalServerError)
		return
	}

	if account.Disabled {
		http.Error(w, "account is disabled", http.StatusForbidden)
		return
	}

	response, err := s.issueTokenPair(ctx, store, account)
	if err != nil {
		log.Printf("issue refreshed token pair failed: %v", err)
		http.Error(w, "failed to refresh session", http.StatusInternalServerError)
		return
	}

	replacement, err := store.GetRefreshTokenByHash(ctx, hashOpaqueToken(response.RefreshToken))
	if err != nil {
		log.Printf("load replacement refresh token failed: %v", err)
		http.Error(w, "failed to refresh session", http.StatusInternalServerError)
		return
	}

	rows, err := store.RotateRefreshToken(ctx, appdb.RotateRefreshTokenParams{
		TokenID:    current.TokenID,
		ReplacedBy: replacement.TokenID,
	})
	if err != nil {
		log.Printf("rotate refresh token failed: %v", err)
		http.Error(w, "failed to refresh session", http.StatusInternalServerError)
		return
	}
	if err := refreshConflict(rows); err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("commit refresh transaction failed: %v", err)
			http.Error(w, "failed to refresh session", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	type sessionResponse struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		UserID:   claims.Subject,
		Username: claims.Username,
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	type profileResponse struct {
		UserID     string `json:"user_id"`
		Username   string `json:"username"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Department string `json:"department"`
		Role       string `json:"role"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.queries == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	profile, err := s.queries.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		log.Printf("load user profile failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, profileResponse{
		UserID:     uuidToString(profile.UserID),
		Username:   profile.Username,
		Name:       profile.Name,
		Email:      profile.Email,
		Department: profile.Department,
		Role:       profile.Role,
	})
}

func (s *Server) handleTickets(w http.ResponseWriter, r *http.Request) {
	type ticketResponse struct {
		ID                 string  `json:"id"`
		Number             int32   `json:"number"`
		Status             string  `json:"status"`
		Description        string  `json:"description"`
		Reason             string  `json:"reason"`
		Urgent             bool    `json:"urgent"`
		Executor           *string `json:"executor"`
		ExecutorName       string  `json:"executorName"`
		ExecutorDepartment string  `json:"executorDepartment"`
		AssignedEnd        *string `json:"assigned_end"`
		WorkstartedAt      *string `json:"workstarted_at"`
		WorkfinishedAt     *string `json:"workfinished_at"`
		DeviceName         string  `json:"deviceName"`
		DeviceSerialNumber string  `json:"deviceSerialNumber"`
		ClientName         string  `json:"clientName"`
		ClientAddress      string  `json:"clientAddress"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	var executorID pgtype.UUID
	if err := executorID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT
			t.id,
			t.number,
			COALESCE(t.status, ''),
			t.description,
			COALESCE(
				NULLIF(
					CASE
						WHEN t.status = 'assigned' THEN tr.future
						WHEN t.status = 'worksDone' THEN tr.past
						ELSE tr.present
					END,
					''
				),
				NULLIF(tr.title, ''),
				'Не указано'
			) AS resolved_reason,
			t.urgent,
			t.executor,
			TRIM(CONCAT(COALESCE(u_exec.first_name, ''), ' ', COALESCE(u_exec.last_name, ''))),
			COALESCE(d_exec.title, ''),
			t.assigned_end,
			t.workstarted_at,
			t.workfinished_at,
			COALESCE(cls.title, ''),
			COALESCE(d.serial_number, ''),
			COALESCE(c.title, ''),
			COALESCE(c.address, '')
		FROM tickets t
		LEFT JOIN clients c ON c.id = t.client
		LEFT JOIN devices d ON d.id = t.device
		LEFT JOIN classificators cls ON cls.id = d.classificator
		LEFT JOIN ticket_reasons tr ON tr.id = t.reason
		LEFT JOIN users u_exec ON u_exec.user_id = t.executor
		LEFT JOIN departments d_exec ON d_exec.id = u_exec.department
		WHERE t.executor = $1
		ORDER BY t.assigned_end DESC NULLS LAST, t.created_at DESC, t.number DESC
	`, executorID)
	if err != nil {
		log.Printf("query tickets failed: %v", err)
		http.Error(w, "failed to load tickets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tickets := make([]ticketResponse, 0)
	for rows.Next() {
		var (
			id             pgtype.UUID
			number         pgtype.Int4
			status         string
			description    string
			reason         string
			urgent         bool
			executor       pgtype.UUID
			executorName   string
			executorDept   string
			assignedEnd    pgtype.Timestamp
			workstartedAt  pgtype.Timestamp
			workfinishedAt pgtype.Timestamp
			deviceName     string
			deviceSerial   string
			clientName     string
			clientAddress  string
		)

		if err := rows.Scan(
			&id,
			&number,
			&status,
			&description,
			&reason,
			&urgent,
			&executor,
			&executorName,
			&executorDept,
			&assignedEnd,
			&workstartedAt,
			&workfinishedAt,
			&deviceName,
			&deviceSerial,
			&clientName,
			&clientAddress,
		); err != nil {
			log.Printf("scan ticket failed: %v", err)
			http.Error(w, "failed to load tickets", http.StatusInternalServerError)
			return
		}

		tickets = append(tickets, ticketResponse{
			ID:                 uuidToString(id),
			Number:             number.Int32,
			Status:             status,
			Description:        description,
			Reason:             reason,
			Urgent:             urgent,
			Executor:           nullableUUIDToString(executor),
			ExecutorName:       executorName,
			ExecutorDepartment: executorDept,
			AssignedEnd:        timestampToRFC3339(assignedEnd),
			WorkstartedAt:      timestampToRFC3339(workstartedAt),
			WorkfinishedAt:     timestampToRFC3339(workfinishedAt),
			DeviceName:         deviceName,
			DeviceSerialNumber: deviceSerial,
			ClientName:         clientName,
			ClientAddress:      clientAddress,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate tickets failed: %v", err)
		http.Error(w, "failed to load tickets", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tickets)
}

func (s *Server) handleTicketByID(w http.ResponseWriter, r *http.Request) {
	type ticketResponse struct {
		ID                 string   `json:"id"`
		Number             int32    `json:"number"`
		Status             string   `json:"status"`
		StatusTitle        string   `json:"statusTitle"`
		Description        string   `json:"description"`
		Result             string   `json:"result"`
		Reason             string   `json:"reason"`
		ResolvedReason     string   `json:"resolvedReason"`
		TicketType         *string  `json:"ticketType"`
		TicketTypeTitle    string   `json:"ticketTypeTitle"`
		Urgent             bool     `json:"urgent"`
		DoubleSigned       bool     `json:"doubleSigned"`
		CreatedAt          *string  `json:"created_at"`
		AssignedAt         *string  `json:"assigned_at"`
		WorkstartedAt      *string  `json:"workstarted_at"`
		WorkfinishedAt     *string  `json:"workfinished_at"`
		PlannedStart       *string  `json:"planned_start"`
		PlannedEnd         *string  `json:"planned_end"`
		AssignedStart      *string  `json:"assigned_start"`
		AssignedEnd        *string  `json:"assigned_end"`
		ClosedAt           *string  `json:"closed_at"`
		Client             *string  `json:"client"`
		ClientName         string   `json:"clientName"`
		ClientAddress      string   `json:"clientAddress"`
		Device             *string  `json:"device"`
		DeviceName         string   `json:"deviceName"`
		DeviceSerialNumber string   `json:"deviceSerialNumber"`
		Author             *string  `json:"author"`
		AuthorName         string   `json:"authorName"`
		Department         *string  `json:"department"`
		DepartmentTitle    string   `json:"departmentTitle"`
		AssignedBy         *string  `json:"assignedBy"`
		AssignedByName     string   `json:"assignedByName"`
		ContactPerson      *string  `json:"contactPerson"`
		ContactName        string   `json:"contactName"`
		ContactPosition    string   `json:"contactPosition"`
		ContactPhone       string   `json:"contactPhone"`
		ContactEmail       string   `json:"contactEmail"`
		Executor           *string  `json:"executor"`
		ExecutorName       string   `json:"executorName"`
		ExecutorDepartment string   `json:"executorDepartment"`
		ReferenceTicket    *string  `json:"referenceTicket"`
		UsedMaterials      []string `json:"usedMaterials"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	ticketIDPath := strings.TrimPrefix(r.URL.Path, "/api/tickets/")
	if ticketIDPath == "" || strings.Contains(ticketIDPath, "/") {
		http.NotFound(w, r)
		return
	}

	var ticketID pgtype.UUID
	if err := ticketID.Scan(ticketIDPath); err != nil {
		http.Error(w, "invalid ticket id", http.StatusBadRequest)
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	row := s.db.QueryRow(ctx, `
		SELECT
			t.id,
			t.number,
			COALESCE(t.status, ''),
			COALESCE(ts.title, ''),
			t.description,
			t.result,
			COALESCE(t.reason, ''),
			COALESCE(
				NULLIF(
					CASE
						WHEN t.status = 'assigned' THEN tr.future
						WHEN t.status = 'worksDone' THEN tr.past
						ELSE tr.present
					END,
					''
				),
				NULLIF(tr.title, ''),
				'Не указано'
			) AS resolved_reason,
			t.ticket_type,
			COALESCE(tt.title, ''),
			t.urgent,
			t.double_signed,
			t.created_at,
			t.assigned_at,
			t.workstarted_at,
			t.workfinished_at,
			t.planned_start,
			t.planned_end,
			t.assigned_start,
			t.assigned_end,
			t.closed_at,
			t.client,
			COALESCE(c.title, ''),
			COALESCE(c.address, ''),
			t.device,
			COALESCE(cls.title, ''),
			COALESCE(d.serial_number, ''),
			t.author,
			TRIM(CONCAT(COALESCE(u_author.first_name, ''), ' ', COALESCE(u_author.last_name, ''))),
			t.department,
			COALESCE(dpt.title, ''),
			t.assigned_by,
			TRIM(CONCAT(COALESCE(u_assigned.first_name, ''), ' ', COALESCE(u_assigned.last_name, ''))),
			t.contact_person,
			COALESCE(cp.name, ''),
			COALESCE(cp.position, ''),
			COALESCE(cp.phone, ''),
			COALESCE(cp.email, ''),
			t.executor,
			TRIM(CONCAT(COALESCE(u_exec.first_name, ''), ' ', COALESCE(u_exec.last_name, ''))),
			COALESCE(d_exec.title, ''),
			t.reference_ticket,
			t.used_materials
		FROM tickets t
		LEFT JOIN ticket_reasons tr ON tr.id = t.reason
		LEFT JOIN ticket_types tt ON tt.type = t.ticket_type
		LEFT JOIN ticket_statuses ts ON ts.type = t.status
		LEFT JOIN clients c ON c.id = t.client
		LEFT JOIN devices d ON d.id = t.device
		LEFT JOIN classificators cls ON cls.id = d.classificator
		LEFT JOIN users u_exec ON u_exec.user_id = t.executor
		LEFT JOIN departments d_exec ON d_exec.id = u_exec.department
		LEFT JOIN users u_author ON u_author.user_id = t.author
		LEFT JOIN users u_assigned ON u_assigned.user_id = t.assigned_by
		LEFT JOIN departments dpt ON dpt.id = t.department
		LEFT JOIN contacts cp ON cp.id = t.contact_person
		WHERE t.id = $1
		  AND (
			t.executor = $2
			OR EXISTS (
				SELECT 1
				FROM users u_req
				WHERE u_req.user_id = $2
				  AND u_req.department IS NOT NULL
				  AND u_req.department = t.department
			)
		  )
	`, ticketID, userID)

	var (
		id                 pgtype.UUID
		number             pgtype.Int4
		status             string
		statusTitle        string
		description        string
		result             string
		reason             string
		resolvedReason     string
		ticketType         pgtype.Text
		ticketTypeTitle    string
		urgent             bool
		doubleSigned       bool
		createdAt          pgtype.Timestamp
		assignedAt         pgtype.Timestamp
		workstartedAt      pgtype.Timestamp
		workfinishedAt     pgtype.Timestamp
		plannedStart       pgtype.Timestamp
		plannedEnd         pgtype.Timestamp
		assignedStart      pgtype.Timestamp
		assignedEnd        pgtype.Timestamp
		closedAt           pgtype.Timestamp
		client             pgtype.UUID
		clientName         string
		clientAddress      string
		device             pgtype.UUID
		deviceName         string
		deviceSerialNumber string
		author             pgtype.UUID
		authorName         string
		department         pgtype.UUID
		departmentTitle    string
		assignedBy         pgtype.UUID
		assignedByName     string
		contactPerson      pgtype.UUID
		contactName        string
		contactPosition    string
		contactPhone       string
		contactEmail       string
		executor           pgtype.UUID
		executorName       string
		executorDepartment string
		referenceTicket    pgtype.UUID
		usedMaterials      []pgtype.UUID
	)

	if err := row.Scan(
		&id,
		&number,
		&status,
		&statusTitle,
		&description,
		&result,
		&reason,
		&resolvedReason,
		&ticketType,
		&ticketTypeTitle,
		&urgent,
		&doubleSigned,
		&createdAt,
		&assignedAt,
		&workstartedAt,
		&workfinishedAt,
		&plannedStart,
		&plannedEnd,
		&assignedStart,
		&assignedEnd,
		&closedAt,
		&client,
		&clientName,
		&clientAddress,
		&device,
		&deviceName,
		&deviceSerialNumber,
		&author,
		&authorName,
		&department,
		&departmentTitle,
		&assignedBy,
		&assignedByName,
		&contactPerson,
		&contactName,
		&contactPosition,
		&contactPhone,
		&contactEmail,
		&executor,
		&executorName,
		&executorDepartment,
		&referenceTicket,
		&usedMaterials,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "ticket not found", http.StatusNotFound)
			return
		}

		log.Printf("load ticket by id failed: %v", err)
		http.Error(w, "failed to load ticket", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ticketResponse{
		ID:                 uuidToString(id),
		Number:             number.Int32,
		Status:             status,
		StatusTitle:        statusTitle,
		Description:        description,
		Result:             result,
		Reason:             reason,
		ResolvedReason:     resolvedReason,
		TicketType:         nullableTextToString(ticketType),
		TicketTypeTitle:    ticketTypeTitle,
		Urgent:             urgent,
		DoubleSigned:       doubleSigned,
		CreatedAt:          timestampToRFC3339(createdAt),
		AssignedAt:         timestampToRFC3339(assignedAt),
		WorkstartedAt:      timestampToRFC3339(workstartedAt),
		WorkfinishedAt:     timestampToRFC3339(workfinishedAt),
		PlannedStart:       timestampToRFC3339(plannedStart),
		PlannedEnd:         timestampToRFC3339(plannedEnd),
		AssignedStart:      timestampToRFC3339(assignedStart),
		AssignedEnd:        timestampToRFC3339(assignedEnd),
		ClosedAt:           timestampToRFC3339(closedAt),
		Client:             nullableUUIDToString(client),
		ClientName:         clientName,
		ClientAddress:      clientAddress,
		Device:             nullableUUIDToString(device),
		DeviceName:         deviceName,
		DeviceSerialNumber: deviceSerialNumber,
		Author:             nullableUUIDToString(author),
		AuthorName:         authorName,
		Department:         nullableUUIDToString(department),
		DepartmentTitle:    departmentTitle,
		AssignedBy:         nullableUUIDToString(assignedBy),
		AssignedByName:     assignedByName,
		ContactPerson:      nullableUUIDToString(contactPerson),
		ContactName:        contactName,
		ContactPosition:    contactPosition,
		ContactPhone:       contactPhone,
		ContactEmail:       contactEmail,
		Executor:           nullableUUIDToString(executor),
		ExecutorName:       executorName,
		ExecutorDepartment: executorDepartment,
		ReferenceTicket:    nullableUUIDToString(referenceTicket),
		UsedMaterials:      uuidSliceToString(usedMaterials),
	})
}

func (s *Server) handleDepartmentTickets(w http.ResponseWriter, r *http.Request) {
	type ticketResponse struct {
		ID                 string  `json:"id"`
		Number             int32   `json:"number"`
		Status             string  `json:"status"`
		Description        string  `json:"description"`
		Reason             string  `json:"reason"`
		Urgent             bool    `json:"urgent"`
		Executor           *string `json:"executor"`
		ExecutorName       string  `json:"executorName"`
		ExecutorDepartment string  `json:"executorDepartment"`
		AssignedEnd        *string `json:"assigned_end"`
		WorkstartedAt      *string `json:"workstarted_at"`
		WorkfinishedAt     *string `json:"workfinished_at"`
		DeviceName         string  `json:"deviceName"`
		DeviceSerialNumber string  `json:"deviceSerialNumber"`
		ClientName         string  `json:"clientName"`
		ClientAddress      string  `json:"clientAddress"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT
			t.id,
			t.number,
			COALESCE(t.status, ''),
			t.description,
			COALESCE(
				NULLIF(
					CASE
						WHEN t.status = 'assigned' THEN tr.future
						WHEN t.status = 'worksDone' THEN tr.past
						ELSE tr.present
					END,
					''
				),
				NULLIF(tr.title, ''),
				'Не указано'
			) AS resolved_reason,
			t.urgent,
			t.executor,
			TRIM(CONCAT(COALESCE(u_exec.first_name, ''), ' ', COALESCE(u_exec.last_name, ''))),
			COALESCE(d_exec.title, ''),
			t.assigned_end,
			t.workstarted_at,
			t.workfinished_at,
			COALESCE(cls.title, ''),
			COALESCE(d.serial_number, ''),
			COALESCE(c.title, ''),
			COALESCE(c.address, '')
		FROM users u
		JOIN tickets t ON t.department = u.department
		LEFT JOIN clients c ON c.id = t.client
		LEFT JOIN devices d ON d.id = t.device
		LEFT JOIN classificators cls ON cls.id = d.classificator
		LEFT JOIN ticket_reasons tr ON tr.id = t.reason
		LEFT JOIN users u_exec ON u_exec.user_id = t.executor
		LEFT JOIN departments d_exec ON d_exec.id = u_exec.department
		WHERE u.user_id = $1
		  AND u.department IS NOT NULL
		ORDER BY t.assigned_end DESC NULLS LAST, t.created_at DESC, t.number DESC
	`, userID)
	if err != nil {
		log.Printf("query department tickets failed: %v", err)
		http.Error(w, "failed to load tickets", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tickets := make([]ticketResponse, 0)
	for rows.Next() {
		var (
			id             pgtype.UUID
			number         pgtype.Int4
			status         string
			description    string
			reason         string
			urgent         bool
			executor       pgtype.UUID
			executorName   string
			executorDept   string
			assignedEnd    pgtype.Timestamp
			workstartedAt  pgtype.Timestamp
			workfinishedAt pgtype.Timestamp
			deviceName     string
			deviceSerial   string
			clientName     string
			clientAddress  string
		)

		if err := rows.Scan(
			&id,
			&number,
			&status,
			&description,
			&reason,
			&urgent,
			&executor,
			&executorName,
			&executorDept,
			&assignedEnd,
			&workstartedAt,
			&workfinishedAt,
			&deviceName,
			&deviceSerial,
			&clientName,
			&clientAddress,
		); err != nil {
			log.Printf("scan department ticket failed: %v", err)
			http.Error(w, "failed to load tickets", http.StatusInternalServerError)
			return
		}

		tickets = append(tickets, ticketResponse{
			ID:                 uuidToString(id),
			Number:             number.Int32,
			Status:             status,
			Description:        description,
			Reason:             reason,
			Urgent:             urgent,
			Executor:           nullableUUIDToString(executor),
			ExecutorName:       executorName,
			ExecutorDepartment: executorDept,
			AssignedEnd:        timestampToRFC3339(assignedEnd),
			WorkstartedAt:      timestampToRFC3339(workstartedAt),
			WorkfinishedAt:     timestampToRFC3339(workfinishedAt),
			DeviceName:         deviceName,
			DeviceSerialNumber: deviceSerial,
			ClientName:         clientName,
			ClientAddress:      clientAddress,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate department tickets failed: %v", err)
		http.Error(w, "failed to load tickets", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tickets)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func hashPassword(password string) (string, error) {
	digest, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(digest), nil
}

func verifyPassword(password string, stored string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err == nil {
		return true
	}

	if !strings.HasPrefix(stored, "sha256$") {
		return false
	}

	return verifyLegacyPassword(password, stored)
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}

	return id.String()
}

func timestampToRFC3339(value pgtype.Timestamp) *string {
	if !value.Valid {
		return nil
	}

	formatted := value.Time.UTC().Format(time.RFC3339)
	return &formatted
}

func nullableUUIDToString(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}

	text := value.String()
	return &text
}

func nullableTextToString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	text := value.String
	return &text
}

func uuidSliceToString(values []pgtype.UUID) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if !value.Valid {
			continue
		}

		result = append(result, value.String())
	}

	return result
}
