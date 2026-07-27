package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"foxygen-vibe/server/internal/moleculer"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestBroadcastTicketEventSendsCurrentAppTicketSnapshot(t *testing.T) {
	t.Parallel()

	ticketID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	var received ticketEventPayload

	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/broadcast/"+ticketUpdatedEvent {
			t.Fatalf("unexpected event path %q", r.URL.Path)
		}

		var request struct {
			Params ticketEventPayload `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode event request: %v", err)
		}
		received = request.Params
		w.WriteHeader(http.StatusOK)
	}))
	defer sidecar.Close()

	client, err := moleculer.New(moleculer.Config{
		Enabled: true,
		URL:     sidecar.URL,
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("create Moleculer client: %v", err)
	}

	srv := &Server{
		moleculer: client,
		editorTicketDetailLoader: func(_ context.Context, id pgtype.UUID) (editorTicketDetailResponse, bool, error) {
			if id != ticketID {
				t.Fatalf("expected ticket %s, got %s", ticketID.String(), id.String())
			}
			return editorTicketDetailResponse{
				ID:          ticketID.String(),
				Number:      42,
				Status:      "inWork",
				Description: "Repair analyzer",
			}, true, nil
		},
	}

	if err := srv.broadcastTicketEvent(
		context.Background(),
		ticketUpdatedEvent,
		"updated",
		ticketID,
	); err != nil {
		t.Fatalf("broadcast ticket event: %v", err)
	}

	if received.Operation != "updated" || received.Source != "app" {
		t.Fatalf("unexpected event envelope: %#v", received)
	}
	if received.OccurredAt.IsZero() {
		t.Fatal("expected occurredAt to be populated")
	}
	if received.Ticket.ID != ticketID.String() ||
		received.Ticket.Number != 42 ||
		received.Ticket.Status != "inWork" {
		t.Fatalf("unexpected ticket snapshot: %#v", received.Ticket)
	}
}

func TestBroadcastTicketEventIsDisabledWithoutSidecar(t *testing.T) {
	t.Parallel()

	loaderCalled := false
	srv := &Server{
		editorTicketDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorTicketDetailResponse, bool, error) {
			loaderCalled = true
			return editorTicketDetailResponse{}, true, nil
		},
	}

	if err := srv.broadcastTicketEvent(
		context.Background(),
		ticketCreatedEvent,
		"created",
		pgtype.UUID{},
	); err != nil {
		t.Fatalf("disabled broadcast returned an error: %v", err)
	}
	if loaderCalled {
		t.Fatal("ticket snapshot should not load when the sidecar is disabled")
	}
}
