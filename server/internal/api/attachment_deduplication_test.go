package api

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Transaction doubles retain committed state across requests, like separate API
// instances using the same database. Locks are held until commit/rollback.
type attachmentDeliveryStore struct {
	sync.Mutex
	ids       []string
	delivered map[string]bool
}
type attachmentDeliveryTx struct {
	pgx.Tx
	store     *attachmentDeliveryStore
	locked    bool
	pending   []string
	commitErr error
}

func (tx *attachmentDeliveryTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "pg_advisory_xact_lock") {
		tx.store.Lock()
		tx.locked = true
	} else if strings.Contains(sql, "INSERT INTO ticket_attachment_deliveries") {
		tx.pending = args[1].([]string)
	} else {
		panic("unexpected SQL: " + sql)
	}
	return pgconn.CommandTag{}, nil
}
func (tx *attachmentDeliveryTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return attachmentDeleteRow(func(dest ...any) error {
		ids := []string{}
		for _, id := range tx.store.ids {
			if !tx.store.delivered[id] {
				ids = append(ids, id)
			}
		}
		*dest[0].(*[]string) = ids
		return nil
	})
}
func (tx *attachmentDeliveryTx) Commit(context.Context) error {
	if tx.commitErr != nil {
		return tx.commitErr
	}
	for _, id := range tx.pending {
		tx.store.delivered[id] = true
	}
	return tx.Rollback(context.Background())
}
func (tx *attachmentDeliveryTx) Rollback(context.Context) error {
	if tx.locked {
		tx.locked = false
		tx.store.Unlock()
	}
	return nil
}
func TestAttachmentDeliveryAcrossUploadsCloseAndConcurrentRetries(t *testing.T) {
	store := &attachmentDeliveryStore{ids: []string{"photo-1"}, delivered: map[string]bool{}}
	var sent []string
	send := func(items []ticketAttachmentResponse) error {
		for _, item := range items {
			sent = append(sent, item.ID)
		}
		return nil
	}
	deliver := func() {
		tx := &attachmentDeliveryTx{store: store}
		defer tx.Rollback(context.Background())
		if err := deliverTicketAttachments(context.Background(), tx, pgtype.UUID{}, send); err != nil {
			t.Error(err)
		}
	}
	deliver() // First photo uploaded.
	store.ids = append(store.ids, "photo-2")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); deliver() }() // upload, close, editor update, retries
	}
	wg.Wait()
	if !reflect.DeepEqual(sent, []string{"photo-1", "photo-2"}) {
		t.Fatalf("sent attachments: %v", sent)
	}
}
func TestAttachmentDeliveryFailureRemainsRetryable(t *testing.T) {
	store := &attachmentDeliveryStore{ids: []string{"photo-1"}, delivered: map[string]bool{}}
	failure := errors.New("sidecar unavailable")
	tx := &attachmentDeliveryTx{store: store}
	err := deliverTicketAttachments(context.Background(), tx, pgtype.UUID{}, func([]ticketAttachmentResponse) error { return failure })
	tx.Rollback(context.Background())
	if !errors.Is(err, failure) || store.delivered["photo-1"] {
		t.Fatalf("failed delivery acknowledged: %v", err)
	}
	retry := &attachmentDeliveryTx{store: store}
	defer retry.Rollback(context.Background())
	if err := deliverTicketAttachments(context.Background(), retry, pgtype.UUID{}, func(items []ticketAttachmentResponse) error {
		if len(items) != 1 || items[0].ID != "photo-1" {
			t.Fatalf("retry items: %v", items)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !store.delivered["photo-1"] {
		t.Fatal("successful retry not recorded")
	}
}

type attachmentUploadTx struct {
	pgx.Tx
	existing                        bool
	allowed                         bool
	lookupErr, insertErr, commitErr error
	inserted                        bool
	committed                       bool
}

func (tx *attachmentUploadTx) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return attachmentDeleteRow(func(dest ...any) error {
		switch {
		case strings.Contains(sql, "FOR UPDATE"):
			*dest[0].(*bool) = tx.allowed
		case strings.Contains(sql, "content_sha256 ="):
			if tx.lookupErr != nil {
				return tx.lookupErr
			}
			if !tx.existing {
				return pgx.ErrNoRows
			}
			*dest[0].(*string) = "original-id"
			*dest[1].(*string) = "original.jpg"
			*dest[2].(*string) = "image/jpeg"
			*dest[3].(*string) = "jpg"
			*dest[4].(*int64) = 42
		case strings.Contains(sql, "INSERT INTO attachments"):
			tx.inserted = true
			return tx.insertErr
		default:
			panic("unexpected SQL: " + sql)
		}
		return nil
	})
}
func (tx *attachmentUploadTx) Commit(context.Context) error {
	tx.committed = tx.commitErr == nil
	return tx.commitErr
}
func TestUploadAttachmentDeduplicatesBeforeWritingStorage(t *testing.T) {
	failure := errors.New("unavailable")
	for _, tc := range []struct {
		name                                    string
		existing, denied                        bool
		lookupErr, insertErr, putErr, commitErr error
		wantStatus, wantPuts, wantRemoves       int
	}{
		{name: "new file", wantStatus: http.StatusCreated, wantPuts: 1},
		{name: "retry or same bytes with another filename", existing: true, wantStatus: http.StatusOK},
		{name: "permission checked before deduplication", existing: true, denied: true, wantStatus: http.StatusConflict},
		{name: "lookup failure never writes storage", lookupErr: failure, wantStatus: http.StatusInternalServerError},
		{name: "storage failure", putErr: failure, wantStatus: http.StatusBadGateway, wantPuts: 1},
		{name: "insert failure cleans orphan", insertErr: &pgconn.PgError{Code: "23514"}, wantStatus: http.StatusInternalServerError, wantPuts: 1, wantRemoves: 1},
		{name: "connection loss preserves stable object", insertErr: failure, wantStatus: http.StatusInternalServerError, wantPuts: 1},
		{name: "commit failure", commitErr: failure, wantStatus: http.StatusInternalServerError, wantPuts: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &attachmentUploadTx{existing: tc.existing, allowed: !tc.denied, lookupErr: tc.lookupErr, insertErr: tc.insertErr, commitErr: tc.commitErr}
			puts, removes := 0, 0
			attachment, status, err := uploadTicketAttachment(context.Background(), tx, pgtype.UUID{}, pgtype.UUID{}, "new.jpg", "image/jpeg", "jpg", "digest", func(context.Context, string) (int64, error) {
				puts++
				return 42, tc.putErr
			}, func(context.Context, string) error { removes++; return nil })
			if status != tc.wantStatus || puts != tc.wantPuts || removes != tc.wantRemoves {
				t.Fatalf("status=%d puts=%d err=%v", status, puts, err)
			}
			if (err == nil) != (status < 400) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.existing && status == http.StatusOK && (attachment.ID != "original-id" || attachment.Name != "original.jpg" || tx.inserted) {
				t.Fatalf("retry changed attachment: %+v", attachment)
			}
			if status < 400 && !tx.committed {
				t.Fatal("success without commit")
			}
		})
	}
}

func TestInterruptedUploadUsesSameObjectKeyOnRetry(t *testing.T) {
	ticketID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	otherTicket := mustUUID(t, "22222222-2222-2222-2222-222222222222")
	keys := []string{}
	for _, input := range []struct {
		ticket           pgtype.UUID
		digest, filename string
	}{
		{ticketID, "image-digest", "first.jpg"},
		{ticketID, "image-digest", "renamed.jpg"},
		{otherTicket, "image-digest", "first.jpg"},
		{ticketID, "different-image", "first.jpg"},
	} {
		tx := &attachmentUploadTx{allowed: true}
		_, _, err := uploadTicketAttachment(context.Background(), tx, input.ticket, pgtype.UUID{}, input.filename, "image/jpeg", "jpg", input.digest, func(_ context.Context, key string) (int64, error) {
			keys = append(keys, key)
			return 0, errors.New("storage response lost")
		}, func(context.Context, string) error { t.Fatal("ambiguous upload must not remove storage"); return nil })
		if err == nil {
			t.Fatal("expected upload error")
		}
	}
	if keys[0] != keys[1] || keys[0] == keys[2] || keys[0] == keys[3] {
		t.Fatalf("unexpected object keys: %v", keys)
	}
}

func TestAttachmentDeliveryCommitFailureIsReported(t *testing.T) {
	store := &attachmentDeliveryStore{ids: []string{"photo-1"}, delivered: map[string]bool{}}
	failure := errors.New("commit response lost")
	tx := &attachmentDeliveryTx{store: store, commitErr: failure}
	defer tx.Rollback(context.Background())
	err := deliverTicketAttachments(context.Background(), tx, pgtype.UUID{}, func([]ticketAttachmentResponse) error { return nil })
	if !errors.Is(err, failure) {
		t.Fatalf("unexpected error: %v", err)
	}
}
