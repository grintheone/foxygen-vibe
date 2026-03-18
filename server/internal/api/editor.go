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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultEditorClientListLimit = 50
	maxEditorClientListLimit     = 100
)

type editorClientListItemResponse struct {
	ActiveAgreementCount int    `json:"activeAgreementCount"`
	Address              string `json:"address"`
	ContactCount         int    `json:"contactCount"`
	ID                   string `json:"id"`
	RegionTitle          string `json:"regionTitle"`
	Title                string `json:"title"`
}

type editorClientDetailResponse struct {
	ActiveAgreementCount int             `json:"activeAgreementCount"`
	Address              string          `json:"address"`
	ContactCount         int             `json:"contactCount"`
	ID                   string          `json:"id"`
	LaboratorySystem     *string         `json:"laboratorySystem"`
	Location             json.RawMessage `json:"location"`
	Manager              []string        `json:"manager"`
	Region               *string         `json:"region"`
	RegionTitle          string          `json:"regionTitle"`
	Title                string          `json:"title"`
}

type editorRegionResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (s *Server) handleEditorClients(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/editor/clients" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := s.requireEditorAccess(w, r); !ok {
		return
	}

	limit := defaultEditorClientListLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		if parsedLimit > maxEditorClientListLimit {
			parsedLimit = maxEditorClientListLimit
		}
		limit = parsedLimit
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT
			c.id,
			COALESCE(c.title, ''),
			COALESCE(c.address, ''),
			COALESCE(r.title, ''),
			(
				SELECT COUNT(*)
				FROM contacts ct
				WHERE ct.client_id = c.id
			) AS contact_count,
			(
				SELECT COUNT(*)
				FROM agreements a
				WHERE a.actual_client = c.id
				  AND a.is_active = TRUE
			) AS active_agreement_count
		FROM clients c
		LEFT JOIN regions r ON r.id = c.region
		WHERE (
			$1 = ''
			OR COALESCE(c.title, '') ILIKE '%' || $1 || '%'
			OR COALESCE(c.address, '') ILIKE '%' || $1 || '%'
		)
		ORDER BY
			CASE WHEN COALESCE(c.title, '') = '' THEN 1 ELSE 0 END ASC,
			COALESCE(c.title, '') ASC,
			c.id ASC
		LIMIT $2
	`, query, limit)
	if err != nil {
		log.Printf("query editor clients failed: %v", err)
		http.Error(w, "failed to load editor clients", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]editorClientListItemResponse, 0)
	for rows.Next() {
		var (
			id                   pgtype.UUID
			title                string
			address              string
			regionTitle          string
			contactCount         int
			activeAgreementCount int
		)

		if err := rows.Scan(&id, &title, &address, &regionTitle, &contactCount, &activeAgreementCount); err != nil {
			log.Printf("scan editor client list item failed: %v", err)
			http.Error(w, "failed to load editor clients", http.StatusInternalServerError)
			return
		}

		response = append(response, editorClientListItemResponse{
			ActiveAgreementCount: activeAgreementCount,
			Address:              address,
			ContactCount:         contactCount,
			ID:                   uuidToString(id),
			RegionTitle:          regionTitle,
			Title:                title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate editor clients failed: %v", err)
		http.Error(w, "failed to load editor clients", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorRegions(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/editor/regions" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, ok := s.requireEditorAccess(w, r); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(title, '')
		FROM regions
		ORDER BY title ASC, id ASC
	`)
	if err != nil {
		log.Printf("query editor regions failed: %v", err)
		http.Error(w, "failed to load regions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]editorRegionResponse, 0)
	for rows.Next() {
		var (
			id    pgtype.UUID
			title string
		)

		if err := rows.Scan(&id, &title); err != nil {
			log.Printf("scan editor region failed: %v", err)
			http.Error(w, "failed to load regions", http.StatusInternalServerError)
			return
		}

		response = append(response, editorRegionResponse{
			ID:    uuidToString(id),
			Title: title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate editor regions failed: %v", err)
		http.Error(w, "failed to load regions", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorClientByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireEditorAccess(w, r); !ok {
		return
	}

	clientPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/editor/clients/"), "/")
	if clientPath == "" || strings.Contains(clientPath, "/") {
		http.NotFound(w, r)
		return
	}

	var clientID pgtype.UUID
	if err := clientID.Scan(clientPath); err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleEditorClientDetail(w, r, clientID)
	case http.MethodPatch:
		s.handleEditorClientPatch(w, r, clientID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEditorClientDetail(w http.ResponseWriter, r *http.Request, clientID pgtype.UUID) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	response, found, err := s.loadEditorClientDetail(ctx, clientID)
	if err != nil {
		log.Printf("load editor client detail failed: %v", err)
		http.Error(w, "failed to load client", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorClientPatch(w http.ResponseWriter, r *http.Request, clientID pgtype.UUID) {
	type patchEditorClientRequest struct {
		Address string `json:"address"`
		Region  string `json:"region"`
		Title   string `json:"title"`
	}

	defer r.Body.Close()

	var input patchEditorClientRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	input.Address = strings.TrimSpace(input.Address)
	input.Region = strings.TrimSpace(input.Region)

	if input.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var regionValue any = nil
	if input.Region != "" {
		var regionID pgtype.UUID
		if err := regionID.Scan(input.Region); err != nil {
			http.Error(w, "region must be a valid UUID", http.StatusBadRequest)
			return
		}

		var regionExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM regions WHERE id = $1)`, regionID).Scan(&regionExists); err != nil {
			log.Printf("validate editor client region failed: %v", err)
			http.Error(w, "failed to update client", http.StatusInternalServerError)
			return
		}
		if !regionExists {
			http.Error(w, "region not found", http.StatusBadRequest)
			return
		}

		regionValue = regionID
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE clients
		SET title = $1,
			address = $2,
			region = $3
		WHERE id = $4
	`, input.Title, input.Address, regionValue, clientID)
	if err != nil {
		log.Printf("update editor client failed: %v", err)
		http.Error(w, "failed to update client", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	response, found, err := s.loadEditorClientDetail(ctx, clientID)
	if err != nil {
		log.Printf("reload editor client failed: %v", err)
		http.Error(w, "failed to update client", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) loadEditorClientDetail(ctx context.Context, clientID pgtype.UUID) (editorClientDetailResponse, bool, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			c.id,
			COALESCE(c.title, ''),
			COALESCE(c.address, ''),
			COALESCE(c.location, '{}'::jsonb),
			c.region,
			COALESCE(r.title, ''),
			c.laboratory_system,
			c.manager,
			(
				SELECT COUNT(*)
				FROM contacts ct
				WHERE ct.client_id = c.id
			) AS contact_count,
			(
				SELECT COUNT(*)
				FROM agreements a
				WHERE a.actual_client = c.id
				  AND a.is_active = TRUE
			) AS active_agreement_count
		FROM clients c
		LEFT JOIN regions r ON r.id = c.region
		WHERE c.id = $1
		LIMIT 1
	`, clientID)

	var (
		id                   pgtype.UUID
		title                string
		address              string
		location             []byte
		region               pgtype.UUID
		regionTitle          string
		laboratorySystem     pgtype.UUID
		manager              []pgtype.UUID
		contactCount         int
		activeAgreementCount int
	)

	if err := row.Scan(
		&id,
		&title,
		&address,
		&location,
		&region,
		&regionTitle,
		&laboratorySystem,
		&manager,
		&contactCount,
		&activeAgreementCount,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return editorClientDetailResponse{}, false, nil
		}

		return editorClientDetailResponse{}, false, err
	}

	return editorClientDetailResponse{
		ActiveAgreementCount: activeAgreementCount,
		Address:              address,
		ContactCount:         contactCount,
		ID:                   uuidToString(id),
		LaboratorySystem:     nullableUUIDToString(laboratorySystem),
		Location:             json.RawMessage(location),
		Manager:              uuidSliceToString(manager),
		Region:               nullableUUIDToString(region),
		RegionTitle:          regionTitle,
		Title:                title,
	}, true, nil
}

func (s *Server) requireEditorAccess(w http.ResponseWriter, r *http.Request) (pgtype.UUID, bool) {
	claims, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return pgtype.UUID{}, false
	}

	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return pgtype.UUID{}, false
	}

	var requesterID pgtype.UUID
	if err := requesterID.Scan(claims.Subject); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return pgtype.UUID{}, false
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var role string
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(r.name, 'user')
		FROM accounts a
		LEFT JOIN account_roles ar ON ar.user_id = a.user_id
		LEFT JOIN roles r ON r.id = ar.role_id
		WHERE a.user_id = $1
		LIMIT 1
	`, requesterID).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "profile not found", http.StatusNotFound)
			return pgtype.UUID{}, false
		}

		log.Printf("load editor requester role failed: %v", err)
		http.Error(w, "failed to verify editor access", http.StatusInternalServerError)
		return pgtype.UUID{}, false
	}

	if role != "admin" && role != "coordinator" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return pgtype.UUID{}, false
	}

	return requesterID, true
}
