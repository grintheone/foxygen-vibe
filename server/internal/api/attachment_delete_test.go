package api

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type attachmentDeleteRow func(...any) error

func (r attachmentDeleteRow) Scan(dest ...any) error { return r(dest...) }

type attachmentDeleteTx struct {
	pgx.Tx
	allowed                                        bool
	ticketErr, attachmentErr, deleteErr, commitErr error
	queries                                        int
	events                                         []string
}

func (tx *attachmentDeleteTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	tx.queries++
	return attachmentDeleteRow(func(dest ...any) error {
		if tx.queries == 1 {
			*dest[0].(*bool) = tx.allowed
			return tx.ticketErr
		}
		*dest[0].(*string) = "attachment-key"
		return tx.attachmentErr
	})
}
func (tx *attachmentDeleteTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	tx.events = append(tx.events, "delete record")
	return pgconn.NewCommandTag("DELETE 1"), tx.deleteErr
}
func (tx *attachmentDeleteTx) Commit(context.Context) error {
	tx.events = append(tx.events, "commit")
	return tx.commitErr
}

func TestDeleteTicketAttachment(t *testing.T) {
	failure := errors.New("unavailable")
	for _, tc := range []struct {
		name                                                       string
		denied                                                     bool
		ticketErr, attachmentErr, storageErr, deleteErr, commitErr error
		status                                                     int
		events                                                     []string
	}{
		{name: "success", status: http.StatusNoContent, events: []string{"remove object", "delete record", "commit"}},
		{name: "wrong executor or status", denied: true, status: http.StatusConflict},
		{name: "missing ticket", ticketErr: pgx.ErrNoRows, status: http.StatusNotFound},
		{name: "already removed or belongs to another ticket", attachmentErr: pgx.ErrNoRows, status: http.StatusNoContent},
		{name: "storage failure preserves record", storageErr: failure, status: http.StatusBadGateway, events: []string{"remove object"}},
		{name: "database failure can be retried", deleteErr: failure, status: http.StatusInternalServerError, events: []string{"remove object", "delete record"}},
		{name: "commit failure is not success", commitErr: failure, status: http.StatusInternalServerError, events: []string{"remove object", "delete record", "commit"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &attachmentDeleteTx{allowed: !tc.denied, ticketErr: tc.ticketErr, attachmentErr: tc.attachmentErr, deleteErr: tc.deleteErr, commitErr: tc.commitErr}
			remove := func(_ context.Context, key string) error {
				if key != "attachment-key" {
					t.Fatalf("unexpected object key: %s", key)
				}
				tx.events = append(tx.events, "remove object")
				return tc.storageErr
			}
			status, err := deleteTicketAttachment(context.Background(), tx, remove, pgtype.UUID{}, pgtype.UUID{}, "attachment-key")
			if status != tc.status {
				t.Fatalf("status = %d, want %d", status, tc.status)
			}
			if (err == nil) != (tc.status == http.StatusNoContent) {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tx.events, tc.events) {
				t.Fatalf("events = %v, want %v", tx.events, tc.events)
			}
		})
	}
}
