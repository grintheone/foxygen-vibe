package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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

func (s *Server) canAccessReference(ctx context.Context, referenceID pgtype.UUID, userID pgtype.UUID) (bool, error) {
	var allowed bool

	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM tickets t
			WHERE (
				t.id = $1
				OR t.client = $1
				OR t.device = $1
				OR EXISTS (
					SELECT 1
					FROM agreements a
					WHERE a.id = $1
					  AND (a.actual_client = t.client OR a.device = t.device)
				)
			)
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
		)
	`, referenceID, userID).Scan(&allowed)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

func (s *Server) handleComments(w http.ResponseWriter, r *http.Request) {
	type commentResponse struct {
		ID          int32   `json:"id"`
		AuthorID    string  `json:"author_id"`
		AuthorName  string  `json:"authorName"`
		Department  string  `json:"department"`
		ReferenceID string  `json:"reference_id"`
		Text        string  `json:"text"`
		CreatedAt   *string `json:"created_at"`
	}

	type createCommentRequest struct {
		ReferenceID string `json:"reference_id"`
		Text        string `json:"text"`
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

	switch r.Method {
	case http.MethodGet:
		referenceIDValue := strings.TrimSpace(r.URL.Query().Get("reference_id"))
		if referenceIDValue == "" {
			http.Error(w, "reference_id is required", http.StatusBadRequest)
			return
		}

		var referenceID pgtype.UUID
		if err := referenceID.Scan(referenceIDValue); err != nil {
			http.Error(w, "invalid reference_id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		allowed, err := s.canAccessReference(ctx, referenceID, userID)
		if err != nil {
			log.Printf("check comments access failed: %v", err)
			http.Error(w, "failed to load comments", http.StatusInternalServerError)
			return
		}
		if !allowed {
			writeJSON(w, http.StatusOK, make([]commentResponse, 0))
			return
		}

		rows, err := s.db.Query(ctx, `
			SELECT
				c.id,
				c.author_id,
				TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))),
				COALESCE(d.title, ''),
				c.reference_id,
				COALESCE(c.text, ''),
				c.created_at
			FROM comments c
			LEFT JOIN users u ON u.user_id = c.author_id
			LEFT JOIN departments d ON d.id = u.department
			WHERE c.reference_id = $1
			ORDER BY c.created_at DESC, c.id DESC
		`, referenceID)
		if err != nil {
			log.Printf("query comments failed: %v", err)
			http.Error(w, "failed to load comments", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		comments := make([]commentResponse, 0)
		for rows.Next() {
			var (
				id         int32
				authorID   pgtype.UUID
				authorName string
				department string
				refID      pgtype.UUID
				text       string
				createdAt  pgtype.Timestamp
			)

			if err := rows.Scan(&id, &authorID, &authorName, &department, &refID, &text, &createdAt); err != nil {
				log.Printf("scan comment failed: %v", err)
				http.Error(w, "failed to load comments", http.StatusInternalServerError)
				return
			}

			comments = append(comments, commentResponse{
				ID:          id,
				AuthorID:    uuidToString(authorID),
				AuthorName:  authorName,
				Department:  department,
				ReferenceID: uuidToString(refID),
				Text:        text,
				CreatedAt:   timestampToRFC3339(createdAt),
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate comments failed: %v", err)
			http.Error(w, "failed to load comments", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, comments)
	case http.MethodPost:
		defer r.Body.Close()

		var input createCommentRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		input.ReferenceID = strings.TrimSpace(input.ReferenceID)
		input.Text = strings.TrimSpace(input.Text)

		if input.ReferenceID == "" {
			http.Error(w, "reference_id is required", http.StatusBadRequest)
			return
		}
		if input.Text == "" {
			http.Error(w, "text is required", http.StatusBadRequest)
			return
		}

		var referenceID pgtype.UUID
		if err := referenceID.Scan(input.ReferenceID); err != nil {
			http.Error(w, "invalid reference_id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		allowed, err := s.canAccessReference(ctx, referenceID, userID)
		if err != nil {
			log.Printf("check comments access failed: %v", err)
			http.Error(w, "failed to create comment", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "reference not found", http.StatusNotFound)
			return
		}

		row := s.db.QueryRow(ctx, `
			WITH inserted AS (
				INSERT INTO comments (author_id, reference_id, text, created_at)
				VALUES ($1, $2, $3, NOW())
				RETURNING id, author_id, reference_id, text, created_at
			)
			SELECT
				i.id,
				i.author_id,
				TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))),
				COALESCE(d.title, ''),
				i.reference_id,
				i.text,
				i.created_at
			FROM inserted i
			LEFT JOIN users u ON u.user_id = i.author_id
			LEFT JOIN departments d ON d.id = u.department
		`, userID, referenceID, input.Text)

		var (
			id         int32
			authorID   pgtype.UUID
			authorName string
			department string
			refID      pgtype.UUID
			text       string
			createdAt  pgtype.Timestamp
		)

		if err := row.Scan(&id, &authorID, &authorName, &department, &refID, &text, &createdAt); err != nil {
			log.Printf("insert comment failed: %v", err)
			http.Error(w, "failed to create comment", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, commentResponse{
			ID:          id,
			AuthorID:    uuidToString(authorID),
			AuthorName:  authorName,
			Department:  department,
			ReferenceID: uuidToString(refID),
			Text:        text,
			CreatedAt:   timestampToRFC3339(createdAt),
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleClientByID(w http.ResponseWriter, r *http.Request) {
	type clientResponse struct {
		ID               string          `json:"id"`
		Title            string          `json:"title"`
		Address          string          `json:"address"`
		Location         json.RawMessage `json:"location"`
		Region           *string         `json:"region"`
		LaboratorySystem *string         `json:"laboratorySystem"`
		Manager          []string        `json:"manager"`
	}

	type clientTicketResponse struct {
		ID                 string  `json:"id"`
		Number             int32   `json:"number"`
		Status             string  `json:"status"`
		Description        string  `json:"description"`
		Result             string  `json:"result"`
		Reason             string  `json:"reason"`
		Urgent             bool    `json:"urgent"`
		Executor           *string `json:"executor"`
		ExecutorName       string  `json:"executorName"`
		ExecutorDepartment string  `json:"executorDepartment"`
		AssignedEnd        *string `json:"assigned_end"`
		WorkstartedAt      *string `json:"workstarted_at"`
		WorkfinishedAt     *string `json:"workfinished_at"`
		ClosedAt           *string `json:"closed_at"`
		DeviceName         string  `json:"deviceName"`
		DeviceSerialNumber string  `json:"deviceSerialNumber"`
		ClientName         string  `json:"clientName"`
		ClientAddress      string  `json:"clientAddress"`
	}

	type clientContactResponse struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Position string `json:"position"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
	}

	type clientAgreementResponse struct {
		ID                 string  `json:"id"`
		Number             int32   `json:"number"`
		Device             *string `json:"device"`
		DeviceName         string  `json:"deviceName"`
		DeviceSerialNumber string  `json:"deviceSerialNumber"`
		AssignedAt         *string `json:"assigned_at"`
		FinishedAt         *string `json:"finished_at"`
		IsActive           bool    `json:"isActive"`
		OnWarranty         bool    `json:"onWarranty"`
		Type               *string `json:"type"`
	}

	if _, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization")); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	clientPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/clients/"), "/")
	if clientPath == "" {
		http.NotFound(w, r)
		return
	}
	pathParts := strings.Split(clientPath, "/")

	var clientID pgtype.UUID
	if err := clientID.Scan(pathParts[0]); err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	switch {
	case len(pathParts) == 1:
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case len(pathParts) == 2 && pathParts[1] == "tickets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := strings.TrimSpace(r.URL.Query().Get("status"))
		limit := 50
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := s.db.Query(ctx, `
			SELECT
				t.id,
				t.number,
				COALESCE(t.status, ''),
				t.description,
				COALESCE(t.result, ''),
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
				t.closed_at,
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
			WHERE t.client = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			ORDER BY t.closed_at DESC NULLS LAST, t.workfinished_at DESC NULLS LAST, t.created_at DESC, t.number DESC
			LIMIT $3
		`, clientID, status, limit)
		if err != nil {
			log.Printf("query client tickets failed: %v", err)
			http.Error(w, "failed to load client tickets", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tickets := make([]clientTicketResponse, 0)
		for rows.Next() {
			var (
				id             pgtype.UUID
				number         pgtype.Int4
				ticketStatus   string
				description    string
				result         string
				reason         string
				urgent         bool
				executor       pgtype.UUID
				executorName   string
				executorDept   string
				assignedEnd    pgtype.Timestamp
				workstartedAt  pgtype.Timestamp
				workfinishedAt pgtype.Timestamp
				closedAt       pgtype.Timestamp
				deviceName     string
				deviceSerial   string
				clientName     string
				clientAddress  string
			)

			if err := rows.Scan(
				&id,
				&number,
				&ticketStatus,
				&description,
				&result,
				&reason,
				&urgent,
				&executor,
				&executorName,
				&executorDept,
				&assignedEnd,
				&workstartedAt,
				&workfinishedAt,
				&closedAt,
				&deviceName,
				&deviceSerial,
				&clientName,
				&clientAddress,
			); err != nil {
				log.Printf("scan client ticket failed: %v", err)
				http.Error(w, "failed to load client tickets", http.StatusInternalServerError)
				return
			}

			tickets = append(tickets, clientTicketResponse{
				ID:                 uuidToString(id),
				Number:             number.Int32,
				Status:             ticketStatus,
				Description:        description,
				Result:             result,
				Reason:             reason,
				Urgent:             urgent,
				Executor:           nullableUUIDToString(executor),
				ExecutorName:       executorName,
				ExecutorDepartment: executorDept,
				AssignedEnd:        timestampToRFC3339(assignedEnd),
				WorkstartedAt:      timestampToRFC3339(workstartedAt),
				WorkfinishedAt:     timestampToRFC3339(workfinishedAt),
				ClosedAt:           timestampToRFC3339(closedAt),
				DeviceName:         deviceName,
				DeviceSerialNumber: deviceSerial,
				ClientName:         clientName,
				ClientAddress:      clientAddress,
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate client tickets failed: %v", err)
			http.Error(w, "failed to load client tickets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, tickets)
		return
	case len(pathParts) == 2 && pathParts[1] == "contacts":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := 100
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := s.db.Query(ctx, `
			SELECT
				ct.id,
				COALESCE(ct.name, ''),
				COALESCE(ct.position, ''),
				COALESCE(ct.phone, ''),
				COALESCE(ct.email, '')
			FROM contacts ct
			WHERE ct.client_id = $1
			ORDER BY ct.name ASC, ct.id ASC
			LIMIT $2
		`, clientID, limit)
		if err != nil {
			log.Printf("query client contacts failed: %v", err)
			http.Error(w, "failed to load client contacts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		contacts := make([]clientContactResponse, 0)
		for rows.Next() {
			var (
				id       pgtype.UUID
				name     string
				position string
				phone    string
				email    string
			)

			if err := rows.Scan(&id, &name, &position, &phone, &email); err != nil {
				log.Printf("scan client contact failed: %v", err)
				http.Error(w, "failed to load client contacts", http.StatusInternalServerError)
				return
			}

			contacts = append(contacts, clientContactResponse{
				ID:       uuidToString(id),
				Name:     name,
				Position: position,
				Phone:    phone,
				Email:    email,
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate client contacts failed: %v", err)
			http.Error(w, "failed to load client contacts", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, contacts)
		return
	case len(pathParts) == 2 && pathParts[1] == "agreements":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := 100
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := s.db.Query(ctx, `
			SELECT
				a.id,
				a.number,
				a.device,
				COALESCE(cls.title, ''),
				COALESCE(d.serial_number, ''),
				a.assigned_at,
				a.finished_at,
				a.is_active,
				a.on_warranty,
				a.type
			FROM agreements a
			LEFT JOIN devices d ON d.id = a.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			WHERE a.actual_client = $1
			ORDER BY a.is_active DESC, a.assigned_at DESC NULLS LAST, a.number DESC
			LIMIT $2
		`, clientID, limit)
		if err != nil {
			log.Printf("query client agreements failed: %v", err)
			http.Error(w, "failed to load client agreements", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		agreements := make([]clientAgreementResponse, 0)
		for rows.Next() {
			var (
				id            pgtype.UUID
				number        pgtype.Int4
				device        pgtype.UUID
				deviceName    string
				deviceSerial  string
				assignedAt    pgtype.Timestamp
				finishedAt    pgtype.Timestamp
				isActive      bool
				onWarranty    bool
				agreementType pgtype.Text
			)

			if err := rows.Scan(
				&id,
				&number,
				&device,
				&deviceName,
				&deviceSerial,
				&assignedAt,
				&finishedAt,
				&isActive,
				&onWarranty,
				&agreementType,
			); err != nil {
				log.Printf("scan client agreement failed: %v", err)
				http.Error(w, "failed to load client agreements", http.StatusInternalServerError)
				return
			}

			agreements = append(agreements, clientAgreementResponse{
				ID:                 uuidToString(id),
				Number:             number.Int32,
				Device:             nullableUUIDToString(device),
				DeviceName:         deviceName,
				DeviceSerialNumber: deviceSerial,
				AssignedAt:         timestampToRFC3339(assignedAt),
				FinishedAt:         timestampToRFC3339(finishedAt),
				IsActive:           isActive,
				OnWarranty:         onWarranty,
				Type:               nullableTextToString(agreementType),
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate client agreements failed: %v", err)
			http.Error(w, "failed to load client agreements", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, agreements)
		return
	default:
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	row := s.db.QueryRow(ctx, `
		SELECT
			c.id,
			COALESCE(c.title, ''),
			COALESCE(c.address, ''),
			c.location,
			c.region,
			c.laboratory_system,
			c.manager
		FROM clients c
		WHERE c.id = $1
		LIMIT 1
	`, clientID)

	var (
		id               pgtype.UUID
		title            string
		address          string
		location         []byte
		region           pgtype.UUID
		laboratorySystem pgtype.UUID
		manager          []pgtype.UUID
	)

	if err := row.Scan(
		&id,
		&title,
		&address,
		&location,
		&region,
		&laboratorySystem,
		&manager,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}

		log.Printf("load client by id failed: %v", err)
		http.Error(w, "failed to load client", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, clientResponse{
		ID:               uuidToString(id),
		Title:            title,
		Address:          address,
		Location:         json.RawMessage(location),
		Region:           nullableUUIDToString(region),
		LaboratorySystem: nullableUUIDToString(laboratorySystem),
		Manager:          uuidSliceToString(manager),
	})
}

func (s *Server) handleDeviceByID(w http.ResponseWriter, r *http.Request) {
	type deviceResponse struct {
		ID                string          `json:"id"`
		Title             string          `json:"title"`
		SerialNumber      string          `json:"serialNumber"`
		Properties        json.RawMessage `json:"properties"`
		ConnectedToLis    bool            `json:"connectedToLis"`
		IsUsed            bool            `json:"isUsed"`
		Client            *string         `json:"client"`
		ClientName        string          `json:"clientName"`
		ClientAddress     string          `json:"clientAddress"`
		Agreement         *string         `json:"agreement"`
		AgreementNumber   *int32          `json:"agreementNumber"`
		AgreementType     *string         `json:"agreementType"`
		IsActiveAgreement bool            `json:"isActiveAgreement"`
		OnWarranty        bool            `json:"onWarranty"`
	}

	type deviceTicketResponse struct {
		ID                 string  `json:"id"`
		Number             int32   `json:"number"`
		Status             string  `json:"status"`
		Description        string  `json:"description"`
		Result             string  `json:"result"`
		Reason             string  `json:"reason"`
		Urgent             bool    `json:"urgent"`
		Executor           *string `json:"executor"`
		ExecutorName       string  `json:"executorName"`
		ExecutorDepartment string  `json:"executorDepartment"`
		AssignedEnd        *string `json:"assigned_end"`
		WorkstartedAt      *string `json:"workstarted_at"`
		WorkfinishedAt     *string `json:"workfinished_at"`
		ClosedAt           *string `json:"closed_at"`
		DeviceName         string  `json:"deviceName"`
		DeviceSerialNumber string  `json:"deviceSerialNumber"`
		ClientName         string  `json:"clientName"`
		ClientAddress      string  `json:"clientAddress"`
	}

	type deviceAgreementResponse struct {
		ID            string  `json:"id"`
		Number        int32   `json:"number"`
		Client        *string `json:"client"`
		ClientName    string  `json:"clientName"`
		ClientAddress string  `json:"clientAddress"`
		AssignedAt    *string `json:"assigned_at"`
		FinishedAt    *string `json:"finished_at"`
		IsActive      bool    `json:"isActive"`
		OnWarranty    bool    `json:"onWarranty"`
		Type          *string `json:"type"`
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

	devicePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/devices/"), "/")
	if devicePath == "" {
		http.NotFound(w, r)
		return
	}
	pathParts := strings.Split(devicePath, "/")

	var deviceID pgtype.UUID
	if err := deviceID.Scan(pathParts[0]); err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	switch {
	case len(pathParts) == 1:
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case len(pathParts) == 2 && pathParts[1] == "tickets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := strings.TrimSpace(r.URL.Query().Get("status"))
		limit := 50
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsedLimit, parseErr := strconv.Atoi(rawLimit)
			if parseErr != nil || parsedLimit <= 0 {
				http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
				return
			}
			if parsedLimit > 100 {
				parsedLimit = 100
			}
			limit = parsedLimit
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := s.db.Query(ctx, `
			SELECT
				t.id,
				t.number,
				COALESCE(t.status, ''),
				t.description,
				COALESCE(t.result, ''),
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
				t.closed_at,
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
			WHERE t.device = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND (
				t.executor = $3
				OR EXISTS (
					SELECT 1
					FROM users u_req
					WHERE u_req.user_id = $3
					  AND u_req.department IS NOT NULL
					  AND u_req.department = t.department
				)
			  )
			ORDER BY t.closed_at DESC NULLS LAST, t.workfinished_at DESC NULLS LAST, t.created_at DESC, t.number DESC
			LIMIT $4
		`, deviceID, status, userID, limit)
		if err != nil {
			log.Printf("query device tickets failed: %v", err)
			http.Error(w, "failed to load device tickets", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tickets := make([]deviceTicketResponse, 0)
		for rows.Next() {
			var (
				id             pgtype.UUID
				number         pgtype.Int4
				ticketStatus   string
				description    string
				result         string
				reason         string
				urgent         bool
				executor       pgtype.UUID
				executorName   string
				executorDept   string
				assignedEnd    pgtype.Timestamp
				workstartedAt  pgtype.Timestamp
				workfinishedAt pgtype.Timestamp
				closedAt       pgtype.Timestamp
				deviceName     string
				deviceSerial   string
				clientName     string
				clientAddress  string
			)

			if err := rows.Scan(
				&id,
				&number,
				&ticketStatus,
				&description,
				&result,
				&reason,
				&urgent,
				&executor,
				&executorName,
				&executorDept,
				&assignedEnd,
				&workstartedAt,
				&workfinishedAt,
				&closedAt,
				&deviceName,
				&deviceSerial,
				&clientName,
				&clientAddress,
			); err != nil {
				log.Printf("scan device ticket failed: %v", err)
				http.Error(w, "failed to load device tickets", http.StatusInternalServerError)
				return
			}

			tickets = append(tickets, deviceTicketResponse{
				ID:                 uuidToString(id),
				Number:             number.Int32,
				Status:             ticketStatus,
				Description:        description,
				Result:             result,
				Reason:             reason,
				Urgent:             urgent,
				Executor:           nullableUUIDToString(executor),
				ExecutorName:       executorName,
				ExecutorDepartment: executorDept,
				AssignedEnd:        timestampToRFC3339(assignedEnd),
				WorkstartedAt:      timestampToRFC3339(workstartedAt),
				WorkfinishedAt:     timestampToRFC3339(workfinishedAt),
				ClosedAt:           timestampToRFC3339(closedAt),
				DeviceName:         deviceName,
				DeviceSerialNumber: deviceSerial,
				ClientName:         clientName,
				ClientAddress:      clientAddress,
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate device tickets failed: %v", err)
			http.Error(w, "failed to load device tickets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, tickets)
		return
	case len(pathParts) == 2 && pathParts[1] == "agreements":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		activeOnly := true
		if rawActive := strings.TrimSpace(r.URL.Query().Get("active")); rawActive != "" {
			switch strings.ToLower(rawActive) {
			case "true", "1", "yes":
				activeOnly = true
			case "false", "0", "no":
				activeOnly = false
			default:
				http.Error(w, "active must be a boolean", http.StatusBadRequest)
				return
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := s.db.Query(ctx, `
			SELECT
				a.id,
				a.number,
				a.actual_client,
				COALESCE(c.title, ''),
				COALESCE(c.address, ''),
				a.assigned_at,
				a.finished_at,
				a.is_active,
				a.on_warranty,
				a.type
			FROM agreements a
			LEFT JOIN clients c ON c.id = a.actual_client
			WHERE a.device = $1
			  AND ($2 = FALSE OR a.is_active = TRUE)
			  AND EXISTS (
				SELECT 1
				FROM tickets t
				WHERE t.device = $1
				  AND (
					t.executor = $3
					OR EXISTS (
						SELECT 1
						FROM users u_req
						WHERE u_req.user_id = $3
						  AND u_req.department IS NOT NULL
						  AND u_req.department = t.department
					)
				  )
			  )
			ORDER BY a.assigned_at DESC NULLS LAST, a.number DESC
		`, deviceID, activeOnly, userID)
		if err != nil {
			log.Printf("query device agreements failed: %v", err)
			http.Error(w, "failed to load device agreements", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		agreements := make([]deviceAgreementResponse, 0)
		for rows.Next() {
			var (
				id            pgtype.UUID
				number        pgtype.Int4
				client        pgtype.UUID
				clientName    string
				clientAddress string
				assignedAt    pgtype.Timestamp
				finishedAt    pgtype.Timestamp
				isActive      bool
				onWarranty    bool
				agreementType pgtype.Text
			)

			if err := rows.Scan(
				&id,
				&number,
				&client,
				&clientName,
				&clientAddress,
				&assignedAt,
				&finishedAt,
				&isActive,
				&onWarranty,
				&agreementType,
			); err != nil {
				log.Printf("scan device agreement failed: %v", err)
				http.Error(w, "failed to load device agreements", http.StatusInternalServerError)
				return
			}

			agreements = append(agreements, deviceAgreementResponse{
				ID:            uuidToString(id),
				Number:        number.Int32,
				Client:        nullableUUIDToString(client),
				ClientName:    clientName,
				ClientAddress: clientAddress,
				AssignedAt:    timestampToRFC3339(assignedAt),
				FinishedAt:    timestampToRFC3339(finishedAt),
				IsActive:      isActive,
				OnWarranty:    onWarranty,
				Type:          nullableTextToString(agreementType),
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("iterate device agreements failed: %v", err)
			http.Error(w, "failed to load device agreements", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, agreements)
		return
	default:
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	row := s.db.QueryRow(ctx, `
		SELECT
			d.id,
			COALESCE(cls.title, ''),
			COALESCE(d.serial_number, ''),
			d.properties,
			d.connected_to_lis,
			d.is_used,
			a.id,
			a.number,
			COALESCE(a.is_active, FALSE),
			COALESCE(a.on_warranty, FALSE),
			a.type,
			c.id,
			COALESCE(c.title, ''),
			COALESCE(c.address, '')
		FROM devices d
		LEFT JOIN classificators cls ON cls.id = d.classificator
		LEFT JOIN LATERAL (
			SELECT
				a.id,
				a.number,
				a.is_active,
				a.on_warranty,
				a.type,
				a.actual_client
			FROM agreements a
			WHERE a.device = d.id
			ORDER BY a.is_active DESC, a.assigned_at DESC NULLS LAST, a.number DESC
			LIMIT 1
		) a ON TRUE
		LEFT JOIN clients c ON c.id = a.actual_client
		WHERE d.id = $1
		LIMIT 1
	`, deviceID)

	var (
		id                pgtype.UUID
		title             string
		serialNumber      string
		properties        []byte
		connectedToLis    bool
		isUsed            bool
		agreementID       pgtype.UUID
		agreementNumber   pgtype.Int4
		isActiveAgreement bool
		onWarranty        bool
		agreementType     pgtype.Text
		clientID          pgtype.UUID
		clientName        string
		clientAddress     string
	)

	if err := row.Scan(
		&id,
		&title,
		&serialNumber,
		&properties,
		&connectedToLis,
		&isUsed,
		&agreementID,
		&agreementNumber,
		&isActiveAgreement,
		&onWarranty,
		&agreementType,
		&clientID,
		&clientName,
		&clientAddress,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		log.Printf("load device by id failed: %v", err)
		http.Error(w, "failed to load device", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, deviceResponse{
		ID:                uuidToString(id),
		Title:             title,
		SerialNumber:      serialNumber,
		Properties:        json.RawMessage(properties),
		ConnectedToLis:    connectedToLis,
		IsUsed:            isUsed,
		Client:            nullableUUIDToString(clientID),
		ClientName:        clientName,
		ClientAddress:     clientAddress,
		Agreement:         nullableUUIDToString(agreementID),
		AgreementNumber:   nullableInt4ToInt32(agreementNumber),
		AgreementType:     nullableTextToString(agreementType),
		IsActiveAgreement: isActiveAgreement,
		OnWarranty:        onWarranty,
	})
}

func (s *Server) handleDepartments(w http.ResponseWriter, r *http.Request) {
	type departmentResponse struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization")); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(title, '')
		FROM departments
		ORDER BY title ASC, id ASC
	`)
	if err != nil {
		log.Printf("query departments failed: %v", err)
		http.Error(w, "failed to load departments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]departmentResponse, 0)
	for rows.Next() {
		var (
			id    pgtype.UUID
			title string
		)

		if err := rows.Scan(&id, &title); err != nil {
			log.Printf("scan department failed: %v", err)
			http.Error(w, "failed to load departments", http.StatusInternalServerError)
			return
		}

		response = append(response, departmentResponse{
			ID:    uuidToString(id),
			Title: title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate departments failed: %v", err)
		http.Error(w, "failed to load departments", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
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
		ID                 string                     `json:"id"`
		Attachments        []ticketAttachmentResponse `json:"attachments"`
		Number             int32                      `json:"number"`
		Status             string                     `json:"status"`
		StatusTitle        string                     `json:"statusTitle"`
		Description        string                     `json:"description"`
		Result             string                     `json:"result"`
		Reason             string                     `json:"reason"`
		ResolvedReason     string                     `json:"resolvedReason"`
		TicketType         *string                    `json:"ticketType"`
		TicketTypeTitle    string                     `json:"ticketTypeTitle"`
		Urgent             bool                       `json:"urgent"`
		DoubleSigned       bool                       `json:"doubleSigned"`
		CreatedAt          *string                    `json:"created_at"`
		AssignedAt         *string                    `json:"assigned_at"`
		WorkstartedAt      *string                    `json:"workstarted_at"`
		WorkfinishedAt     *string                    `json:"workfinished_at"`
		PlannedStart       *string                    `json:"planned_start"`
		PlannedEnd         *string                    `json:"planned_end"`
		AssignedStart      *string                    `json:"assigned_start"`
		AssignedEnd        *string                    `json:"assigned_end"`
		ClosedAt           *string                    `json:"closed_at"`
		Client             *string                    `json:"client"`
		ClientName         string                     `json:"clientName"`
		ClientAddress      string                     `json:"clientAddress"`
		Device             *string                    `json:"device"`
		DeviceName         string                     `json:"deviceName"`
		DeviceSerialNumber string                     `json:"deviceSerialNumber"`
		Author             *string                    `json:"author"`
		AuthorName         string                     `json:"authorName"`
		Department         *string                    `json:"department"`
		DepartmentTitle    string                     `json:"departmentTitle"`
		AssignedBy         *string                    `json:"assignedBy"`
		AssignedByName     string                     `json:"assignedByName"`
		ContactPerson      *string                    `json:"contactPerson"`
		ContactName        string                     `json:"contactName"`
		ContactPosition    string                     `json:"contactPosition"`
		ContactPhone       string                     `json:"contactPhone"`
		ContactEmail       string                     `json:"contactEmail"`
		Executor           *string                    `json:"executor"`
		ExecutorName       string                     `json:"executorName"`
		ExecutorDepartment string                     `json:"executorDepartment"`
		ReferenceTicket    *string                    `json:"referenceTicket"`
		UsedMaterials      []string                   `json:"usedMaterials"`
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

	ticketPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/tickets/"), "/")
	if ticketPath == "" {
		http.NotFound(w, r)
		return
	}
	pathParts := strings.Split(ticketPath, "/")

	var ticketID pgtype.UUID
	if err := ticketID.Scan(pathParts[0]); err != nil {
		http.Error(w, "invalid ticket id", http.StatusBadRequest)
		return
	}

	var userID pgtype.UUID
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	switch {
	case len(pathParts) == 1:
		if r.Method == http.MethodPatch {
			s.handleTicketByIDPatch(w, r, ticketID, userID)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	case len(pathParts) == 2 && pathParts[1] == "attachments":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s.handleTicketAttachmentUpload(w, r, ticketID, userID)
		return
	case len(pathParts) == 4 && pathParts[1] == "attachments" && pathParts[3] == "download":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s.handleTicketAttachmentDownload(w, r, ticketID, pathParts[2], userID)
		return
	default:
		http.NotFound(w, r)
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

	attachments, err := s.loadTicketAttachments(ctx, ticketID)
	if err != nil {
		log.Printf("load ticket attachments failed: %v", err)
		http.Error(w, "failed to load ticket", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, ticketResponse{
		ID:                 uuidToString(id),
		Attachments:        attachments,
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

func (s *Server) handleTicketByIDPatch(w http.ResponseWriter, r *http.Request, ticketID pgtype.UUID, userID pgtype.UUID) {
	type patchTicketAttachmentRequest struct {
		ClientID  string `json:"client_id"`
		Ext       string `json:"ext"`
		MediaType string `json:"media_type"`
		Name      string `json:"name"`
	}

	type patchTicketAttachmentResponse struct {
		ClientID string `json:"client_id"`
		ID       string `json:"id"`
		Status   string `json:"status"`
	}

	type patchTicketRequest struct {
		Attachments              []patchTicketAttachmentRequest `json:"attachments"`
		ClosedAt                 string                         `json:"closed_at"`
		DoubleSigned             bool                           `json:"double_signed"`
		Recommendation           string                         `json:"recommendation"`
		RecommendationDepartment string                         `json:"recommendation_department"`
		Result                   string                         `json:"result"`
		Status                   string                         `json:"status"`
		WorkstartedAt            string                         `json:"workstarted_at"`
		WorkfinishedAt           string                         `json:"workfinished_at"`
	}

	type patchTicketFollowUpResponse struct {
		ID     string `json:"id"`
		Number int32  `json:"number"`
		Status string `json:"status"`
	}

	type patchTicketResponse struct {
		Attachments    []patchTicketAttachmentResponse `json:"attachments,omitempty"`
		ClosedAt       *string                         `json:"closed_at,omitempty"`
		FollowUpTicket *patchTicketFollowUpResponse    `json:"followUpTicket,omitempty"`
		ID             string                          `json:"id"`
		Status         string                          `json:"status"`
		WorkfinishedAt *string                         `json:"workfinished_at,omitempty"`
		WorkstartedAt  *string                         `json:"workstarted_at,omitempty"`
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	defer r.Body.Close()

	var input patchTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Status = strings.TrimSpace(input.Status)
	input.WorkstartedAt = strings.TrimSpace(input.WorkstartedAt)
	input.WorkfinishedAt = strings.TrimSpace(input.WorkfinishedAt)
	input.ClosedAt = strings.TrimSpace(input.ClosedAt)
	input.Result = strings.TrimSpace(input.Result)
	input.Recommendation = strings.TrimSpace(input.Recommendation)
	input.RecommendationDepartment = strings.TrimSpace(input.RecommendationDepartment)

	for index := range input.Attachments {
		input.Attachments[index].ClientID = strings.TrimSpace(input.Attachments[index].ClientID)
		input.Attachments[index].Name = strings.TrimSpace(input.Attachments[index].Name)
		input.Attachments[index].MediaType = strings.TrimSpace(input.Attachments[index].MediaType)
		input.Attachments[index].Ext = strings.TrimSpace(input.Attachments[index].Ext)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var (
		err                   error
		result                pgconn.CommandTag
		expectedCurrentStatus string
		response              patchTicketResponse
		tx                    pgx.Tx
	)

	response.ID = ticketID.String()
	response.Status = input.Status

	switch input.Status {
	case "inWork":
		if input.WorkstartedAt == "" {
			http.Error(w, "workstarted_at is required", http.StatusBadRequest)
			return
		}

		workstartedAt, parseErr := time.Parse(time.RFC3339Nano, input.WorkstartedAt)
		if parseErr != nil {
			http.Error(w, "workstarted_at must be an ISO timestamp", http.StatusBadRequest)
			return
		}

		expectedCurrentStatus = "assigned"
		result, err = s.db.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				workstarted_at = $2
			WHERE id = $3
			  AND executor = $4
			  AND status = $5
		`, input.Status, workstartedAt.UTC(), ticketID, userID, expectedCurrentStatus)
		if err == nil {
			formatted := workstartedAt.UTC().Format(time.RFC3339)
			response.WorkstartedAt = &formatted
		}
	case "worksDone":
		if input.WorkfinishedAt == "" {
			http.Error(w, "workfinished_at is required", http.StatusBadRequest)
			return
		}

		workfinishedAt, parseErr := time.Parse(time.RFC3339Nano, input.WorkfinishedAt)
		if parseErr != nil {
			http.Error(w, "workfinished_at must be an ISO timestamp", http.StatusBadRequest)
			return
		}

		expectedCurrentStatus = "inWork"
		result, err = s.db.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				workfinished_at = $2
			WHERE id = $3
			  AND executor = $4
			  AND status = $5
		`, input.Status, workfinishedAt.UTC(), ticketID, userID, expectedCurrentStatus)
		if err == nil {
			formatted := workfinishedAt.UTC().Format(time.RFC3339)
			response.WorkfinishedAt = &formatted
		}
	case "closed":
		if input.ClosedAt == "" {
			http.Error(w, "closed_at is required", http.StatusBadRequest)
			return
		}
		if input.Result == "" {
			http.Error(w, "result is required", http.StatusBadRequest)
			return
		}
		if (input.Recommendation == "") != (input.RecommendationDepartment == "") {
			http.Error(w, "recommendation and recommendation_department must be provided together", http.StatusBadRequest)
			return
		}

		closedAt, parseErr := time.Parse(time.RFC3339Nano, input.ClosedAt)
		if parseErr != nil {
			http.Error(w, "closed_at must be an ISO timestamp", http.StatusBadRequest)
			return
		}

		tx, err = s.db.Begin(ctx)
		if err != nil {
			log.Printf("begin patch ticket transaction failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		expectedCurrentStatus = "worksDone"
		result, err = tx.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				closed_at = $2,
				result = $3,
				double_signed = $4
			WHERE id = $5
			  AND executor = $6
			  AND status = $7
		`, input.Status, closedAt.UTC(), input.Result, input.DoubleSigned, ticketID, userID, expectedCurrentStatus)
		if err != nil {
			break
		}

		if err == nil {
			formatted := closedAt.UTC().Format(time.RFC3339)
			response.ClosedAt = &formatted
		}

		if err != nil || result.RowsAffected() == 0 || input.Recommendation == "" {
			break
		}

		var recommendationDepartmentID pgtype.UUID
		if scanErr := recommendationDepartmentID.Scan(input.RecommendationDepartment); scanErr != nil {
			http.Error(w, "recommendation_department must be a valid UUID", http.StatusBadRequest)
			return
		}

		var departmentExists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`, recommendationDepartmentID).Scan(&departmentExists); err != nil {
			break
		}
		if !departmentExists {
			http.Error(w, "recommendation_department not found", http.StatusBadRequest)
			return
		}

		var (
			followUpID     pgtype.UUID
			followUpNumber int32
			followUpStatus string
		)
		err = tx.QueryRow(ctx, `
			INSERT INTO tickets (
				client,
				device,
				ticket_type,
				author,
				department,
				reason,
				description,
				contact_person,
				status,
				urgent,
				reference_ticket
			)
			SELECT
				client,
				device,
				ticket_type,
				author,
				$1,
				reason,
				$2,
				contact_person,
				'created',
				urgent,
				id
			FROM tickets
			WHERE id = $3
			RETURNING id, number, COALESCE(status, '')
		`, recommendationDepartmentID, input.Recommendation, ticketID).Scan(&followUpID, &followUpNumber, &followUpStatus)
		if err == nil {
			response.FollowUpTicket = &patchTicketFollowUpResponse{
				ID:     uuidToString(followUpID),
				Number: followUpNumber,
				Status: followUpStatus,
			}
		}
	default:
		http.Error(w, "supported transitions: assigned->inWork, inWork->worksDone, worksDone->closed", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("patch ticket failed: %v", err)
		http.Error(w, "failed to update ticket", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		var ticketExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tickets WHERE id = $1)`, ticketID).Scan(&ticketExists); err != nil {
			log.Printf("check ticket existence failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}

		if !ticketExists {
			http.Error(w, "ticket not found", http.StatusNotFound)
			return
		}

		http.Error(w, "ticket must be assigned to you and in expected status", http.StatusConflict)
		return
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			log.Printf("commit patch ticket transaction failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
	}

	writeJSON(w, http.StatusOK, response)
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

func timestamptzToRFC3339(value pgtype.Timestamptz) *string {
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

func nullableInt4ToInt32(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}

	number := value.Int32
	return &number
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
