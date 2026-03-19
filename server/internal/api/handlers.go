package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"foxygen-vibe/server/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultTicketListLimit     = 50
	defaultTicketSyncSource    = "external-sync"
	maxTicketListLimit         = 100
	maxProfileAvatarUploadSize = 10 << 20
)

var supportedProfileAvatarMediaTypes = map[string]struct{}{
	"image/gif":  {},
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

type paginatedResponse[T any] struct {
	Items   []T  `json:"items"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasNext bool `json:"hasNext"`
	HasPrev bool `json:"hasPrev"`
}

type ticketListFilters struct {
	DeviceName  string
	EndDate     *time.Time
	ReasonTitle string
	SortBy      string
	StartDate   *time.Time
	Status      string
}

type ticketArchiveFacetsResponse struct {
	DeviceNames  []string `json:"deviceNames"`
	ReasonTitles []string `json:"reasonTitles"`
}

type profileTicketResponse struct {
	ID                 string  `json:"id"`
	Number             int32   `json:"number"`
	Status             string  `json:"status"`
	Description        string  `json:"description"`
	Result             string  `json:"result"`
	Reason             string  `json:"reason"`
	ReasonTitle        string  `json:"reasonTitle"`
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

type profileTicketStatsResponse struct {
	Total           int `json:"total"`
	Closed          int `json:"closed"`
	Overdue         int `json:"overdue"`
	ClosedThisMonth int `json:"closedThisMonth"`
}

func parseTicketListPagination(r *http.Request) (int, int, error) {
	limit := defaultTicketListLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil || parsedLimit <= 0 {
			return 0, 0, errors.New("limit must be a positive integer")
		}
		if parsedLimit > maxTicketListLimit {
			parsedLimit = maxTicketListLimit
		}
		limit = parsedLimit
	}

	offset := 0
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		parsedOffset, parseErr := strconv.Atoi(rawOffset)
		if parseErr != nil || parsedOffset < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
		offset = parsedOffset
	}

	return limit, offset, nil
}

func newPaginatedResponse[T any](items []T, limit, offset, total int) paginatedResponse[T] {
	return paginatedResponse[T]{
		Items:   items,
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		HasNext: offset+len(items) < total,
		HasPrev: offset > 0,
	}
}

func parseTicketListFilters(r *http.Request) (ticketListFilters, error) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	reasonTitle := strings.TrimSpace(r.URL.Query().Get("reasonTitle"))
	deviceName := strings.TrimSpace(r.URL.Query().Get("deviceName"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sortBy"))
	if sortBy == "" {
		sortBy = "newest"
	}
	if sortBy != "newest" && sortBy != "oldest" {
		return ticketListFilters{}, errors.New("sortBy must be newest or oldest")
	}

	startDate, err := parseArchiveDate(strings.TrimSpace(r.URL.Query().Get("startDate")))
	if err != nil {
		return ticketListFilters{}, err
	}

	endDate, err := parseArchiveDate(strings.TrimSpace(r.URL.Query().Get("endDate")))
	if err != nil {
		return ticketListFilters{}, err
	}
	if endDate != nil {
		endDateValue := endDate.Add(24 * time.Hour)
		endDate = &endDateValue
	}

	return ticketListFilters{
		DeviceName:  deviceName,
		EndDate:     endDate,
		ReasonTitle: reasonTitle,
		SortBy:      sortBy,
		StartDate:   startDate,
		Status:      status,
	}, nil
}

func parseArchiveDate(rawValue string) (*time.Time, error) {
	if rawValue == "" {
		return nil, nil
	}

	parsedDate, err := time.Parse("2006-01-02", rawValue)
	if err != nil {
		return nil, errors.New("date filters must use YYYY-MM-DD")
	}

	return &parsedDate, nil
}

func queryFacetValues(ctx context.Context, db *pgxpool.Pool, query string, args ...any) ([]string, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func scanProfileTicketRows(rows pgx.Rows) ([]profileTicketResponse, error) {
	tickets := make([]profileTicketResponse, 0)

	for rows.Next() {
		var (
			id             pgtype.UUID
			number         pgtype.Int4
			ticketStatus   string
			description    string
			result         string
			reason         string
			reasonTitle    string
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
			&reasonTitle,
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
			return nil, err
		}

		tickets = append(tickets, profileTicketResponse{
			ID:                 uuidToString(id),
			Number:             number.Int32,
			Status:             ticketStatus,
			Description:        description,
			Result:             result,
			Reason:             reason,
			ReasonTitle:        reasonTitle,
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
		return nil, err
	}

	return tickets, nil
}

func (s *Server) canAccessProfile(ctx context.Context, requesterID pgtype.UUID, targetID pgtype.UUID) (bool, error) {
	if requesterID == targetID {
		return true, nil
	}

	var allowed bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM users requester
			JOIN users target ON target.user_id = $2
			WHERE requester.user_id = $1
			  AND requester.department IS NOT NULL
			  AND requester.department = target.department
		)
	`, requesterID, targetID).Scan(&allowed)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func updateLatestTicketReference(ctx context.Context, executor sqlExecutor, userID pgtype.UUID, ticketID pgtype.UUID) error {
	result, err := executor.Exec(ctx, `
		UPDATE users
		SET latest_ticket = $1
		WHERE user_id = $2
	`, ticketID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func buildProfileAvatarDownloadURL(userID string) string {
	return "/api/profile/" + strings.TrimSpace(userID) + "/avatar"
}

func buildVersionedProfileAvatarURL(userID string, version int64) string {
	return buildProfileAvatarDownloadURL(userID) + "?v=" + strconv.FormatInt(version, 10)
}

func isSupportedProfileAvatarMediaType(mediaType string) bool {
	_, ok := supportedProfileAvatarMediaTypes[strings.TrimSpace(strings.ToLower(mediaType))]
	return ok
}

func (s *Server) handleProfileAvatarUpload(w http.ResponseWriter, r *http.Request) {
	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	userID := pgtype.UUID{}
	if err := userID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	if s.storage == nil {
		http.Error(w, "object storage not configured", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxProfileAvatarUploadSize+(1<<20))

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}
	if len(fileBytes) == 0 {
		http.Error(w, "file must not be empty", http.StatusBadRequest)
		return
	}
	if len(fileBytes) > maxProfileAvatarUploadSize {
		http.Error(w, "profile photo must not exceed 10 MB", http.StatusRequestEntityTooLarge)
		return
	}

	sniffLength := 512
	if len(fileBytes) < sniffLength {
		sniffLength = len(fileBytes)
	}
	mediaType := http.DetectContentType(fileBytes[:sniffLength])
	if !isSupportedProfileAvatarMediaType(mediaType) {
		http.Error(w, "only JPG, PNG, GIF, and WebP images are supported", http.StatusUnsupportedMediaType)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	objectKey := storage.ProfileAvatarObjectKey(userID.String())
	if _, err := s.storage.PutObject(ctx, objectKey, bytes.NewReader(fileBytes), int64(len(fileBytes)), mediaType); err != nil {
		log.Printf("upload profile avatar to MinIO failed: %v", err)
		http.Error(w, "failed to upload profile photo", http.StatusBadGateway)
		return
	}

	logo := buildVersionedProfileAvatarURL(userID.String(), time.Now().UTC().UnixMilli())
	result, err := s.db.Exec(ctx, `
		UPDATE users
		SET logo = $2
		WHERE user_id = $1
	`, userID, logo)
	if err != nil {
		log.Printf("update profile avatar failed: %v", err)
		http.Error(w, "failed to upload profile photo", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected() == 0 {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"logo":    logo,
		"user_id": userID.String(),
	})
}

func (s *Server) handleProfileAvatarDownload(w http.ResponseWriter, r *http.Request, userID pgtype.UUID) {
	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	if s.storage == nil {
		http.Error(w, "object storage not configured", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	var hasAvatar bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE user_id = $1
			  AND COALESCE(logo, '') <> ''
		)
	`, userID).Scan(&hasAvatar); err != nil {
		log.Printf("check profile avatar failed: %v", err)
		http.Error(w, "failed to load profile photo", http.StatusInternalServerError)
		return
	}
	if !hasAvatar {
		http.Error(w, "profile photo not found", http.StatusNotFound)
		return
	}

	object, info, err := s.storage.GetObject(ctx, storage.ProfileAvatarObjectKey(userID.String()))
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			http.Error(w, "profile photo not found", http.StatusNotFound)
			return
		}

		log.Printf("download profile avatar from MinIO failed: %v", err)
		http.Error(w, "failed to load profile photo", http.StatusBadGateway)
		return
	}
	defer object.Close()

	mediaType := strings.TrimSpace(info.ContentType)
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", mediaType)
	if info.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	}

	if _, err := io.Copy(w, object); err != nil {
		log.Printf("stream profile avatar failed: %v", err)
	}
}

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
		UserID             string                     `json:"user_id"`
		Username           string                     `json:"username"`
		Name               string                     `json:"name"`
		Email              string                     `json:"email"`
		Phone              string                     `json:"phone"`
		Logo               string                     `json:"logo"`
		Department         string                     `json:"department"`
		Role               string                     `json:"role"`
		LatestTicket       *string                    `json:"latestTicket,omitempty"`
		LatestTicketStatus string                     `json:"latestTicketStatus"`
		TicketStats        profileTicketStatsResponse `json:"ticketStats"`
		ActiveTickets      []profileTicketResponse    `json:"activeTickets"`
	}

	if r.Method == http.MethodPost && r.URL.Path == "/api/profile/avatar" {
		s.handleProfileAvatarUpload(w, r)
		return
	}

	if r.Method == http.MethodGet && r.URL.Path == "/api/profile/avatar" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet && r.URL.Path != "/api/profile" {
		profilePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/profile/"), "/")
		if profilePath != "" {
			pathParts := strings.Split(profilePath, "/")
			if len(pathParts) == 2 && pathParts[1] == "avatar" {
				targetID := pgtype.UUID{}
				if err := targetID.Scan(pathParts[0]); err != nil {
					http.Error(w, "invalid user id", http.StatusBadRequest)
					return
				}

				s.handleProfileAvatarDownload(w, r, targetID)
				return
			}
		}
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

	requesterID := pgtype.UUID{}
	if err := requesterID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	targetID := requesterID
	subresource := ""
	if r.URL.Path != "/api/profile" {
		profilePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/profile/"), "/")
		if profilePath == "" {
			http.NotFound(w, r)
			return
		}

		pathParts := strings.Split(profilePath, "/")
		if err := targetID.Scan(pathParts[0]); err != nil {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}

		switch {
		case len(pathParts) == 1:
		case len(pathParts) == 2 && pathParts[1] == "tickets":
			subresource = "tickets"
		case len(pathParts) == 3 && pathParts[1] == "tickets" && pathParts[2] == "facets":
			subresource = "tickets-facets"
		default:
			http.NotFound(w, r)
			return
		}
	}

	if s.db == nil {
		if subresource != "" {
			http.Error(w, "database not configured", http.StatusServiceUnavailable)
			return
		}
		if s.queries == nil {
			http.Error(w, "database not configured", http.StatusServiceUnavailable)
			return
		}
		if requesterID != targetID {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		profile, err := s.queries.GetUserProfileByUserID(r.Context(), targetID)
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
			UserID:             uuidToString(profile.UserID),
			Username:           profile.Username,
			Name:               profile.Name,
			Email:              profile.Email,
			Phone:              "",
			Logo:               "",
			Department:         profile.Department,
			Role:               profile.Role,
			LatestTicket:       nil,
			LatestTicketStatus: "",
			TicketStats:        profileTicketStatsResponse{},
			ActiveTickets:      []profileTicketResponse{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	allowed, err := s.canAccessProfile(ctx, requesterID, targetID)
	if err != nil {
		log.Printf("check profile access failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	switch subresource {
	case "tickets-facets":
		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reasonTitles, err := queryFacetValues(ctx, s.db, `
			SELECT DISTINCT COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.executor = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(cls.title, '') = $3)
			  AND ($4::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $4::timestamp)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $5::timestamp)
			ORDER BY reason_title ASC
		`, targetID, filters.Status, filters.DeviceName, filters.StartDate, filters.EndDate)
		if err != nil {
			log.Printf("query profile ticket reason facets failed: %v", err)
			http.Error(w, "failed to load profile ticket facets", http.StatusInternalServerError)
			return
		}

		deviceNames, err := queryFacetValues(ctx, s.db, `
			SELECT DISTINCT COALESCE(cls.title, '') AS device_name
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.executor = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $4::timestamp)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $5::timestamp)
			ORDER BY device_name ASC
		`, targetID, filters.Status, filters.ReasonTitle, filters.StartDate, filters.EndDate)
		if err != nil {
			log.Printf("query profile ticket device facets failed: %v", err)
			http.Error(w, "failed to load profile ticket facets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, ticketArchiveFacetsResponse{
			DeviceNames:  deviceNames,
			ReasonTitles: reasonTitles,
		})
		return
	case "tickets":
		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		limit, offset, err := parseTicketListPagination(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var total int
		if err := s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.executor = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
		`, targetID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate).Scan(&total); err != nil {
			log.Printf("count profile tickets failed: %v", err)
			http.Error(w, "failed to load profile tickets", http.StatusInternalServerError)
			return
		}

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
				COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title,
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
			WHERE t.executor = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
			ORDER BY
			  CASE WHEN $7 = 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END ASC NULLS FIRST,
			  CASE WHEN $7 <> 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END DESC NULLS LAST,
			  CASE WHEN $7 = 'oldest' THEN t.number END ASC,
			  CASE WHEN $7 <> 'oldest' THEN t.number END DESC,
			  t.id ASC
			LIMIT $8
			OFFSET $9
		`, targetID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate, filters.SortBy, limit, offset)
		if err != nil {
			log.Printf("query profile tickets failed: %v", err)
			http.Error(w, "failed to load profile tickets", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tickets, err := scanProfileTicketRows(rows)
		if err != nil {
			log.Printf("scan profile ticket failed: %v", err)
			http.Error(w, "failed to load profile tickets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, newPaginatedResponse(tickets, limit, offset, total))
		return
	}

	row := s.db.QueryRow(ctx, `
		SELECT
			a.user_id,
			a.username,
			TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))) AS name,
			COALESCE(u.email, ''),
			COALESCE(u.phone, ''),
			COALESCE(u.logo, ''),
			COALESCE(d.title, ''),
			COALESCE(r.name, 'user'),
			u.latest_ticket,
			COALESCE(lt.status, '')
		FROM accounts AS a
		JOIN users AS u ON u.user_id = a.user_id
		LEFT JOIN departments AS d ON d.id = u.department
		LEFT JOIN account_roles AS ar ON ar.user_id = a.user_id
		LEFT JOIN roles AS r ON r.id = ar.role_id
		LEFT JOIN tickets AS lt ON lt.id = u.latest_ticket
		WHERE a.user_id = $1
		LIMIT 1
	`, targetID)

	var (
		profileUserID      pgtype.UUID
		username           string
		name               string
		email              string
		phone              string
		logo               string
		department         string
		role               string
		latestTicket       pgtype.UUID
		latestTicketStatus string
	)

	if err := row.Scan(
		&profileUserID,
		&username,
		&name,
		&email,
		&phone,
		&logo,
		&department,
		&role,
		&latestTicket,
		&latestTicketStatus,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		log.Printf("load user profile failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	stats := profileTicketStatsResponse{}
	if err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_count,
			COUNT(*) FILTER (WHERE COALESCE(status, '') = 'closed') AS closed_count,
			COUNT(*) FILTER (
				WHERE COALESCE(status, '') NOT IN ('closed', 'canceled', 'cancelled')
				  AND assigned_end IS NOT NULL
				  AND assigned_end::date <= (NOW() AT TIME ZONE 'UTC')::date
			) AS overdue_count,
			COUNT(*) FILTER (
				WHERE COALESCE(status, '') = 'closed'
				  AND closed_at >= date_trunc('month', NOW() AT TIME ZONE 'UTC')
				  AND closed_at < date_trunc('month', NOW() AT TIME ZONE 'UTC') + INTERVAL '1 month'
			) AS closed_this_month_count
		FROM tickets
		WHERE executor = $1
	`, targetID).Scan(&stats.Total, &stats.Closed, &stats.Overdue, &stats.ClosedThisMonth); err != nil {
		log.Printf("load profile ticket stats failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	activeRows, err := s.db.Query(ctx, `
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
			COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title,
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
		WHERE t.executor = $1
		  AND COALESCE(t.status, '') NOT IN ('closed', 'canceled', 'cancelled')
		ORDER BY
		  CASE WHEN t.assigned_end IS NULL THEN 1 ELSE 0 END ASC,
		  t.assigned_end ASC NULLS LAST,
		  t.created_at DESC,
		  t.number DESC
		LIMIT 2
	`, targetID)
	if err != nil {
		log.Printf("query profile active tickets failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}
	defer activeRows.Close()

	activeTickets, err := scanProfileTicketRows(activeRows)
	if err != nil {
		log.Printf("scan profile active ticket failed: %v", err)
		http.Error(w, "failed to load profile", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, profileResponse{
		UserID:             uuidToString(profileUserID),
		Username:           username,
		Name:               name,
		Email:              email,
		Phone:              phone,
		Logo:               logo,
		Department:         department,
		Role:               role,
		LatestTicket:       nullableUUIDToString(latestTicket),
		LatestTicketStatus: latestTicketStatus,
		TicketStats:        stats,
		ActiveTickets:      activeTickets,
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
		AvatarURL   string  `json:"avatarUrl"`
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
				COALESCE(u.logo, ''),
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
				avatarURL  string
				department string
				refID      pgtype.UUID
				text       string
				createdAt  pgtype.Timestamp
			)

			if err := rows.Scan(&id, &authorID, &authorName, &avatarURL, &department, &refID, &text, &createdAt); err != nil {
				log.Printf("scan comment failed: %v", err)
				http.Error(w, "failed to load comments", http.StatusInternalServerError)
				return
			}

			comments = append(comments, commentResponse{
				ID:          id,
				AuthorID:    uuidToString(authorID),
				AuthorName:  authorName,
				AvatarURL:   avatarURL,
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
				COALESCE(u.logo, ''),
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
			avatarURL  string
			department string
			refID      pgtype.UUID
			text       string
			createdAt  pgtype.Timestamp
		)

		if err := row.Scan(&id, &authorID, &authorName, &avatarURL, &department, &refID, &text, &createdAt); err != nil {
			log.Printf("insert comment failed: %v", err)
			http.Error(w, "failed to create comment", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, commentResponse{
			ID:          id,
			AuthorID:    uuidToString(authorID),
			AuthorName:  authorName,
			AvatarURL:   avatarURL,
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
		ReasonTitle        string  `json:"reasonTitle"`
		Urgent             bool    `json:"urgent"`
		ExternalAuthor     *string `json:"externalAuthor"`
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
	case len(pathParts) == 3 && pathParts[1] == "tickets" && pathParts[2] == "facets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		reasonTitles, err := queryFacetValues(ctx, s.db, `
			SELECT DISTINCT COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.client = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(cls.title, '') = $3)
			  AND ($4::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $4::timestamp)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $5::timestamp)
			ORDER BY reason_title ASC
		`, clientID, filters.Status, filters.DeviceName, filters.StartDate, filters.EndDate)
		if err != nil {
			log.Printf("query client ticket reason facets failed: %v", err)
			http.Error(w, "failed to load client ticket facets", http.StatusInternalServerError)
			return
		}

		deviceNames, err := queryFacetValues(ctx, s.db, `
			SELECT DISTINCT COALESCE(cls.title, '') AS device_name
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.client = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $4::timestamp)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $5::timestamp)
			ORDER BY device_name ASC
		`, clientID, filters.Status, filters.ReasonTitle, filters.StartDate, filters.EndDate)
		if err != nil {
			log.Printf("query client ticket device facets failed: %v", err)
			http.Error(w, "failed to load client ticket facets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, ticketArchiveFacetsResponse{
			DeviceNames:  deviceNames,
			ReasonTitles: reasonTitles,
		})
		return
	case len(pathParts) == 2 && pathParts[1] == "tickets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		limit, offset, err := parseTicketListPagination(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var total int
		if err := s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.client = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
		`, clientID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate).Scan(&total); err != nil {
			log.Printf("count client tickets failed: %v", err)
			http.Error(w, "failed to load client tickets", http.StatusInternalServerError)
			return
		}

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
				COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title,
				t.urgent,
				t.external_author,
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
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
			ORDER BY
			  CASE WHEN $7 = 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END ASC NULLS FIRST,
			  CASE WHEN $7 <> 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END DESC NULLS LAST,
			  CASE WHEN $7 = 'oldest' THEN t.number END ASC,
			  CASE WHEN $7 <> 'oldest' THEN t.number END DESC,
			  t.id ASC
			LIMIT $8
			OFFSET $9
		`, clientID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate, filters.SortBy, limit, offset)
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
				reasonTitle    string
				urgent         bool
				externalAuthor pgtype.UUID
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
				&reasonTitle,
				&urgent,
				&externalAuthor,
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
				ReasonTitle:        reasonTitle,
				Urgent:             urgent,
				ExternalAuthor:     nullableUUIDToString(externalAuthor),
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

		writeJSON(w, http.StatusOK, newPaginatedResponse(tickets, limit, offset, total))
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
				a.on_warranty
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
				id           pgtype.UUID
				number       pgtype.Int4
				device       pgtype.UUID
				deviceName   string
				deviceSerial string
				assignedAt   pgtype.Timestamp
				finishedAt   pgtype.Timestamp
				isActive     bool
				onWarranty   bool
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
		ReasonTitle        string  `json:"reasonTitle"`
		Urgent             bool    `json:"urgent"`
		ExternalAuthor     *string `json:"externalAuthor"`
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
	case len(pathParts) == 3 && pathParts[1] == "tickets" && pathParts[2] == "facets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		reasonTitles, err := queryFacetValues(ctx, s.db, `
			SELECT DISTINCT COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.device = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $3::timestamp)
			  AND ($4::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $4::timestamp)
			ORDER BY reason_title ASC
		`, deviceID, filters.Status, filters.StartDate, filters.EndDate)
		if err != nil {
			log.Printf("query device ticket reason facets failed: %v", err)
			http.Error(w, "failed to load device ticket facets", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, ticketArchiveFacetsResponse{
			DeviceNames:  []string{},
			ReasonTitles: reasonTitles,
		})
		return
	case len(pathParts) == 2 && pathParts[1] == "tickets":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filters, err := parseTicketListFilters(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		limit, offset, err := parseTicketListPagination(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var total int
		if err := s.db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM tickets t
			LEFT JOIN devices d ON d.id = t.device
			LEFT JOIN classificators cls ON cls.id = d.classificator
			LEFT JOIN ticket_reasons tr ON tr.id = t.reason
			WHERE t.device = $1
			  AND ($2 = '' OR COALESCE(t.status, '') = $2)
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
		`, deviceID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate).Scan(&total); err != nil {
			log.Printf("count device tickets failed: %v", err)
			http.Error(w, "failed to load device tickets", http.StatusInternalServerError)
			return
		}

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
				COALESCE(NULLIF(tr.title, ''), 'Не указано') AS reason_title,
				t.urgent,
				t.external_author,
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
			  AND ($3 = '' OR COALESCE(NULLIF(tr.title, ''), 'Не указано') = $3)
			  AND ($4 = '' OR COALESCE(cls.title, '') = $4)
			  AND ($5::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) >= $5::timestamp)
			  AND ($6::timestamp IS NULL OR COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) < $6::timestamp)
			ORDER BY
			  CASE WHEN $7 = 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END ASC NULLS FIRST,
			  CASE WHEN $7 <> 'oldest' THEN COALESCE(t.closed_at, t.workfinished_at, t.assigned_end, t.workstarted_at) END DESC NULLS LAST,
			  CASE WHEN $7 = 'oldest' THEN t.number END ASC,
			  CASE WHEN $7 <> 'oldest' THEN t.number END DESC,
			  t.id ASC
			LIMIT $8
			OFFSET $9
		`, deviceID, filters.Status, filters.ReasonTitle, filters.DeviceName, filters.StartDate, filters.EndDate, filters.SortBy, limit, offset)
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
				reasonTitle    string
				urgent         bool
				externalAuthor pgtype.UUID
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
				&reasonTitle,
				&urgent,
				&externalAuthor,
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
				ReasonTitle:        reasonTitle,
				Urgent:             urgent,
				ExternalAuthor:     nullableUUIDToString(externalAuthor),
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

		writeJSON(w, http.StatusOK, newPaginatedResponse(tickets, limit, offset, total))
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
				a.on_warranty
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

func (s *Server) handleDepartmentMembers(w http.ResponseWriter, r *http.Request) {
	type departmentMemberResponse struct {
		ID                 string  `json:"id"`
		Name               string  `json:"name"`
		Username           string  `json:"username"`
		Department         string  `json:"department"`
		AvatarURL          string  `json:"avatarUrl"`
		IsDisabled         bool    `json:"isDisabled"`
		LatestTicket       *string `json:"latestTicket,omitempty"`
		LatestTicketStatus string  `json:"latestTicketStatus"`
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

	var requesterID pgtype.UUID
	if err := requesterID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT
			u.user_id,
			TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))) AS name,
			a.username,
			COALESCE(d.title, ''),
			COALESCE(u.logo, ''),
			a.disabled,
			u.latest_ticket,
			COALESCE(lt.status, '')
		FROM users u
		JOIN accounts a ON a.user_id = u.user_id
		LEFT JOIN departments d ON d.id = u.department
		LEFT JOIN tickets lt ON lt.id = u.latest_ticket
		WHERE u.department = (
			SELECT department
			FROM users
			WHERE user_id = $1
		)
		  AND u.department IS NOT NULL
		ORDER BY
			TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))) ASC,
			a.username ASC,
			u.user_id ASC
	`, requesterID)
	if err != nil {
		log.Printf("query department members failed: %v", err)
		http.Error(w, "failed to load department members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]departmentMemberResponse, 0)
	for rows.Next() {
		var (
			id                 pgtype.UUID
			name               string
			username           string
			department         string
			avatarURL          string
			isDisabled         bool
			latestTicket       pgtype.UUID
			latestTicketStatus string
		)

		if err := rows.Scan(&id, &name, &username, &department, &avatarURL, &isDisabled, &latestTicket, &latestTicketStatus); err != nil {
			log.Printf("scan department member failed: %v", err)
			http.Error(w, "failed to load department members", http.StatusInternalServerError)
			return
		}

		response = append(response, departmentMemberResponse{
			ID:                 uuidToString(id),
			Name:               name,
			Username:           username,
			Department:         department,
			AvatarURL:          avatarURL,
			IsDisabled:         isDisabled,
			LatestTicket:       nullableUUIDToString(latestTicket),
			LatestTicketStatus: latestTicketStatus,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate department members failed: %v", err)
		http.Error(w, "failed to load department members", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleTicketReasons(w http.ResponseWriter, r *http.Request) {
	type ticketReasonResponse struct {
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
		SELECT COALESCE(id, ''), COALESCE(title, '')
		FROM ticket_reasons
		ORDER BY title ASC, id ASC
	`)
	if err != nil {
		log.Printf("query ticket reasons failed: %v", err)
		http.Error(w, "failed to load ticket reasons", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]ticketReasonResponse, 0)
	for rows.Next() {
		var (
			id    string
			title string
		)

		if err := rows.Scan(&id, &title); err != nil {
			log.Printf("scan ticket reason failed: %v", err)
			http.Error(w, "failed to load ticket reasons", http.StatusInternalServerError)
			return
		}

		response = append(response, ticketReasonResponse{
			ID:    id,
			Title: title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate ticket reasons failed: %v", err)
		http.Error(w, "failed to load ticket reasons", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func parseTicketDateInput(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}

	if parsedDate, err := time.Parse("2006-01-02", value); err == nil {
		return time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 12, 0, 0, 0, time.UTC), nil
	}

	parsedTimestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}

	return parsedTimestamp.UTC(), nil
}

func syncSecretMatches(expected string, provided string) bool {
	expected = strings.TrimSpace(expected)
	provided = strings.TrimSpace(provided)
	if expected == "" || len(expected) != len(provided) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func normalizeTicketSyncMetadata(source string, key string) (string, string) {
	source = strings.TrimSpace(source)
	key = strings.TrimSpace(key)
	if key != "" && source == "" {
		source = defaultTicketSyncSource
	}

	return source, key
}

func normalizeTicketSyncAuthor(author string, authorTitle string, legacyAuthor string, legacyAuthorTitle string) (string, string) {
	author = strings.TrimSpace(author)
	authorTitle = strings.TrimSpace(authorTitle)
	if author != "" || authorTitle != "" {
		return author, authorTitle
	}

	return strings.TrimSpace(legacyAuthor), strings.TrimSpace(legacyAuthorTitle)
}

func (s *Server) handleTicketSync(w http.ResponseWriter, r *http.Request) {
	type syncTicketRequest struct {
		Author              string `json:"author"`
		AuthorTitle         string `json:"author_title"`
		Client              string `json:"client"`
		ContactPerson       string `json:"contact_person"`
		Department          string `json:"department"`
		Description         string `json:"description"`
		Device              string `json:"device"`
		ExternalAuthorID    string `json:"external_author_id"`
		ExternalAuthorTitle string `json:"external_author_title"`
		Reason              string `json:"reason"`
		Source              string `json:"source"`
		SyncKey             string `json:"sync_key"`
		TicketType          string `json:"ticket_type"`
		Urgent              bool   `json:"urgent"`
	}

	type syncTicketResponse struct {
		Author     *string `json:"author,omitempty"`
		Department string  `json:"department"`
		Duplicate  bool    `json:"duplicate"`
		ID         string  `json:"id"`
		Number     int32   `json:"number"`
		Source     string  `json:"source,omitempty"`
		Status     string  `json:"status"`
		SyncKey    string  `json:"sync_key,omitempty"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.sync.Enabled() {
		http.Error(w, "ticket sync is not configured", http.StatusServiceUnavailable)
		return
	}

	if !syncSecretMatches(s.sync.sharedSecret, r.Header.Get("X-Sync-Secret")) {
		log.Printf("ticket sync rejected: invalid secret remote=%q", r.RemoteAddr)
		http.Error(w, "invalid sync secret", http.StatusUnauthorized)
		return
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	defer r.Body.Close()

	var input syncTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Author, input.AuthorTitle = normalizeTicketSyncAuthor(input.Author, input.AuthorTitle, input.ExternalAuthorID, input.ExternalAuthorTitle)
	input.Client = strings.TrimSpace(input.Client)
	input.ContactPerson = strings.TrimSpace(input.ContactPerson)
	input.Department = strings.TrimSpace(input.Department)
	input.Description = strings.TrimSpace(input.Description)
	input.Device = strings.TrimSpace(input.Device)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Source, input.SyncKey = normalizeTicketSyncMetadata(input.Source, input.SyncKey)
	input.TicketType = strings.TrimSpace(input.TicketType)

	log.Printf(
		"ticket sync received remote=%q source=%q sync_key=%q client=%q device=%q department=%q author=%q urgent=%t",
		r.RemoteAddr,
		input.Source,
		input.SyncKey,
		input.Client,
		input.Device,
		input.Department,
		input.Author,
		input.Urgent,
	)

	switch {
	case input.Device == "":
		http.Error(w, "device is required", http.StatusBadRequest)
		return
	case input.Client == "":
		http.Error(w, "client is required", http.StatusBadRequest)
		return
	case input.Reason == "":
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	case input.Description == "":
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	case input.ContactPerson == "":
		http.Error(w, "contact_person is required", http.StatusBadRequest)
		return
	case input.Department == "":
		http.Error(w, "department is required", http.StatusBadRequest)
		return
	case input.AuthorTitle != "" && input.Author == "":
		http.Error(w, "author is required when author_title is provided", http.StatusBadRequest)
		return
	}

	var (
		clientID         pgtype.UUID
		contactID        pgtype.UUID
		departmentID     pgtype.UUID
		deviceID         pgtype.UUID
		externalAuthorID pgtype.UUID
	)

	if err := clientID.Scan(input.Client); err != nil {
		http.Error(w, "client must be a valid UUID", http.StatusBadRequest)
		return
	}
	if err := contactID.Scan(input.ContactPerson); err != nil {
		http.Error(w, "contact_person must be a valid UUID", http.StatusBadRequest)
		return
	}
	if err := deviceID.Scan(input.Device); err != nil {
		http.Error(w, "device must be a valid UUID", http.StatusBadRequest)
		return
	}
	if input.Author != "" {
		if err := externalAuthorID.Scan(input.Author); err != nil {
			http.Error(w, "author must be a valid UUID", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if input.Source != "" && input.SyncKey != "" {
		var (
			existingTicketID       pgtype.UUID
			existingTicketNo       int32
			existingStatus         string
			existingDepartmentID   pgtype.UUID
			existingExternalAuthor pgtype.UUID
		)
		err := s.db.QueryRow(ctx, `
			SELECT
				id,
				number,
				COALESCE(status, ''),
				department,
				external_author
			FROM tickets
			WHERE sync_source = $1
			  AND sync_key = $2
		`, input.Source, input.SyncKey).Scan(
			&existingTicketID,
			&existingTicketNo,
			&existingStatus,
			&existingDepartmentID,
			&existingExternalAuthor,
		)
		if err == nil {
			log.Printf(
				"ticket sync duplicate remote=%q source=%q sync_key=%q ticket_id=%s ticket_number=%d",
				r.RemoteAddr,
				input.Source,
				input.SyncKey,
				uuidToString(existingTicketID),
				existingTicketNo,
			)
			writeJSON(w, http.StatusOK, syncTicketResponse{
				Author:     nullableUUIDToString(existingExternalAuthor),
				Department: uuidToString(existingDepartmentID),
				Duplicate:  true,
				ID:         uuidToString(existingTicketID),
				Number:     existingTicketNo,
				Source:     input.Source,
				Status:     existingStatus,
				SyncKey:    input.SyncKey,
			})
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("load existing synced ticket failed: %v", err)
			http.Error(w, "failed to create ticket", http.StatusInternalServerError)
			return
		}
	}

	if err := departmentID.Scan(input.Department); err != nil {
		if queryErr := s.db.QueryRow(ctx, `
			SELECT id
			FROM departments
			WHERE LOWER(title) = LOWER($1)
		`, input.Department).Scan(&departmentID); queryErr != nil {
			if errors.Is(queryErr, pgx.ErrNoRows) {
				http.Error(w, "department not found", http.StatusBadRequest)
				return
			}

			log.Printf("resolve ticket sync department failed: %v", queryErr)
			http.Error(w, "failed to create ticket", http.StatusInternalServerError)
			return
		}
	}

	var departmentExists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1)`, departmentID).Scan(&departmentExists); err != nil {
		log.Printf("validate ticket sync department failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !departmentExists {
		http.Error(w, "department not found", http.StatusBadRequest)
		return
	}

	var reasonExists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ticket_reasons WHERE id = $1)`, input.Reason).Scan(&reasonExists); err != nil {
		log.Printf("validate ticket sync reason failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !reasonExists {
		http.Error(w, "reason not found", http.StatusBadRequest)
		return
	}

	if input.TicketType != "" {
		var ticketTypeExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ticket_types WHERE type = $1)`, input.TicketType).Scan(&ticketTypeExists); err != nil {
			log.Printf("validate ticket sync ticket type failed: %v", err)
			http.Error(w, "failed to create ticket", http.StatusInternalServerError)
			return
		}
		if !ticketTypeExists {
			http.Error(w, "ticket_type not found", http.StatusBadRequest)
			return
		}
	}

	var resolvedClientID pgtype.UUID
	if err := s.db.QueryRow(ctx, `
		SELECT a.actual_client
		FROM devices d
		LEFT JOIN LATERAL (
			SELECT a.actual_client
			FROM agreements a
			WHERE a.device = d.id
			ORDER BY a.is_active DESC, a.assigned_at DESC NULLS LAST, a.number DESC
			LIMIT 1
		) a ON TRUE
		WHERE d.id = $1
	`, deviceID).Scan(&resolvedClientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		log.Printf("load ticket sync device client failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !resolvedClientID.Valid {
		http.Error(w, "device has no client", http.StatusBadRequest)
		return
	}
	if resolvedClientID != clientID {
		http.Error(w, "client does not match device", http.StatusBadRequest)
		return
	}

	var contactExists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM contacts
			WHERE id = $1
			  AND client_id = $2
		)
	`, contactID, clientID).Scan(&contactExists); err != nil {
		log.Printf("validate ticket sync contact failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !contactExists {
		http.Error(w, "contact_person not found", http.StatusBadRequest)
		return
	}

	if input.Author != "" && input.AuthorTitle == "" {
		var externalAuthorExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM external_users WHERE id = $1)`, externalAuthorID).Scan(&externalAuthorExists); err != nil {
			log.Printf("validate ticket sync external author failed: %v", err)
			http.Error(w, "failed to create ticket", http.StatusInternalServerError)
			return
		}
		if !externalAuthorExists {
			log.Printf(
				"ticket sync rejected: unknown author source=%q sync_key=%q author=%q",
				input.Source,
				input.SyncKey,
				input.Author,
			)
			http.Error(w, "author not found; provide author_title to create it", http.StatusBadRequest)
			return
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("begin ticket sync transaction failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	if input.Author != "" && input.AuthorTitle != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO external_users (id, title)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE
			SET title = EXCLUDED.title
		`, externalAuthorID, input.AuthorTitle); err != nil {
			log.Printf("upsert ticket sync external author failed: %v", err)
			http.Error(w, "failed to create ticket", http.StatusInternalServerError)
			return
		}

		log.Printf(
			"ticket sync author upserted source=%q sync_key=%q author=%q title=%q",
			input.Source,
			input.SyncKey,
			input.Author,
			input.AuthorTitle,
		)
	}

	var (
		createdTicketID pgtype.UUID
		createdTicketNo int32
		createdStatus   string
	)
	if err := tx.QueryRow(ctx, `
		INSERT INTO tickets (
			client,
			device,
			ticket_type,
			external_author,
			department,
			reason,
			description,
			contact_person,
			status,
			urgent,
			sync_source,
			sync_key
		)
		VALUES (
			$1,
			$2,
			NULLIF($3, ''),
			$4,
			$5,
			$6,
			$7,
			$8,
			'created',
			$9,
			NULLIF($10, ''),
			NULLIF($11, '')
		)
		ON CONFLICT (sync_source, sync_key)
		WHERE sync_source IS NOT NULL AND sync_key IS NOT NULL
		DO NOTHING
		RETURNING id, number, COALESCE(status, '')
	`, clientID, deviceID, input.TicketType, nullableUUID(externalAuthorID), departmentID, input.Reason, input.Description, contactID, input.Urgent, input.Source, input.SyncKey).Scan(
		&createdTicketID,
		&createdTicketNo,
		&createdStatus,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) && input.Source != "" && input.SyncKey != "" {
			var existingDepartmentID pgtype.UUID
			var existingExternalAuthor pgtype.UUID
			if err := tx.QueryRow(ctx, `
				SELECT id, number, COALESCE(status, ''), department, external_author
				FROM tickets
				WHERE sync_source = $1
				  AND sync_key = $2
			`, input.Source, input.SyncKey).Scan(&createdTicketID, &createdTicketNo, &createdStatus, &existingDepartmentID, &existingExternalAuthor); err != nil {
				log.Printf("load duplicate synced ticket failed: %v", err)
				http.Error(w, "failed to create ticket", http.StatusInternalServerError)
				return
			}

			if err := tx.Commit(ctx); err != nil {
				log.Printf("commit duplicate ticket sync transaction failed: %v", err)
				http.Error(w, "failed to create ticket", http.StatusInternalServerError)
				return
			}

			log.Printf(
				"ticket sync duplicate after insert race source=%q sync_key=%q ticket_id=%s ticket_number=%d",
				input.Source,
				input.SyncKey,
				uuidToString(createdTicketID),
				createdTicketNo,
			)

			writeJSON(w, http.StatusOK, syncTicketResponse{
				Author:     nullableUUIDToString(existingExternalAuthor),
				Department: uuidToString(existingDepartmentID),
				Duplicate:  true,
				ID:         uuidToString(createdTicketID),
				Number:     createdTicketNo,
				Source:     input.Source,
				Status:     createdStatus,
				SyncKey:    input.SyncKey,
			})
			return
		}

		log.Printf("create synced ticket failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit ticket sync transaction failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"ticket sync created source=%q sync_key=%q ticket_id=%s ticket_number=%d department=%s author=%q",
		input.Source,
		input.SyncKey,
		uuidToString(createdTicketID),
		createdTicketNo,
		uuidToString(departmentID),
		input.Author,
	)

	writeJSON(w, http.StatusCreated, syncTicketResponse{
		Author:     nullableUUIDToString(externalAuthorID),
		Department: uuidToString(departmentID),
		Duplicate:  false,
		ID:         uuidToString(createdTicketID),
		Number:     createdTicketNo,
		Source:     input.Source,
		Status:     createdStatus,
		SyncKey:    input.SyncKey,
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
		ExternalAuthor     *string `json:"externalAuthor"`
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

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
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

	if r.Method == http.MethodPost {
		s.handleTicketsCreate(w, r, claims.Subject)
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
			t.external_author,
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
			externalAuthor pgtype.UUID
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
			&externalAuthor,
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
			ExternalAuthor:     nullableUUIDToString(externalAuthor),
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

func (s *Server) handleTicketsCreate(w http.ResponseWriter, r *http.Request, requesterSubject string) {
	type createTicketRequest struct {
		AssignedEnd   string `json:"assigned_end"`
		AssignedStart string `json:"assigned_start"`
		Client        string `json:"client"`
		ContactPerson string `json:"contact_person"`
		Description   string `json:"description"`
		Device        string `json:"device"`
		Executor      string `json:"executor"`
		Reason        string `json:"reason"`
		Urgent        bool   `json:"urgent"`
	}

	type createTicketResponse struct {
		AssignedAt    *string `json:"assigned_at,omitempty"`
		AssignedEnd   *string `json:"assigned_end,omitempty"`
		AssignedStart *string `json:"assigned_start,omitempty"`
		ID            string  `json:"id"`
		Number        int32   `json:"number"`
		Status        string  `json:"status"`
	}

	defer r.Body.Close()

	var input createTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.AssignedEnd = strings.TrimSpace(input.AssignedEnd)
	input.AssignedStart = strings.TrimSpace(input.AssignedStart)
	input.Client = strings.TrimSpace(input.Client)
	input.ContactPerson = strings.TrimSpace(input.ContactPerson)
	input.Description = strings.TrimSpace(input.Description)
	input.Device = strings.TrimSpace(input.Device)
	input.Executor = strings.TrimSpace(input.Executor)
	input.Reason = strings.TrimSpace(input.Reason)

	switch {
	case input.Device == "":
		http.Error(w, "device is required", http.StatusBadRequest)
		return
	case input.Client == "":
		http.Error(w, "client is required", http.StatusBadRequest)
		return
	case input.Reason == "":
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	case input.Description == "":
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	case input.ContactPerson == "":
		http.Error(w, "contact_person is required", http.StatusBadRequest)
		return
	case input.Executor == "":
		http.Error(w, "executor is required", http.StatusBadRequest)
		return
	case input.AssignedStart == "":
		http.Error(w, "assigned_start is required", http.StatusBadRequest)
		return
	case input.AssignedEnd == "":
		http.Error(w, "assigned_end is required", http.StatusBadRequest)
		return
	}

	assignedStart, err := parseTicketDateInput(input.AssignedStart)
	if err != nil {
		http.Error(w, "assigned_start must be a date or ISO timestamp", http.StatusBadRequest)
		return
	}

	assignedEnd, err := parseTicketDateInput(input.AssignedEnd)
	if err != nil {
		http.Error(w, "assigned_end must be a date or ISO timestamp", http.StatusBadRequest)
		return
	}

	if assignedStart.After(assignedEnd) {
		http.Error(w, "assigned_end must be greater than or equal to assigned_start", http.StatusBadRequest)
		return
	}

	var (
		requesterID pgtype.UUID
		clientID    pgtype.UUID
		contactID   pgtype.UUID
		deviceID    pgtype.UUID
		executorID  pgtype.UUID
	)

	if err := requesterID.Scan(requesterSubject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}
	if err := clientID.Scan(input.Client); err != nil {
		http.Error(w, "client must be a valid UUID", http.StatusBadRequest)
		return
	}
	if err := contactID.Scan(input.ContactPerson); err != nil {
		http.Error(w, "contact_person must be a valid UUID", http.StatusBadRequest)
		return
	}
	if err := deviceID.Scan(input.Device); err != nil {
		http.Error(w, "device must be a valid UUID", http.StatusBadRequest)
		return
	}
	if err := executorID.Scan(input.Executor); err != nil {
		http.Error(w, "executor must be a valid UUID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var (
		requesterDepartment pgtype.UUID
		requesterRole       string
	)
	if err := s.db.QueryRow(ctx, `
		SELECT
			u.department,
			COALESCE(r.name, 'user')
		FROM users u
		LEFT JOIN account_roles ar ON ar.user_id = u.user_id
		LEFT JOIN roles r ON r.id = ar.role_id
		WHERE u.user_id = $1
	`, requesterID).Scan(&requesterDepartment, &requesterRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "profile not found", http.StatusNotFound)
			return
		}

		log.Printf("load requester profile failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	if requesterRole != "admin" && requesterRole != "coordinator" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !requesterDepartment.Valid {
		http.Error(w, "requester department is required", http.StatusBadRequest)
		return
	}

	var reasonExists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ticket_reasons WHERE id = $1)`, input.Reason).Scan(&reasonExists); err != nil {
		log.Printf("validate ticket reason failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !reasonExists {
		http.Error(w, "reason not found", http.StatusBadRequest)
		return
	}

	var resolvedClientID pgtype.UUID
	if err := s.db.QueryRow(ctx, `
		SELECT a.actual_client
		FROM devices d
		LEFT JOIN LATERAL (
			SELECT a.actual_client
			FROM agreements a
			WHERE a.device = d.id
			ORDER BY a.is_active DESC, a.assigned_at DESC NULLS LAST, a.number DESC
			LIMIT 1
		) a ON TRUE
		WHERE d.id = $1
	`, deviceID).Scan(&resolvedClientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		log.Printf("load device client failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !resolvedClientID.Valid {
		http.Error(w, "device has no client", http.StatusBadRequest)
		return
	}
	if resolvedClientID != clientID {
		http.Error(w, "client does not match device", http.StatusBadRequest)
		return
	}

	var contactExists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM contacts
			WHERE id = $1
			  AND client_id = $2
		)
	`, contactID, clientID).Scan(&contactExists); err != nil {
		log.Printf("validate contact failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !contactExists {
		http.Error(w, "contact_person not found", http.StatusBadRequest)
		return
	}

	var executorExists bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users u
			JOIN accounts a ON a.user_id = u.user_id
			WHERE u.user_id = $1
			  AND u.department = $2
			  AND a.disabled = FALSE
		)
	`, executorID, requesterDepartment).Scan(&executorExists); err != nil {
		log.Printf("validate executor failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	if !executorExists {
		http.Error(w, "executor not found in requester department", http.StatusBadRequest)
		return
	}

	var (
		createdTicketID    pgtype.UUID
		createdTicketNo    int32
		createdTicketState string
		assignedAt         pgtype.Timestamp
		storedStart        pgtype.Timestamp
		storedEnd          pgtype.Timestamp
		tx                 pgx.Tx
	)
	tx, err = s.db.Begin(ctx)
	if err != nil {
		log.Printf("begin create ticket transaction failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, `
		INSERT INTO tickets (
			assigned_at,
			assigned_start,
			assigned_end,
			urgent,
			client,
			device,
			author,
			department,
			assigned_by,
			reason,
			description,
			contact_person,
			executor,
			status
		)
		VALUES (
			(NOW() AT TIME ZONE 'UTC'),
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			'assigned'
		)
		RETURNING id, number, COALESCE(status, ''), assigned_at, assigned_start, assigned_end
	`, assignedStart, assignedEnd, input.Urgent, clientID, deviceID, requesterID, requesterDepartment, requesterID, input.Reason, input.Description, contactID, executorID).Scan(
		&createdTicketID,
		&createdTicketNo,
		&createdTicketState,
		&assignedAt,
		&storedStart,
		&storedEnd,
	); err != nil {
		log.Printf("create ticket failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	if err := updateLatestTicketReference(ctx, tx, executorID, createdTicketID); err != nil {
		log.Printf("update latest ticket reference failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit create ticket transaction failed: %v", err)
		http.Error(w, "failed to create ticket", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, createTicketResponse{
		AssignedAt:    timestampToRFC3339(assignedAt),
		AssignedEnd:   timestampToRFC3339(storedEnd),
		AssignedStart: timestampToRFC3339(storedStart),
		ID:            uuidToString(createdTicketID),
		Number:        createdTicketNo,
		Status:        createdTicketState,
	})
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
		ExternalAuthor     *string                    `json:"externalAuthor"`
		AuthorName         string                     `json:"authorName"`
		Department         *string                    `json:"department"`
		DepartmentTitle    string                     `json:"departmentTitle"`
		AssignedBy         *string                    `json:"assignedBy"`
		AssignedByName     string                     `json:"assignedByName"`
		AssignedByAvatar   string                     `json:"assignedByAvatarUrl"`
		ContactPerson      *string                    `json:"contactPerson"`
		ContactName        string                     `json:"contactName"`
		ContactPosition    string                     `json:"contactPosition"`
		ContactPhone       string                     `json:"contactPhone"`
		ContactEmail       string                     `json:"contactEmail"`
		Executor           *string                    `json:"executor"`
		ExecutorName       string                     `json:"executorName"`
		ExecutorAvatar     string                     `json:"executorAvatarUrl"`
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

		s.handleTicketAttachmentDownload(w, r, ticketID, pathParts[2])
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
			COALESCE(t.author, eu.linked_user_id),
			t.external_author,
			COALESCE(
				NULLIF(TRIM(CONCAT(COALESCE(u_author.first_name, ''), ' ', COALESCE(u_author.last_name, ''))), ''),
				NULLIF(TRIM(CONCAT(COALESCE(u_external.first_name, ''), ' ', COALESCE(u_external.last_name, ''))), ''),
				COALESCE(eu.title, '')
			),
			t.department,
			COALESCE(dpt.title, ''),
			t.assigned_by,
			TRIM(CONCAT(COALESCE(u_assigned.first_name, ''), ' ', COALESCE(u_assigned.last_name, ''))),
			COALESCE(u_assigned.logo, ''),
			t.contact_person,
			COALESCE(cp.name, ''),
			COALESCE(cp.position, ''),
			COALESCE(cp.phone, ''),
			COALESCE(cp.email, ''),
			t.executor,
			TRIM(CONCAT(COALESCE(u_exec.first_name, ''), ' ', COALESCE(u_exec.last_name, ''))),
			COALESCE(u_exec.logo, ''),
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
		LEFT JOIN external_users eu ON eu.id = t.external_author
		LEFT JOIN users u_external ON u_external.user_id = eu.linked_user_id
		LEFT JOIN users u_assigned ON u_assigned.user_id = t.assigned_by
		LEFT JOIN departments dpt ON dpt.id = t.department
		LEFT JOIN contacts cp ON cp.id = t.contact_person
		WHERE t.id = $1
	`, ticketID)

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
		externalAuthor     pgtype.UUID
		authorName         string
		department         pgtype.UUID
		departmentTitle    string
		assignedBy         pgtype.UUID
		assignedByName     string
		assignedByAvatar   string
		contactPerson      pgtype.UUID
		contactName        string
		contactPosition    string
		contactPhone       string
		contactEmail       string
		executor           pgtype.UUID
		executorName       string
		executorAvatar     string
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
		&externalAuthor,
		&authorName,
		&department,
		&departmentTitle,
		&assignedBy,
		&assignedByName,
		&assignedByAvatar,
		&contactPerson,
		&contactName,
		&contactPosition,
		&contactPhone,
		&contactEmail,
		&executor,
		&executorName,
		&executorAvatar,
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
		ExternalAuthor:     nullableUUIDToString(externalAuthor),
		AuthorName:         authorName,
		Department:         nullableUUIDToString(department),
		DepartmentTitle:    departmentTitle,
		AssignedBy:         nullableUUIDToString(assignedBy),
		AssignedByName:     assignedByName,
		AssignedByAvatar:   assignedByAvatar,
		ContactPerson:      nullableUUIDToString(contactPerson),
		ContactName:        contactName,
		ContactPosition:    contactPosition,
		ContactPhone:       contactPhone,
		ContactEmail:       contactEmail,
		Executor:           nullableUUIDToString(executor),
		ExecutorName:       executorName,
		ExecutorAvatar:     executorAvatar,
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
		AssignedEnd              string                         `json:"assigned_end"`
		AssignedStart            string                         `json:"assigned_start"`
		ClosedAt                 string                         `json:"closed_at"`
		ContactPerson            string                         `json:"contact_person"`
		Description              string                         `json:"description"`
		DoubleSigned             bool                           `json:"double_signed"`
		Executor                 string                         `json:"executor"`
		Recommendation           string                         `json:"recommendation"`
		RecommendationDepartment string                         `json:"recommendation_department"`
		Reason                   string                         `json:"reason"`
		Result                   string                         `json:"result"`
		Status                   string                         `json:"status"`
		Urgent                   bool                           `json:"urgent"`
		WorkstartedAt            string                         `json:"workstarted_at"`
		WorkfinishedAt           string                         `json:"workfinished_at"`
	}

	type patchTicketFollowUpResponse struct {
		ID     string `json:"id"`
		Number int32  `json:"number"`
		Status string `json:"status"`
	}

	type patchTicketResponse struct {
		AssignedAt     *string                         `json:"assigned_at,omitempty"`
		AssignedEnd    *string                         `json:"assigned_end,omitempty"`
		AssignedStart  *string                         `json:"assigned_start,omitempty"`
		Attachments    []patchTicketAttachmentResponse `json:"attachments,omitempty"`
		ClosedAt       *string                         `json:"closed_at,omitempty"`
		ContactPerson  *string                         `json:"contact_person,omitempty"`
		Description    string                          `json:"description,omitempty"`
		Executor       *string                         `json:"executor,omitempty"`
		FollowUpTicket *patchTicketFollowUpResponse    `json:"followUpTicket,omitempty"`
		ID             string                          `json:"id"`
		Reason         string                          `json:"reason,omitempty"`
		Status         string                          `json:"status"`
		Urgent         *bool                           `json:"urgent,omitempty"`
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
	input.AssignedStart = strings.TrimSpace(input.AssignedStart)
	input.AssignedEnd = strings.TrimSpace(input.AssignedEnd)
	input.WorkstartedAt = strings.TrimSpace(input.WorkstartedAt)
	input.WorkfinishedAt = strings.TrimSpace(input.WorkfinishedAt)
	input.ClosedAt = strings.TrimSpace(input.ClosedAt)
	input.ContactPerson = strings.TrimSpace(input.ContactPerson)
	input.Description = strings.TrimSpace(input.Description)
	input.Executor = strings.TrimSpace(input.Executor)
	input.Reason = strings.TrimSpace(input.Reason)
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
		conflictMessage       = "ticket must be assigned to you and in expected status"
		expectedCurrentStatus string
		latestTicketUserID    *pgtype.UUID
		response              patchTicketResponse
		tx                    pgx.Tx
	)

	response.ID = ticketID.String()
	response.Status = input.Status

	tx, err = s.db.Begin(ctx)
	if err != nil {
		log.Printf("begin patch ticket transaction failed: %v", err)
		http.Error(w, "failed to update ticket", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	switch input.Status {
	case "assigned":
		if input.Description == "" {
			http.Error(w, "description is required", http.StatusBadRequest)
			return
		}
		if input.Reason == "" {
			http.Error(w, "reason is required", http.StatusBadRequest)
			return
		}
		if input.ContactPerson == "" {
			http.Error(w, "contact_person is required", http.StatusBadRequest)
			return
		}
		if input.Executor == "" {
			http.Error(w, "executor is required", http.StatusBadRequest)
			return
		}
		if input.AssignedStart == "" {
			http.Error(w, "assigned_start is required", http.StatusBadRequest)
			return
		}
		if input.AssignedEnd == "" {
			http.Error(w, "assigned_end is required", http.StatusBadRequest)
			return
		}

		assignedStart, parseErr := parseTicketDateInput(input.AssignedStart)
		if parseErr != nil {
			http.Error(w, "assigned_start must be a date or ISO timestamp", http.StatusBadRequest)
			return
		}

		assignedEnd, parseErr := parseTicketDateInput(input.AssignedEnd)
		if parseErr != nil {
			http.Error(w, "assigned_end must be a date or ISO timestamp", http.StatusBadRequest)
			return
		}

		if assignedStart.After(assignedEnd) {
			http.Error(w, "assigned_end must be greater than or equal to assigned_start", http.StatusBadRequest)
			return
		}

		var executorID pgtype.UUID
		if err := executorID.Scan(input.Executor); err != nil {
			http.Error(w, "executor must be a valid UUID", http.StatusBadRequest)
			return
		}
		var contactPersonID pgtype.UUID
		if err := contactPersonID.Scan(input.ContactPerson); err != nil {
			http.Error(w, "contact_person must be a valid UUID", http.StatusBadRequest)
			return
		}

		var (
			requesterDepartment pgtype.UUID
			requesterRole       string
		)
		if err := s.db.QueryRow(ctx, `
			SELECT
				u.department,
				COALESCE(r.name, 'user')
			FROM users u
			LEFT JOIN account_roles ar ON ar.user_id = u.user_id
			LEFT JOIN roles r ON r.id = ar.role_id
			WHERE u.user_id = $1
		`, userID).Scan(&requesterDepartment, &requesterRole); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "profile not found", http.StatusNotFound)
				return
			}

			log.Printf("load requester profile failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}

		if requesterRole != "coordinator" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !requesterDepartment.Valid {
			http.Error(w, "requester department is required", http.StatusBadRequest)
			return
		}

		var reasonExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ticket_reasons WHERE id = $1)`, input.Reason).Scan(&reasonExists); err != nil {
			log.Printf("validate ticket reason failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
		if !reasonExists {
			http.Error(w, "reason not found", http.StatusBadRequest)
			return
		}

		var contactExists bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM tickets t
				JOIN contacts c ON c.client_id = t.client
				WHERE t.id = $1
				  AND c.id = $2
			)
		`, ticketID, contactPersonID).Scan(&contactExists); err != nil {
			log.Printf("validate contact failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
		if !contactExists {
			http.Error(w, "contact_person not found", http.StatusBadRequest)
			return
		}

		var executorExists bool
		if err := s.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM users u
				JOIN accounts a ON a.user_id = u.user_id
				WHERE u.user_id = $1
				  AND u.department = $2
				  AND a.disabled = FALSE
			)
		`, executorID, requesterDepartment).Scan(&executorExists); err != nil {
			log.Printf("validate executor failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
		if !executorExists {
			http.Error(w, "executor not found in requester department", http.StatusBadRequest)
			return
		}

		expectedCurrentStatus = "created"
		conflictMessage = "ticket must belong to your department and be in created status"
		result, err = tx.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				description = $2,
				reason = $3,
				contact_person = $4,
				executor = $5,
				urgent = $6,
				assigned_by = $7,
				assigned_at = (NOW() AT TIME ZONE 'UTC'),
				assigned_start = $8,
				assigned_end = $9
			WHERE id = $10
			  AND department = $11
			  AND status = $12
		`, input.Status, input.Description, input.Reason, contactPersonID, executorID, input.Urgent, userID, assignedStart, assignedEnd, ticketID, requesterDepartment, expectedCurrentStatus)
		if err == nil {
			latestTicketUserID = &executorID
			formattedAssignedAt := time.Now().UTC().Format(time.RFC3339)
			formattedAssignedStart := assignedStart.UTC().Format(time.RFC3339)
			formattedAssignedEnd := assignedEnd.UTC().Format(time.RFC3339)
			response.AssignedAt = &formattedAssignedAt
			response.AssignedStart = &formattedAssignedStart
			response.AssignedEnd = &formattedAssignedEnd
			response.ContactPerson = &input.ContactPerson
			response.Description = input.Description
			response.Executor = &input.Executor
			response.Reason = input.Reason
			response.Urgent = &input.Urgent
		}
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
		result, err = tx.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				workstarted_at = $2
			WHERE id = $3
			  AND executor = $4
			  AND status = $5
		`, input.Status, workstartedAt.UTC(), ticketID, userID, expectedCurrentStatus)
		if err == nil {
			latestTicketUserID = &userID
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
		result, err = tx.Exec(ctx, `
			UPDATE tickets
			SET status = $1,
				workfinished_at = $2
			WHERE id = $3
			  AND executor = $4
			  AND status = $5
		`, input.Status, workfinishedAt.UTC(), ticketID, userID, expectedCurrentStatus)
		if err == nil {
			latestTicketUserID = &userID
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
			latestTicketUserID = &userID
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
		http.Error(w, "supported transitions: created->assigned, assigned->inWork, inWork->worksDone, worksDone->closed", http.StatusBadRequest)
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

		http.Error(w, conflictMessage, http.StatusConflict)
		return
	}

	if latestTicketUserID != nil {
		if err := updateLatestTicketReference(ctx, tx, *latestTicketUserID, ticketID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "profile not found", http.StatusNotFound)
				return
			}

			log.Printf("update latest ticket reference failed: %v", err)
			http.Error(w, "failed to update ticket", http.StatusInternalServerError)
			return
		}
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
		ExternalAuthor     *string `json:"externalAuthor"`
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
			t.external_author,
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
			externalAuthor pgtype.UUID
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
			&externalAuthor,
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
			ExternalAuthor:     nullableUUIDToString(externalAuthor),
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

func nullableUUID(value pgtype.UUID) any {
	if !value.Valid {
		return nil
	}

	return value
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
