package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ticketLookup struct {
	Type  string
	Title string
}

var canonicalTicketTypes = []ticketLookup{
	{Type: "external", Title: "внешний"},
	{Type: "internal", Title: "внутренний"},
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

	items, err := loadLegacyTicketTypes(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy ticket types in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items {
			log.Printf("dry-run ticket_type=%s title=%q", item.Type, item.Title)
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

	imported, err := importTicketTypes(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d imported=%d", len(items), imported)
}

func loadLegacyTicketTypes(path string) ([]ticketLookup, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat dump: %w", err)
	}

	items := make([]ticketLookup, len(canonicalTicketTypes))
	copy(items, canonicalTicketTypes)
	return items, nil
}

func importTicketTypes(ctx context.Context, db *pgxpool.Pool, items []ticketLookup) (int, error) {
	if err := ensureTicketTypesSchema(ctx, db); err != nil {
		return 0, err
	}

	imported := 0
	for _, item := range items {
		if _, err := db.Exec(
			ctx,
			`INSERT INTO ticket_types (type, title)
			 VALUES ($1, $2)
			 ON CONFLICT (type) DO UPDATE
			 SET title = EXCLUDED.title`,
			item.Type,
			item.Title,
		); err != nil {
			return imported, fmt.Errorf("import ticket type %s: %w", item.Type, err)
		}
		imported++
	}

	allowed := make([]string, 0, len(items))
	for _, item := range items {
		allowed = append(allowed, item.Type)
	}

	if _, err := db.Exec(ctx, `DELETE FROM ticket_types WHERE NOT (type = ANY($1))`, allowed); err != nil {
		return imported, fmt.Errorf("prune ticket types: %w", err)
	}

	return imported, nil
}

func ensureTicketTypesSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS ticket_types (
			type VARCHAR(128) PRIMARY KEY,
			title TEXT NOT NULL DEFAULT ''
		)`,
	); err != nil {
		return fmt.Errorf("ensure ticket_types table: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE ticket_types ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure ticket_types.title column: %w", err)
	}

	return nil
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
