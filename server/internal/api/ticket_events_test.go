package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		moleculer:     client,
		publicBaseURL: "https://foxygen.example.com",
		auth:          authConfig{jwtSecret: []byte("test-signing-key")},
		ticketAttachmentsLoader: func(_ context.Context, id pgtype.UUID) ([]ticketAttachmentResponse, error) {
			if id != ticketID {
				t.Fatalf("unexpected attachment ticket ID %s", id.String())
			}
			return []ticketAttachmentResponse{{
				ID: "attachment-1", Name: "photo.jpg", MediaType: "image/jpeg",
				DownloadURL: buildTicketAttachmentDownloadURL(id.String(), "attachment-1"),
			}}, nil
		},
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
	expectedURL, err := srv.ticketAttachmentEventURL(ticketID.String(), "attachment-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Ticket.Attachments) != 1 || received.Ticket.Attachments[0].Path != expectedURL || received.Ticket.Attachments[0].ID != "attachment-1" {
		t.Fatalf("unexpected attachments: %#v", received.Ticket.Attachments)
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

func TestBroadcastTicketEventAttachmentFailuresAndEmptyArray(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		attachments []ticketAttachmentResponse
		loadErr     error
		wantErr     string
	}{
		{name: "empty attachments"},
		{name: "attachment load failure", loadErr: errors.New("database failure"), wantErr: "load ticket event attachments"},
		{name: "missing public URL", attachments: []ticketAttachmentResponse{{ID: "attachment-1"}}, wantErr: "PUBLIC_BASE_URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v1/broadcast/"+ticketCreatedEvent {
					t.Errorf("unexpected event path %s", r.URL.Path)
				}
				var body struct {
					Params struct {
						Ticket map[string]json.RawMessage `json:"ticket"`
					} `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if string(body.Params.Ticket["attachments"]) != "[]" {
					t.Errorf("expected empty attachment array: %s", body.Params.Ticket["attachments"])
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer sidecar.Close()
			client, err := moleculer.New(moleculer.Config{Enabled: true, URL: sidecar.URL, Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			srv := &Server{
				moleculer: client,
				editorTicketDetailLoader: func(context.Context, pgtype.UUID) (editorTicketDetailResponse, bool, error) {
					return editorTicketDetailResponse{}, true, nil
				},
				ticketAttachmentsLoader: func(context.Context, pgtype.UUID) ([]ticketAttachmentResponse, error) {
					return tc.attachments, tc.loadErr
				},
			}
			err = srv.broadcastTicketEvent(context.Background(), ticketCreatedEvent, "created", pgtype.UUID{})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("unexpected error: %v", err)
				}
				if calls != 0 {
					t.Fatal("broadcast an incomplete event")
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("calls=%d, error=%v", calls, err)
			}
		})
	}
}
