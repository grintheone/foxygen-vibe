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

type legacyManufacturerDoc struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
}

type legacyManufacturer struct {
	ID    string
	Title string
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

	items, err := loadLegacyManufacturers(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy manufacturers in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items {
			log.Printf("dry-run manufacturer=%s title=%q", item.ID, item.Title)
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

	imported, err := importManufacturers(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d imported=%d", len(items), imported)
}

func loadLegacyManufacturers(path string) ([]legacyManufacturer, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyManufacturer, 0)
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
		if !strings.HasPrefix(meta.ID, "manufacturer_") {
			continue
		}

		var doc legacyManufacturerDoc
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse manufacturer document %s: %w", meta.ID, err)
		}

		id := trimLegacyPrefix(doc.ID)
		title := normalizeWhitespace(doc.Title)
		if id == "" || title == "" {
			continue
		}

		items = append(items, legacyManufacturer{ID: id, Title: title})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importManufacturers(ctx context.Context, db *pgxpool.Pool, items []legacyManufacturer) (int, error) {
	if err := ensureManufacturersSchema(ctx, db); err != nil {
		return 0, err
	}

	imported := 0
	for _, item := range items {
		if _, err := db.Exec(
			ctx,
			`INSERT INTO manufacturers (id, title)
			 VALUES ($1, $2)
			 ON CONFLICT (id) DO UPDATE
			 SET title = EXCLUDED.title`,
			item.ID,
			item.Title,
		); err != nil {
			return imported, fmt.Errorf("import manufacturer %s (%s): %w", item.ID, item.Title, err)
		}
		imported++
	}

	return imported, nil
}

func ensureManufacturersSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS manufacturers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL DEFAULT ''
		)`,
	); err != nil {
		return fmt.Errorf("ensure manufacturers table: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE manufacturers
		 ALTER COLUMN id SET DEFAULT gen_random_uuid()`,
	); err != nil {
		return fmt.Errorf("ensure manufacturers.id default: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE manufacturers
		 ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure manufacturers.title column: %w", err)
	}

	return nil
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func trimLegacyPrefix(value string) string {
	parts := strings.Split(value, "_")
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(parts[len(parts)-1])
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
