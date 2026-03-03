package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type legacyDump struct {
	Rows []legacyRow `json:"rows"`
}

type legacyRow struct {
	Doc json.RawMessage `json:"doc"`
}

type legacyAttachmentDoc struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"mediaType"`
	Ext       string `json:"ext"`
}

type legacyAttachment struct {
	ID        string
	Name      string
	MediaType string
	Ext       string
	RefID     string
}

type attachmentImportStats struct {
	Found         int
	Imported      int
	MissingTicket int
}

func main() {
	var (
		sourcePath     string
		dryRun         bool
		commandTimeout time.Duration
	)

	flag.StringVar(&sourcePath, "source", "", "Path to the legacy CouchDB _all_docs JSON export")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse and plan the import without writing to PostgreSQL")
	flag.DurationVar(&commandTimeout, "timeout", 5*time.Minute, "Overall import timeout")
	flag.Parse()

	if strings.TrimSpace(sourcePath) == "" {
		log.Fatal("missing required -source")
	}

	items, err := loadLegacyAttachments(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy attachments in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items[:min(10, len(items))] {
			log.Printf("dry-run attachment=%s ref_id=%s name=%q", item.ID, item.RefID, item.Name)
		}
		log.Printf("dry-run complete")
		return
	}

	databaseURL, err := resolveDatabaseURL(".env")
	if err != nil {
		log.Fatal(err)
	}
	if databaseURL == "" {
		log.Fatal("database is not configured; set DATABASE_URL or DB_* in server/.env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	stats, err := importAttachments(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d imported=%d missing_ticket=%d", stats.Found, stats.Imported, stats.MissingTicket)
}

func loadLegacyAttachments(path string) ([]legacyAttachment, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyAttachment, 0)
	for _, row := range dump.Rows {
		if len(row.Doc) == 0 || string(row.Doc) == "null" {
			continue
		}

		var meta struct {
			ID string `json:"_id"`
		}
		if err := json.Unmarshal(row.Doc, &meta); err != nil {
			return nil, fmt.Errorf("parse document metadata: %w", err)
		}
		if !strings.HasPrefix(meta.ID, "ticket_") {
			continue
		}

		ticketID := trimLegacyPrefix(meta.ID)
		if ticketID == "" {
			continue
		}

		var doc struct {
			Attachments []legacyAttachmentDoc `json:"attachments"`
		}
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse ticket attachments %s: %w", meta.ID, err)
		}

		for _, attachment := range doc.Attachments {
			id := strings.TrimSpace(attachment.ID)
			name := strings.TrimSpace(attachment.Name)
			mediaType := strings.TrimSpace(attachment.MediaType)
			ext := strings.TrimSpace(attachment.Ext)
			if id == "" || name == "" || mediaType == "" || ext == "" {
				continue
			}

			items = append(items, legacyAttachment{
				ID:        id,
				Name:      name,
				MediaType: mediaType,
				Ext:       ext,
				RefID:     ticketID,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RefID != items[j].RefID {
			return items[i].RefID < items[j].RefID
		}
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importAttachments(ctx context.Context, db *pgxpool.Pool, items []legacyAttachment) (attachmentImportStats, error) {
	stats := attachmentImportStats{Found: len(items)}

	if err := ensureAttachmentsSchema(ctx, db); err != nil {
		return stats, err
	}

	ticketIDs, err := loadTicketIDs(ctx, db)
	if err != nil {
		return stats, err
	}

	for _, item := range items {
		if !ticketIDs[item.RefID] {
			stats.MissingTicket++
			log.Printf("attachment=%s has missing ticket=%s; skipping", item.ID, item.RefID)
			continue
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO attachments (id, name, media_type, ext, ref_id)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (id) DO UPDATE
			 SET name = EXCLUDED.name,
			     media_type = EXCLUDED.media_type,
			     ext = EXCLUDED.ext,
			     ref_id = EXCLUDED.ref_id`,
			item.ID,
			item.Name,
			item.MediaType,
			item.Ext,
			item.RefID,
		); err != nil {
			return stats, fmt.Errorf("import attachment %s: %w", item.ID, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureAttachmentsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS attachments (
			id TEXT NOT NULL PRIMARY KEY,
			name TEXT NOT NULL,
			media_type TEXT NOT NULL,
			ext TEXT NOT NULL,
			ref_id UUID NOT NULL
		)`,
	); err != nil {
		return fmt.Errorf("ensure attachments table: %w", err)
	}

	statements := []struct {
		query string
		label string
	}{
		{`ALTER TABLE attachments ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''`, "name"},
		{`ALTER TABLE attachments ADD COLUMN IF NOT EXISTS media_type TEXT NOT NULL DEFAULT ''`, "media_type"},
		{`ALTER TABLE attachments ADD COLUMN IF NOT EXISTS ext TEXT NOT NULL DEFAULT ''`, "ext"},
		{`ALTER TABLE attachments ADD COLUMN IF NOT EXISTS ref_id UUID`, "ref_id"},
	}

	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt.query); err != nil {
			return fmt.Errorf("ensure attachments.%s column: %w", stmt.label, err)
		}
	}

	return nil
}

func loadTicketIDs(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT id FROM tickets`)
	if err != nil {
		return nil, fmt.Errorf("load tickets: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan ticket id: %w", err)
		}
		ids[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tickets: %w", err)
	}

	return ids, nil
}

func trimLegacyPrefix(value string) string {
	parts := strings.Split(value, "_")
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func resolveDatabaseURL(dotEnvPath string) (string, error) {
	fileEnv, err := loadDotEnv(dotEnvPath)
	if err != nil {
		return "", err
	}

	if databaseURL := getConfigValue(fileEnv, "DATABASE_URL"); databaseURL != "" {
		return databaseURL, nil
	}

	host := getConfigValue(fileEnv, "DB_HOST")
	port := getConfigValue(fileEnv, "DB_PORT")
	user := getConfigValue(fileEnv, "DB_USER")
	password := getConfigValue(fileEnv, "DB_PASSWORD")
	name := getConfigValue(fileEnv, "DB_NAME")
	sslmode := getConfigValue(fileEnv, "DB_SSLMODE")

	if host == "" || port == "" || user == "" || name == "" {
		return "", nil
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	query := url.Values{}
	query.Set("sslmode", sslmode)

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     host + ":" + port,
		Path:     name,
		RawQuery: query.Encode(),
	}).String(), nil
}

func loadDotEnv(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	values := make(map[string]string)
	for index, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: invalid line", path, index+1)
		}

		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return values, nil
}

func getConfigValue(fileEnv map[string]string, key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}

	return strings.TrimSpace(fileEnv[key])
}
