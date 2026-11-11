package api

import (
	"bytes"
	"context"
	"crypto/sha256"
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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	if s.ticketAttachmentsLoader != nil {
		return s.ticketAttachmentsLoader(ctx, ticketID)
	}
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

	r.Body = http.MaxBytesReader(w, r.Body, maxTicketAttachmentUploadSize+(1<<20))

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

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
		return
	}
	if len(fileBytes) == 0 {
		http.Error(w, "file must not be empty", http.StatusBadRequest)
		return
	}
	if len(fileBytes) > maxTicketAttachmentUploadSize {
		http.Error(w, "attachment must not exceed 25 MB", http.StatusRequestEntityTooLarge)
		return
	}

	mediaType, ok := detectSupportedImageUploadMediaType(fileBytes)
	if !ok {
		http.Error(w, "only supported image formats can be uploaded", http.StatusUnsupportedMediaType)
		return
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to upload attachment", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())
	digest := fmt.Sprintf("%x", sha256.Sum256(fileBytes))
	attachment, status, err := uploadTicketAttachment(ctx, tx, ticketID, userID, fileName, mediaType, ext, digest,
		func(ctx context.Context, attachmentID string) (int64, error) {
			objectKey := storage.TicketAttachmentObjectKey(ticketID.String(), attachmentID, fileName)
			uploaded, err := s.storage.PutObject(ctx, objectKey, bytes.NewReader(fileBytes), int64(len(fileBytes)), mediaType)
			return uploaded.Size, err
		}, s.storage.RemoveObject)
	if err != nil {
		log.Printf("upload ticket attachment failed: %v", err)
		message := "failed to upload attachment"
		if status == http.StatusNotFound || status == http.StatusConflict {
			message = err.Error()
		}
		http.Error(w, message, status)
		return
	}

	// A retry also retries any undelivered event; delivery receipts exclude files
	// already accepted by the sidecar.
	s.publishTicketUpdated(ticketID)
	writeJSON(w, status, attachment)
}

// The ticket lock serializes identical uploads (including requests handled by
// different API instances) and deletion. A retry returns the original metadata.
func uploadTicketAttachment(ctx context.Context, tx pgx.Tx, ticketID, userID pgtype.UUID,
	name, mediaType, ext, digest string, put func(context.Context, string) (int64, error), remove func(context.Context, string) error,
) (ticketAttachmentResponse, int, error) {
	var attachment ticketAttachmentResponse
	var allowed bool
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(executor = $2, false) AND COALESCE(status, '') IN ('worksDone', 'closed')
		FROM tickets WHERE id = $1 FOR UPDATE
	`, ticketID, userID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return attachment, http.StatusNotFound, fmt.Errorf("ticket not found")
	}
	if err != nil {
		return attachment, http.StatusInternalServerError, err
	}
	if !allowed {
		return attachment, http.StatusConflict, fmt.Errorf("ticket must be assigned to you and in worksDone or closed status")
	}

	var uploadedAt pgtype.Timestamptz
	err = tx.QueryRow(ctx, `
		SELECT id, name, media_type, ext, size_bytes, uploaded_at
		FROM attachments WHERE ref_id = $1 AND content_sha256 = $2
	`, ticketID, digest).Scan(&attachment.ID, &attachment.Name, &attachment.MediaType, &attachment.Ext, &attachment.SizeBytes, &uploadedAt)
	status := http.StatusOK
	if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusCreated
		// A stable object key also makes retries after an interrupted storage
		// write overwrite the same object instead of leaving another copy.
		attachment.ID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("ticket-attachment:"+ticketID.String()+":"+digest)).String()
		size, err := put(ctx, attachment.ID)
		if err != nil {
			return attachment, http.StatusBadGateway, err
		}
		attachment.Name, attachment.MediaType, attachment.Ext, attachment.SizeBytes = name, mediaType, ext, size
		err = tx.QueryRow(ctx, `
			INSERT INTO attachments (id, name, media_type, ext, ref_id, size_bytes, content_sha256)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING uploaded_at
		`, attachment.ID, name, mediaType, ext, ticketID, size, digest).Scan(&uploadedAt)
		if err != nil {
			// A PostgreSQL error leaves this transaction holding the ticket
			// lock. On connection loss the lock may already be released and a
			// retry may be writing the same key, so leave the object intact.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				if cleanupErr := remove(ctx, attachment.ID); cleanupErr != nil {
					log.Printf("cleanup orphaned attachment object failed: %v", cleanupErr)
				}
			}
			return attachment, http.StatusInternalServerError, err
		}
	} else if err != nil {
		return attachment, http.StatusInternalServerError, err
	}
	if err := tx.Commit(ctx); err != nil {
		// Commit may have succeeded even when its response was lost. Preserve
		// the object so the next request can safely return the stored record.
		return attachment, http.StatusInternalServerError, err
	}
	attachment.UploadedAt = timestamptzToRFC3339(uploadedAt)
	attachment.DownloadURL = buildTicketAttachmentDownloadURL(ticketID.String(), attachment.ID)
	return attachment, status, nil
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
		JOIN visible_tickets t ON t.id = a.ref_id
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

	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": fileName}))
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", mediaType)
	if info.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	}

	if _, err := io.Copy(w, object); err != nil {
		log.Printf("stream attachment failed: %v", err)
	}
}

// Delete storage first so a storage failure leaves the database record available
// for retry. MinIO removal is idempotent, including after a database commit fails.
func deleteTicketAttachment(ctx context.Context, tx pgx.Tx, removeObject func(context.Context, string) error, ticketID, userID pgtype.UUID, attachmentID string) (int, error) {
	var allowed bool
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(executor = $2, false) AND COALESCE(status, '') IN ('worksDone', 'closed')
		FROM tickets WHERE id = $1 FOR UPDATE
	`, ticketID, userID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusNotFound, fmt.Errorf("ticket not found")
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !allowed {
		return http.StatusConflict, fmt.Errorf("ticket must be assigned to you and in worksDone or closed status")
	}

	var objectKey string
	err = tx.QueryRow(ctx, `SELECT id FROM attachments WHERE ref_id = $1 AND id = $2 FOR UPDATE`, ticketID, attachmentID).Scan(&objectKey)
	if errors.Is(err, pgx.ErrNoRows) {
		// A repeated request after a successful deletion is also successful.
		return http.StatusNoContent, nil
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if err := removeObject(ctx, objectKey); err != nil {
		return http.StatusBadGateway, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM attachments WHERE ref_id = $1 AND id = $2`, ticketID, attachmentID); err != nil {
		return http.StatusInternalServerError, err
	}
	if err := tx.Commit(ctx); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusNoContent, nil
}

func (s *Server) handleTicketAttachmentDelete(w http.ResponseWriter, r *http.Request, ticketID, userID pgtype.UUID, attachmentID string) {
	if s.db == nil || s.storage == nil {
		http.Error(w, "attachment storage not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to delete attachment", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	status, err := deleteTicketAttachment(ctx, tx, s.storage.RemoveObject, ticketID, userID, attachmentID)
	if err != nil {
		log.Printf("delete ticket attachment failed: %v", err)
		message := "failed to delete attachment"
		if status == http.StatusNotFound || status == http.StatusConflict {
			message = err.Error()
		}
		http.Error(w, message, status)
		return
	}
	s.publishTicketUpdated(ticketID)
	w.WriteHeader(status)
}
