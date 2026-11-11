package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

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

	ticket, found, err := s.loadTicketDetail(ctx, ticketID, false)
	if err != nil {
		return fmt.Errorf("load ticket event snapshot: %w", err)
	}
	if !found {
		return errors.New("load ticket event snapshot: ticket not found")
	}

	attachments, err := s.loadTicketAttachments(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("load ticket event attachments: %w", err)
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
