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

type editorContactListItemResponse struct {
	Client     *string `json:"client"`
	ClientName string  `json:"clientName"`
	Email      string  `json:"email"`
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	Position   string  `json:"position"`
}

type editorContactDetailResponse struct {
	Client     *string `json:"client"`
	ClientName string  `json:"clientName"`
	Email      string  `json:"email"`
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	Position   string  `json:"position"`
}

type editorDeviceListItemResponse struct {
	Agreement         *string `json:"agreement"`
	AgreementNumber   *int32  `json:"agreementNumber"`
	Client            *string `json:"client"`
	ClientName        string  `json:"clientName"`
	ConnectedToLis    bool    `json:"connectedToLis"`
	ID                string  `json:"id"`
	IsActiveAgreement bool    `json:"isActiveAgreement"`
	IsUsed            bool    `json:"isUsed"`
	SerialNumber      string  `json:"serialNumber"`
	Title             string  `json:"title"`
}

type editorDeviceDetailResponse struct {
	Agreement         *string         `json:"agreement"`
	AgreementNumber   *int32          `json:"agreementNumber"`
	Classificator     *string         `json:"classificator"`
	Client            *string         `json:"client"`
	ClientAddress     string          `json:"clientAddress"`
	ClientName        string          `json:"clientName"`
	ConnectedToLis    bool            `json:"connectedToLis"`
	ID                string          `json:"id"`
	IsActiveAgreement bool            `json:"isActiveAgreement"`
	IsUsed            bool            `json:"isUsed"`
	OnWarranty        bool            `json:"onWarranty"`
	Properties        json.RawMessage `json:"properties"`
	SerialNumber      string          `json:"serialNumber"`
	Title             string          `json:"title"`
}

type editorClassificatorResponse struct {
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

func (s *Server) handleEditorContacts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/editor/contacts" {
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
			ct.id,
			COALESCE(ct.name, ''),
			COALESCE(ct.position, ''),
			COALESCE(ct.phone, ''),
			COALESCE(ct.email, ''),
			ct.client_id,
			COALESCE(c.title, '')
		FROM contacts ct
		LEFT JOIN clients c ON c.id = ct.client_id
		WHERE (
			$1 = ''
			OR COALESCE(ct.name, '') ILIKE '%' || $1 || '%'
			OR COALESCE(ct.position, '') ILIKE '%' || $1 || '%'
			OR COALESCE(ct.phone, '') ILIKE '%' || $1 || '%'
			OR COALESCE(ct.email, '') ILIKE '%' || $1 || '%'
			OR COALESCE(c.title, '') ILIKE '%' || $1 || '%'
		)
		ORDER BY
			CASE WHEN COALESCE(ct.name, '') = '' THEN 1 ELSE 0 END ASC,
			COALESCE(ct.name, '') ASC,
			ct.id ASC
		LIMIT $2
	`, query, limit)
	if err != nil {
		log.Printf("query editor contacts failed: %v", err)
		http.Error(w, "failed to load editor contacts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]editorContactListItemResponse, 0)
	for rows.Next() {
		var (
			id         pgtype.UUID
			name       string
			position   string
			phone      string
			email      string
			clientID   pgtype.UUID
			clientName string
		)

		if err := rows.Scan(&id, &name, &position, &phone, &email, &clientID, &clientName); err != nil {
			log.Printf("scan editor contact failed: %v", err)
			http.Error(w, "failed to load editor contacts", http.StatusInternalServerError)
			return
		}

		response = append(response, editorContactListItemResponse{
			Client:     nullableUUIDToString(clientID),
			ClientName: clientName,
			Email:      email,
			ID:         uuidToString(id),
			Name:       name,
			Phone:      phone,
			Position:   position,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate editor contacts failed: %v", err)
		http.Error(w, "failed to load editor contacts", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorDevices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/editor/devices" {
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
			d.id,
			COALESCE(cls.title, ''),
			COALESCE(d.serial_number, ''),
			d.connected_to_lis,
			d.is_used,
			a.id,
			a.number,
			COALESCE(a.is_active, FALSE),
			c.id,
			COALESCE(c.title, '')
		FROM devices d
		LEFT JOIN classificators cls ON cls.id = d.classificator
		LEFT JOIN LATERAL (
			SELECT
				a.id,
				a.number,
				a.is_active,
				a.actual_client
			FROM agreements a
			WHERE a.device = d.id
			ORDER BY a.is_active DESC, a.assigned_at DESC NULLS LAST, a.number DESC
			LIMIT 1
		) a ON TRUE
		LEFT JOIN clients c ON c.id = a.actual_client
		WHERE (
			$1 = ''
			OR COALESCE(cls.title, '') ILIKE '%' || $1 || '%'
			OR COALESCE(d.serial_number, '') ILIKE '%' || $1 || '%'
			OR COALESCE(c.title, '') ILIKE '%' || $1 || '%'
		)
		ORDER BY
			CASE WHEN COALESCE(cls.title, '') = '' THEN 1 ELSE 0 END ASC,
			COALESCE(cls.title, '') ASC,
			CASE WHEN COALESCE(d.serial_number, '') = '' THEN 1 ELSE 0 END ASC,
			COALESCE(d.serial_number, '') ASC,
			d.id ASC
		LIMIT $2
	`, query, limit)
	if err != nil {
		log.Printf("query editor devices failed: %v", err)
		http.Error(w, "failed to load editor devices", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]editorDeviceListItemResponse, 0)
	for rows.Next() {
		var (
			id                pgtype.UUID
			title             string
			serialNumber      string
			connectedToLis    bool
			isUsed            bool
			agreementID       pgtype.UUID
			agreementNumber   pgtype.Int4
			isActiveAgreement bool
			clientID          pgtype.UUID
			clientName        string
		)

		if err := rows.Scan(
			&id,
			&title,
			&serialNumber,
			&connectedToLis,
			&isUsed,
			&agreementID,
			&agreementNumber,
			&isActiveAgreement,
			&clientID,
			&clientName,
		); err != nil {
			log.Printf("scan editor device failed: %v", err)
			http.Error(w, "failed to load editor devices", http.StatusInternalServerError)
			return
		}

		response = append(response, editorDeviceListItemResponse{
			Agreement:         nullableUUIDToString(agreementID),
			AgreementNumber:   nullableInt32ToPointer(agreementNumber),
			Client:            nullableUUIDToString(clientID),
			ClientName:        clientName,
			ConnectedToLis:    connectedToLis,
			ID:                uuidToString(id),
			IsActiveAgreement: isActiveAgreement,
			IsUsed:            isUsed,
			SerialNumber:      serialNumber,
			Title:             title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate editor devices failed: %v", err)
		http.Error(w, "failed to load editor devices", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorClassificators(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/editor/classificators" {
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
		FROM classificators
		ORDER BY
			CASE WHEN COALESCE(title, '') = '' THEN 1 ELSE 0 END ASC,
			COALESCE(title, '') ASC,
			id ASC
	`)
	if err != nil {
		log.Printf("query editor classificators failed: %v", err)
		http.Error(w, "failed to load classificators", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make([]editorClassificatorResponse, 0)
	for rows.Next() {
		var (
			id    pgtype.UUID
			title string
		)

		if err := rows.Scan(&id, &title); err != nil {
			log.Printf("scan editor classificator failed: %v", err)
			http.Error(w, "failed to load classificators", http.StatusInternalServerError)
			return
		}

		response = append(response, editorClassificatorResponse{
			ID:    uuidToString(id),
			Title: title,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate editor classificators failed: %v", err)
		http.Error(w, "failed to load classificators", http.StatusInternalServerError)
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

func (s *Server) handleEditorContactByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireEditorAccess(w, r); !ok {
		return
	}

	contactPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/editor/contacts/"), "/")
	if contactPath == "" || strings.Contains(contactPath, "/") {
		http.NotFound(w, r)
		return
	}

	var contactID pgtype.UUID
	if err := contactID.Scan(contactPath); err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleEditorContactDetail(w, r, contactID)
	case http.MethodPatch:
		s.handleEditorContactPatch(w, r, contactID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEditorDeviceByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireEditorAccess(w, r); !ok {
		return
	}

	devicePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/editor/devices/"), "/")
	if devicePath == "" || strings.Contains(devicePath, "/") {
		http.NotFound(w, r)
		return
	}

	var deviceID pgtype.UUID
	if err := deviceID.Scan(devicePath); err != nil {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleEditorDeviceDetail(w, r, deviceID)
	case http.MethodPatch:
		s.handleEditorDevicePatch(w, r, deviceID)
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
		Address  string `json:"address"`
		Location string `json:"location"`
		Region   string `json:"region"`
		Title    string `json:"title"`
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
	input.Location = strings.TrimSpace(input.Location)
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

	var locationValue any = nil
	if input.Location != "" {
		rawLocation := json.RawMessage(input.Location)
		if !json.Valid(rawLocation) {
			http.Error(w, "location must be valid JSON", http.StatusBadRequest)
			return
		}

		locationValue = rawLocation
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE clients
		SET title = $1,
			address = $2,
			region = $3,
			location = $4
		WHERE id = $5
	`, input.Title, input.Address, regionValue, locationValue, clientID)
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

func (s *Server) handleEditorContactDetail(w http.ResponseWriter, r *http.Request, contactID pgtype.UUID) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	response, found, err := s.loadEditorContactDetail(ctx, contactID)
	if err != nil {
		log.Printf("load editor contact detail failed: %v", err)
		http.Error(w, "failed to load contact", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "contact not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorContactPatch(w http.ResponseWriter, r *http.Request, contactID pgtype.UUID) {
	type patchEditorContactRequest struct {
		Client   string `json:"client"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
		Position string `json:"position"`
	}

	defer r.Body.Close()

	var input patchEditorContactRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Client = strings.TrimSpace(input.Client)
	input.Email = strings.TrimSpace(input.Email)
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Position = strings.TrimSpace(input.Position)

	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Client == "" {
		http.Error(w, "client is required", http.StatusBadRequest)
		return
	}

	var clientID pgtype.UUID
	if err := clientID.Scan(input.Client); err != nil {
		http.Error(w, "client must be a valid UUID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var clientExists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM clients WHERE id = $1)`, clientID).Scan(&clientExists); err != nil {
		log.Printf("validate editor contact client failed: %v", err)
		http.Error(w, "failed to update contact", http.StatusInternalServerError)
		return
	}
	if !clientExists {
		http.Error(w, "client not found", http.StatusBadRequest)
		return
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE contacts
		SET name = $1,
			position = $2,
			phone = $3,
			email = $4,
			client_id = $5
		WHERE id = $6
	`, input.Name, input.Position, input.Phone, input.Email, clientID, contactID)
	if err != nil {
		log.Printf("update editor contact failed: %v", err)
		http.Error(w, "failed to update contact", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "contact not found", http.StatusNotFound)
		return
	}

	response, found, err := s.loadEditorContactDetail(ctx, contactID)
	if err != nil {
		log.Printf("reload editor contact failed: %v", err)
		http.Error(w, "failed to update contact", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "contact not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorDeviceDetail(w http.ResponseWriter, r *http.Request, deviceID pgtype.UUID) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	response, found, err := s.loadEditorDeviceDetail(ctx, deviceID)
	if err != nil {
		log.Printf("load editor device detail failed: %v", err)
		http.Error(w, "failed to load device", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleEditorDevicePatch(w http.ResponseWriter, r *http.Request, deviceID pgtype.UUID) {
	type patchEditorDeviceRequest struct {
		Classificator  string `json:"classificator"`
		ConnectedToLis bool   `json:"connectedToLis"`
		IsUsed         bool   `json:"isUsed"`
		Properties     string `json:"properties"`
		SerialNumber   string `json:"serialNumber"`
	}

	defer r.Body.Close()

	var input patchEditorDeviceRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Classificator = strings.TrimSpace(input.Classificator)
	input.Properties = strings.TrimSpace(input.Properties)
	input.SerialNumber = strings.TrimSpace(input.SerialNumber)

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var classificatorValue any = nil
	if input.Classificator != "" {
		var classificatorID pgtype.UUID
		if err := classificatorID.Scan(input.Classificator); err != nil {
			http.Error(w, "classificator must be a valid UUID", http.StatusBadRequest)
			return
		}

		var classificatorExists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM classificators WHERE id = $1)`, classificatorID).Scan(&classificatorExists); err != nil {
			log.Printf("validate editor device classificator failed: %v", err)
			http.Error(w, "failed to update device", http.StatusInternalServerError)
			return
		}
		if !classificatorExists {
			http.Error(w, "classificator not found", http.StatusBadRequest)
			return
		}

		classificatorValue = classificatorID
	}

	propertiesValue := input.Properties
	if propertiesValue == "" {
		propertiesValue = "{}"
	}
	rawProperties := json.RawMessage(propertiesValue)
	if !json.Valid(rawProperties) {
		http.Error(w, "properties must be valid JSON", http.StatusBadRequest)
		return
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE devices
		SET classificator = $1,
			serial_number = $2,
			properties = $3,
			connected_to_lis = $4,
			is_used = $5
		WHERE id = $6
	`, classificatorValue, input.SerialNumber, rawProperties, input.ConnectedToLis, input.IsUsed, deviceID)
	if err != nil {
		log.Printf("update editor device failed: %v", err)
		http.Error(w, "failed to update device", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	response, found, err := s.loadEditorDeviceDetail(ctx, deviceID)
	if err != nil {
		log.Printf("reload editor device failed: %v", err)
		http.Error(w, "failed to update device", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "device not found", http.StatusNotFound)
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

func (s *Server) loadEditorContactDetail(ctx context.Context, contactID pgtype.UUID) (editorContactDetailResponse, bool, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			ct.id,
			COALESCE(ct.name, ''),
			COALESCE(ct.position, ''),
			COALESCE(ct.phone, ''),
			COALESCE(ct.email, ''),
			ct.client_id,
			COALESCE(c.title, '')
		FROM contacts ct
		LEFT JOIN clients c ON c.id = ct.client_id
		WHERE ct.id = $1
		LIMIT 1
	`, contactID)

	var (
		id         pgtype.UUID
		name       string
		position   string
		phone      string
		email      string
		clientID   pgtype.UUID
		clientName string
	)

	if err := row.Scan(&id, &name, &position, &phone, &email, &clientID, &clientName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return editorContactDetailResponse{}, false, nil
		}

		return editorContactDetailResponse{}, false, err
	}

	return editorContactDetailResponse{
		Client:     nullableUUIDToString(clientID),
		ClientName: clientName,
		Email:      email,
		ID:         uuidToString(id),
		Name:       name,
		Phone:      phone,
		Position:   position,
	}, true, nil
}

func (s *Server) loadEditorDeviceDetail(ctx context.Context, deviceID pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			d.id,
			d.classificator,
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
		classificatorID   pgtype.UUID
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
		&classificatorID,
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
			return editorDeviceDetailResponse{}, false, nil
		}

		return editorDeviceDetailResponse{}, false, err
	}

	return editorDeviceDetailResponse{
		Agreement:         nullableUUIDToString(agreementID),
		AgreementNumber:   nullableInt32ToPointer(agreementNumber),
		Classificator:     nullableUUIDToString(classificatorID),
		Client:            nullableUUIDToString(clientID),
		ClientAddress:     clientAddress,
		ClientName:        clientName,
		ConnectedToLis:    connectedToLis,
		ID:                uuidToString(id),
		IsActiveAgreement: isActiveAgreement,
		IsUsed:            isUsed,
		OnWarranty:        onWarranty,
		Properties:        json.RawMessage(properties),
		SerialNumber:      serialNumber,
		Title:             title,
	}, true, nil
}

func nullableInt32ToPointer(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}

	result := value.Int32
	return &result
}

func nullableTextToPointer(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	result := value.String
	return &result
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
