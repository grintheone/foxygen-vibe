package externalusers

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

type legacyExternalUserDoc struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
}

type legacyUserDoc struct {
	ID        string `json:"_id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type legacyExternalUser struct {
	ID           string
	Title        string
	LinkedUserID string
}

type importStats struct {
	Found          int
	Imported       int
	Linked         int
	UnresolvedLink int
}

func main() {
	var (
		sourcePath     string
		dryRun         bool
		commandTimeout time.Duration
	)

	flag.StringVar(&sourcePath, "source", "", "Path to the legacy CouchDB _all_docs JSON export")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse and plan the import without writing to PostgreSQL")
	flag.DurationVar(&commandTimeout, "timeout", 2*time.Minute, "Overall import timeout")
	flag.Parse()

	if strings.TrimSpace(sourcePath) == "" {
		log.Fatal("missing required -source")
	}

	items, err := loadLegacyExternalUsers(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy external users in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items[:min(10, len(items))] {
			log.Printf("dry-run external_user=%s title=%q linked_user=%q", item.ID, item.Title, item.LinkedUserID)
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

	stats, err := importExternalUsers(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d imported=%d linked=%d unresolved_link=%d", stats.Found, stats.Imported, stats.Linked, stats.UnresolvedLink)
}

func loadLegacyExternalUsers(path string) ([]legacyExternalUser, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	externalDocs := make([]legacyExternalUserDoc, 0)
	userIDs := make(map[string]bool)
	userNames := make(map[string][]string)

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

		switch {
		case strings.HasPrefix(meta.ID, "externalUser_"):
			var doc legacyExternalUserDoc
			if err := json.Unmarshal(row.Doc, &doc); err != nil {
				return nil, fmt.Errorf("parse external user document %s: %w", meta.ID, err)
			}
			externalDocs = append(externalDocs, doc)
		case strings.HasPrefix(meta.ID, "user_"):
			var doc legacyUserDoc
			if err := json.Unmarshal(row.Doc, &doc); err != nil {
				return nil, fmt.Errorf("parse user document %s: %w", meta.ID, err)
			}

			legacyID := trimLegacyPrefix(doc.ID)
			if legacyID == "" {
				continue
			}
			userIDs[legacyID] = true

			name := normalizeName(strings.TrimSpace(strings.Join([]string{doc.FirstName, doc.LastName}, " ")))
			if name != "" {
				userNames[name] = append(userNames[name], legacyID)
			}
		}
	}

	items := make([]legacyExternalUser, 0, len(externalDocs))
	for _, doc := range externalDocs {
		id := trimLegacyPrefix(doc.ID)
		if id == "" {
			continue
		}

		title := strings.Join(strings.Fields(strings.TrimSpace(doc.Title)), " ")
		var linkedUserID string
		switch {
		case userIDs[id]:
			linkedUserID = id
		default:
			name := normalizeName(title)
			if matches := userNames[name]; len(matches) == 1 {
				linkedUserID = matches[0]
			}
		}

		items = append(items, legacyExternalUser{
			ID:           id,
			Title:        title,
			LinkedUserID: linkedUserID,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importExternalUsers(ctx context.Context, db *pgxpool.Pool, items []legacyExternalUser) (importStats, error) {
	stats := importStats{Found: len(items)}

	if err := ensureExternalUsersSchema(ctx, db); err != nil {
		return stats, err
	}

	accountIDs, err := loadIDSet(ctx, db, `SELECT user_id FROM accounts`)
	if err != nil {
		return stats, fmt.Errorf("load accounts: %w", err)
	}

	for _, item := range items {
		var linkedUserID any
		if accountIDs[item.LinkedUserID] {
			linkedUserID = item.LinkedUserID
			stats.Linked++
		} else if strings.TrimSpace(item.LinkedUserID) != "" {
			stats.UnresolvedLink++
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO external_users (id, title, linked_user_id)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (id) DO UPDATE
			 SET title = EXCLUDED.title,
			     linked_user_id = EXCLUDED.linked_user_id`,
			item.ID,
			item.Title,
			linkedUserID,
		); err != nil {
			return stats, fmt.Errorf("import external user %s: %w", item.ID, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureExternalUsersSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS external_users (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			linked_user_id UUID REFERENCES accounts(user_id) ON DELETE SET NULL
		)`,
	); err != nil {
		return fmt.Errorf("ensure external_users table: %w", err)
	}

	statements := []struct {
		query string
		label string
	}{
		{`ALTER TABLE external_users ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`, "title"},
		{`ALTER TABLE external_users ADD COLUMN IF NOT EXISTS linked_user_id UUID REFERENCES accounts(user_id) ON DELETE SET NULL`, "linked_user_id"},
	}

	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt.query); err != nil {
			return fmt.Errorf("ensure external_users.%s column: %w", stmt.label, err)
		}
	}

	return nil
}

func loadIDSet(ctx context.Context, db *pgxpool.Pool, query string) (map[string]bool, error) {
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func normalizeName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
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
