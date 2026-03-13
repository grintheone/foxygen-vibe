package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"foxygen-vibe/server/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxTicketAttachmentUploadSize = 25 << 20

type ticketAttachmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	MediaType   string  `json:"mediaType"`
	Ext         string  `json:"ext"`
	SizeBytes   int64   `json:"sizeBytes"`
	UploadedAt  *string `json:"uploadedAt,omitempty"`
	DownloadURL string  `json:"downloadUrl,omitempty"`
}

func buildTicketAttachmentDownloadURL(ticketID string, attachmentID string) string {
	return fmt.Sprintf("/api/tickets/%s/attachments/%s/download", ticketID, attachmentID)
}

func (s *Server) loadTicketAttachments(ctx context.Context, ticketID pgtype.UUID) ([]ticketAttachmentResponse, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, media_type, ext, size_bytes, uploaded_at
		FROM attachments
		WHERE ref_id = $1
		ORDER BY uploaded_at ASC, id ASC
	`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]ticketAttachmentResponse, 0)
	for rows.Next() {
		var (
			id         string
			name       string
			mediaType  string
			ext        string
			sizeBytes  int64
			uploadedAt pgtype.Timestamptz
		)

		if err := rows.Scan(&id, &name, &mediaType, &ext, &sizeBytes, &uploadedAt); err != nil {
			return nil, err
		}

		attachment := ticketAttachmentResponse{
			ID:          id,
			Name:        name,
			MediaType:   mediaType,
			Ext:         ext,
			SizeBytes:   sizeBytes,
			UploadedAt:  timestamptzToRFC3339(uploadedAt),
			DownloadURL: buildTicketAttachmentDownloadURL(ticketID.String(), id),
		}

		attachments = append(attachments, attachment)
	}

	return attachments, rows.Err()
}

func (s *Server) handleTicketAttachmentUpload(w http.ResponseWriter, r *http.Request, ticketID pgtype.UUID, userID pgtype.UUID) {
	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	if s.storage == nil {
		http.Error(w, "object storage not configured", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxTicketAttachmentUploadSize)

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileName := strings.TrimSpace(fileHeader.Filename)
	if fileName == "" {
		http.Error(w, "file name is required", http.StatusBadRequest)
		return
	}
	if fileHeader.Size <= 0 {
		http.Error(w, "file must not be empty", http.StatusBadRequest)
		return
	}

	mediaType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	canUpload, ticketExists, err := s.canUploadTicketAttachment(ctx, ticketID, userID)
	if err != nil {
		log.Printf("check upload permissions failed: %v", err)
		http.Error(w, "failed to upload attachment", http.StatusInternalServerError)
		return
	}
	if !ticketExists {
		http.Error(w, "ticket not found", http.StatusNotFound)
		return
	}
	if !canUpload {
		http.Error(w, "ticket must be assigned to you and in worksDone or closed status", http.StatusConflict)
		return
	}

	var attachmentID string
	if err := s.db.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&attachmentID); err != nil {
		log.Printf("generate attachment id failed: %v", err)
		http.Error(w, "failed to upload attachment", http.StatusInternalServerError)
		return
	}

	objectKey := storage.TicketAttachmentObjectKey(ticketID.String(), attachmentID, fileName)
	uploaded, err := s.storage.PutObject(ctx, objectKey, file, fileHeader.Size, mediaType)
	if err != nil {
		log.Printf("upload attachment to MinIO failed: %v", err)
		http.Error(w, "failed to upload attachment", http.StatusBadGateway)
		return
	}

	var uploadedAt pgtype.Timestamptz
	if err := s.db.QueryRow(ctx, `
		INSERT INTO attachments (id, name, media_type, ext, ref_id, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING uploaded_at
	`, attachmentID, fileName, mediaType, ext, ticketID, uploaded.Size).Scan(&uploadedAt); err != nil {
		if cleanupErr := s.storage.RemoveObject(context.Background(), objectKey); cleanupErr != nil {
			log.Printf("cleanup orphaned attachment object failed: %v", cleanupErr)
		}

		log.Printf("insert attachment failed: %v", err)
		http.Error(w, "failed to upload attachment", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, ticketAttachmentResponse{
		ID:          attachmentID,
		Name:        fileName,
		MediaType:   mediaType,
		Ext:         ext,
		SizeBytes:   uploaded.Size,
		UploadedAt:  timestamptzToRFC3339(uploadedAt),
		DownloadURL: buildTicketAttachmentDownloadURL(ticketID.String(), attachmentID),
	})
}

func (s *Server) handleTicketAttachmentDownload(w http.ResponseWriter, r *http.Request, ticketID pgtype.UUID, attachmentID string) {
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

	var (
		name      string
		mediaType string
	)

	err := s.db.QueryRow(ctx, `
		SELECT a.name, a.media_type
		FROM attachments a
		JOIN tickets t ON t.id = a.ref_id
		WHERE a.ref_id = $1
		  AND a.id = $2
	`, ticketID, attachmentID).Scan(&name, &mediaType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "attachment not found", http.StatusNotFound)
			return
		}

		log.Printf("load attachment failed: %v", err)
		http.Error(w, "failed to download attachment", http.StatusInternalServerError)
		return
	}

	object, info, err := s.storage.GetObject(ctx, attachmentID)
	if err != nil {
		log.Printf("download attachment from MinIO failed: %v", err)
		http.Error(w, "failed to download attachment", http.StatusBadGateway)
		return
	}
	defer object.Close()

	if mediaType == "" {
		mediaType = strings.TrimSpace(info.ContentType)
	}
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	fileName := strings.TrimSpace(name)
	if fileName == "" {
		fileName = attachmentID
	}

	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fileName}))
	w.Header().Set("Content-Type", mediaType)
	if info.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	}

	if _, err := io.Copy(w, object); err != nil {
		log.Printf("stream attachment failed: %v", err)
	}
}

func (s *Server) canUploadTicketAttachment(ctx context.Context, ticketID pgtype.UUID, userID pgtype.UUID) (bool, bool, error) {
	var status string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(status, '')
		FROM tickets
		WHERE id = $1
		  AND executor = $2
	`, ticketID, userID).Scan(&status)
	if err == nil {
		return status == "worksDone" || status == "closed", true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, false, err
	}

	var ticketExists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tickets WHERE id = $1)`, ticketID).Scan(&ticketExists); err != nil {
		return false, false, err
	}

	return false, ticketExists, nil
}
