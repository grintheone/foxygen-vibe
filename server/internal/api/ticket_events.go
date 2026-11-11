package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	ticketCreatedEvent = "foxygen.ticket.created"
	ticketUpdatedEvent = "foxygen.ticket.updated"

	ticketEventPublishTimeout = 5 * time.Second
)

type ticketEventPayload struct {
	OccurredAt time.Time           `json:"occurredAt"`
	Operation  string              `json:"operation"`
	Source     string              `json:"source"`
	Ticket     ticketEventSnapshot `json:"ticket"`
}

type ticketEventSnapshot struct {
	editorTicketDetailResponse
	Attachments []ticketEventAttachment `json:"attachments"`
}

type ticketEventAttachment struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func (s *Server) publishTicketCreated(ticketID pgtype.UUID) {
	s.publishTicketEvent(ticketCreatedEvent, "created", ticketID)
}

func (s *Server) publishTicketUpdated(ticketID pgtype.UUID) {
	s.publishTicketEvent(ticketUpdatedEvent, "updated", ticketID)
}

func (s *Server) publishTicketEvent(eventName string, operation string, ticketID pgtype.UUID) {
	if s.moleculer == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), ticketEventPublishTimeout)
	defer cancel()

	if err := s.broadcastTicketEvent(ctx, eventName, operation, ticketID); err != nil {
		log.Printf("broadcast %s for ticket %s failed: %v", eventName, ticketID.String(), err)
	}
}

func (s *Server) broadcastTicketEvent(
	ctx context.Context,
	eventName string,
	operation string,
	ticketID pgtype.UUID,
) error {
	if s.moleculer == nil {
		return nil
	}

	if s.db == nil {
		// Loader-backed servers are used by snapshot unit tests.
		attachments, err := s.loadTicketAttachments(ctx, ticketID)
		if err != nil {
			return fmt.Errorf("load ticket event attachments: %w", err)
		}
		return s.sendTicketEvent(ctx, eventName, operation, ticketID, attachments, s.db)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	return deliverTicketAttachments(ctx, tx, ticketID, func(attachments []ticketAttachmentResponse) error {
		return s.sendTicketEvent(ctx, eventName, operation, ticketID, attachments, tx)
	})
}

// Serialize delivery per ticket across API instances. Only acknowledge files
// after a successful sidecar response; failed sends remain eligible for retry.
func deliverTicketAttachments(ctx context.Context, tx pgx.Tx, ticketID pgtype.UUID, send func([]ticketAttachmentResponse) error) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "ticket-attachments:"+ticketID.String()); err != nil {
		return err
	}
	var ids []string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(array_agg(a.id ORDER BY a.uploaded_at, a.id), ARRAY[]::text[])
		FROM attachments a
		WHERE a.ref_id = $1 AND NOT EXISTS (
			SELECT 1 FROM ticket_attachment_deliveries d
			WHERE d.ticket_id = a.ref_id AND d.attachment_id = a.id
		)
	`, ticketID).Scan(&ids); err != nil {
		return fmt.Errorf("load ticket event attachments: %w", err)
	}
	attachments := make([]ticketAttachmentResponse, 0, len(ids))
	for _, id := range ids {
		attachments = append(attachments, ticketAttachmentResponse{ID: id})
	}
	if err := send(attachments); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ticket_attachment_deliveries (ticket_id, attachment_id)
			SELECT $1, unnest($2::text[]) ON CONFLICT DO NOTHING
		`, ticketID, ids); err != nil {
			return fmt.Errorf("record ticket attachment delivery: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Server) sendTicketEvent(ctx context.Context, eventName, operation string, ticketID pgtype.UUID, attachments []ticketAttachmentResponse, query ticketDetailQuerier) error {
	ticket, found, err := s.loadTicketDetailUsing(ctx, query, ticketID, false)
	if err != nil {
		return fmt.Errorf("load ticket event snapshot: %w", err)
	}
	if !found {
		return errors.New("load ticket event snapshot: ticket not found")
	}
	eventAttachments := make([]ticketEventAttachment, 0, len(attachments))
	for _, attachment := range attachments {
		downloadURL, err := s.ticketAttachmentEventURL(ticketID.String(), attachment.ID)
		if err != nil {
			return fmt.Errorf("create ticket event attachment URL: %w", err)
		}
		eventAttachments = append(eventAttachments, ticketEventAttachment{
			ID:   attachment.ID,
			Path: downloadURL,
		})
	}

	return s.moleculer.BroadcastEvent(ctx, eventName, ticketEventPayload{
		OccurredAt: time.Now().UTC(),
		Operation:  operation,
		Source:     "app",
		Ticket: ticketEventSnapshot{
			editorTicketDetailResponse: ticket,
			Attachments:                eventAttachments,
		},
	})
}
