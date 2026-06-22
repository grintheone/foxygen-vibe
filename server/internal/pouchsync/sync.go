package pouchsync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSource = "default"

type Config struct {
	Enabled       bool
	URL           string
	Username      string
	Password      string
	Source        string
	Since         string
	Heartbeat     time.Duration
	RetryMinDelay time.Duration
	RetryMaxDelay time.Duration
}

func (c Config) Normalized() Config {
	if c.Source == "" {
		c.Source = defaultSource
	}
	if c.Since == "" {
		c.Since = "now"
	}
	if c.Heartbeat <= 0 {
		c.Heartbeat = 30 * time.Second
	}
	if c.RetryMinDelay <= 0 {
		c.RetryMinDelay = time.Second
	}
	if c.RetryMaxDelay <= 0 {
		c.RetryMaxDelay = 30 * time.Second
	}
	if c.RetryMaxDelay < c.RetryMinDelay {
		c.RetryMaxDelay = c.RetryMinDelay
	}
	return c
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.URL) == "" {
		return errors.New("POUCHDB_URL is required when pouchdb sync is enabled")
	}
	if _, err := url.ParseRequestURI(c.URL); err != nil {
		return fmt.Errorf("POUCHDB_URL: %w", err)
	}
	switch c.Since {
	case "now", "checkpoint":
	default:
		return fmt.Errorf("POUCHDB_SINCE must be either now or checkpoint, got %q", c.Since)
	}
	return nil
}

type Runner struct {
	config Config
	db     *pgxpool.Pool
	client *http.Client
}

func New(config Config, db *pgxpool.Pool) (*Runner, error) {
	config = config.Normalized()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, errors.New("database pool is required")
	}
	return &Runner{
		config: config,
		db:     db,
		client: &http.Client{},
	}, nil
}

func (r *Runner) Run(ctx context.Context) {
	if !r.config.Enabled {
		return
	}

	log.Printf("pouchdb sync: connecting source=%s url=%s since_mode=%s", r.config.Source, redactURL(r.config.URL), r.config.Since)
	delay := r.config.RetryMinDelay
	useInitialSince := true
	for ctx.Err() == nil {
		since, err := r.resolveSince(ctx, useInitialSince)
		if err != nil {
			log.Printf("pouchdb sync: resolve since: %v", err)
			sleep(ctx, delay)
			delay = nextDelay(delay, r.config.RetryMaxDelay)
			continue
		}
		useInitialSince = false

		if err := r.streamChanges(ctx, since); err != nil && ctx.Err() == nil {
			log.Printf("pouchdb sync: changes feed disconnected source=%s retry_in=%s error=%v", r.config.Source, delay, err)
			sleep(ctx, delay)
			delay = nextDelay(delay, r.config.RetryMaxDelay)
			continue
		}

		delay = r.config.RetryMinDelay
	}
}

func (r *Runner) resolveSince(ctx context.Context, useInitialSince bool) (string, error) {
	if useInitialSince && r.config.Since == "now" {
		return "now", nil
	}
	return r.loadCheckpoint(ctx)
}

func (r *Runner) loadCheckpoint(ctx context.Context) (string, error) {
	var seq string
	err := r.db.QueryRow(ctx, `SELECT last_seq FROM pouchdb_checkpoints WHERE source = $1`, r.config.Source).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return "now", nil
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(seq) == "" {
		return "now", nil
	}
	return seq, nil
}

func (r *Runner) streamChanges(ctx context.Context, since string) error {
	changesURL, err := r.changesURL(since)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, changesURL, nil)
	if err != nil {
		return err
	}
	if r.config.Username != "" || r.config.Password != "" {
		req.SetBasicAuth(r.config.Username, r.config.Password)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("changes feed returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	decoder := json.NewDecoder(resp.Body)
	for ctx.Err() == nil {
		var change changeEvent
		if err := decoder.Decode(&change); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if change.ID == "" {
			continue
		}
		if err := r.applyChange(ctx, change); err != nil {
			return fmt.Errorf("apply change id=%s seq=%s: %w", change.ID, change.seqString(), err)
		}
	}

	return ctx.Err()
}

func (r *Runner) changesURL(since string) (string, error) {
	parsed, err := url.Parse(r.config.URL)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(parsed.Path, "/_changes") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/_changes"
	}

	query := parsed.Query()
	query.Set("feed", "continuous")
	query.Set("include_docs", "true")
	query.Set("since", since)
	query.Set("heartbeat", fmt.Sprintf("%d", r.config.Heartbeat.Milliseconds()))
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (r *Runner) applyChange(ctx context.Context, change changeEvent) error {
	seq := change.seqString()
	if seq == "" {
		return errors.New("change has empty sequence")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := mirrorDocument(ctx, tx, change, seq); err != nil {
		return err
	}
	if err := applyTypedDocument(ctx, tx, change); err != nil {
		return err
	}
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO pouchdb_checkpoints (source, last_seq)
		 VALUES ($1, $2)
		 ON CONFLICT (source) DO UPDATE
		 SET last_seq = EXCLUDED.last_seq,
		     updated_at = NOW() AT TIME ZONE 'UTC'`,
		r.config.Source,
		seq,
	); err != nil {
		return fmt.Errorf("update checkpoint: %w", err)
	}

	return tx.Commit(ctx)
}

type changeEvent struct {
	Seq     json.RawMessage `json:"seq"`
	ID      string          `json:"id"`
	Changes []struct {
		Rev string `json:"rev"`
	} `json:"changes"`
	Deleted bool            `json:"deleted"`
	Doc     json.RawMessage `json:"doc"`
}

func (c changeEvent) seqString() string {
	raw := bytes.TrimSpace(c.Seq)
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return string(raw)
}

func (c changeEvent) rev() string {
	if len(c.Changes) == 0 {
		return ""
	}
	return c.Changes[0].Rev
}

func mirrorDocument(ctx context.Context, tx pgx.Tx, change changeEvent, seq string) error {
	var doc any
	if !change.Deleted && len(change.Doc) > 0 && string(change.Doc) != "null" {
		doc = []byte(change.Doc)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO pouchdb_documents (id, rev, deleted, doc, last_seq)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE
		 SET rev = EXCLUDED.rev,
		     deleted = EXCLUDED.deleted,
		     doc = EXCLUDED.doc,
		     last_seq = EXCLUDED.last_seq,
		     updated_at = NOW() AT TIME ZONE 'UTC'`,
		change.ID,
		change.rev(),
		change.Deleted,
		doc,
		seq,
	); err != nil {
		return fmt.Errorf("mirror document: %w", err)
	}
	return nil
}

func applyTypedDocument(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	switch {
	case strings.HasPrefix(change.ID, "region_"):
		return applyRegion(ctx, tx, change)
	case strings.HasPrefix(change.ID, "manufacturer_"):
		return applyManufacturer(ctx, tx, change)
	case strings.HasPrefix(change.ID, "researchType_"):
		return applyResearchType(ctx, tx, change)
	case strings.HasPrefix(change.ID, "ticketReason_"):
		return applyTicketReason(ctx, tx, change)
	case strings.HasPrefix(change.ID, "client_"):
		return applyClient(ctx, tx, change)
	case strings.HasPrefix(change.ID, "contact_"):
		return applyContact(ctx, tx, change)
	case strings.HasPrefix(change.ID, "classificator_"):
		return applyClassificator(ctx, tx, change)
	case strings.HasPrefix(change.ID, "device_"):
		return applyDevice(ctx, tx, change)
	case strings.HasPrefix(change.ID, "ticket_"):
		return applyTicket(ctx, tx, change)
	default:
		return nil
	}
}

func sleep(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func nextDelay(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

func redactURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return "<invalid-url>"
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(parsed.User.Username(), "xxxxx")
	}
	return parsed.String()
}
