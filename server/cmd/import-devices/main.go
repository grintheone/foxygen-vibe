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

type legacyDeviceDoc struct {
	ID             string          `json:"_id"`
	Classificator  string          `json:"classificator"`
	SerialNumber   string          `json:"serialNumber"`
	Properties     json.RawMessage `json:"properties"`
	ConnectedToLis bool            `json:"connectedToLis"`
	IsUsed         *bool           `json:"isUsed"`
}

type legacyDevice struct {
	ID             string
	Classificator  string
	SerialNumber   string
	Properties     json.RawMessage
	ConnectedToLis bool
	IsUsed         bool
}

type deviceImportStats struct {
	Found                int
	Imported             int
	MissingClassificator int
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

	items, err := loadLegacyDevices(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy devices in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items {
			log.Printf("dry-run device=%s classificator=%q serial=%q", item.ID, item.Classificator, item.SerialNumber)
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

	stats, err := importDevices(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"import complete: found=%d imported=%d missing_classificator=%d",
		stats.Found,
		stats.Imported,
		stats.MissingClassificator,
	)
}

func loadLegacyDevices(path string) ([]legacyDevice, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyDevice, 0)
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
		if !strings.HasPrefix(meta.ID, "device_") {
			continue
		}

		var doc legacyDeviceDoc
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse device document %s: %w", meta.ID, err)
		}

		id := trimLegacyPrefix(doc.ID)
		if id == "" {
			continue
		}

		items = append(items, legacyDevice{
			ID:             id,
			Classificator:  strings.TrimSpace(doc.Classificator),
			SerialNumber:   strings.TrimSpace(doc.SerialNumber),
			Properties:     normalizeJSON(doc.Properties, []byte(`{}`)),
			ConnectedToLis: doc.ConnectedToLis,
			IsUsed:         doc.IsUsed != nil && *doc.IsUsed,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importDevices(ctx context.Context, db *pgxpool.Pool, items []legacyDevice) (deviceImportStats, error) {
	stats := deviceImportStats{Found: len(items)}

	if err := ensureDevicesSchema(ctx, db); err != nil {
		return stats, err
	}

	classificatorIDs, err := loadExistingClassificatorIDs(ctx, db)
	if err != nil {
		return stats, err
	}

	for _, item := range items {
		var classificator any
		if classificatorIDs[item.Classificator] {
			classificator = item.Classificator
		} else {
			stats.MissingClassificator++
			log.Printf("device=%s has missing classificator=%s; importing with classificator=NULL", item.ID, item.Classificator)
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO devices (id, classificator, serial_number, properties, connected_to_lis, is_used)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO UPDATE
			 SET classificator = EXCLUDED.classificator,
			     serial_number = EXCLUDED.serial_number,
			     properties = EXCLUDED.properties,
			     connected_to_lis = EXCLUDED.connected_to_lis,
			     is_used = EXCLUDED.is_used`,
			item.ID,
			classificator,
			item.SerialNumber,
			[]byte(item.Properties),
			item.ConnectedToLis,
			item.IsUsed,
		); err != nil {
			return stats, fmt.Errorf("import device %s: %w", item.ID, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureDevicesSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS devices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			classificator UUID REFERENCES classificators(id) ON DELETE SET NULL,
			serial_number TEXT NOT NULL DEFAULT '',
			properties JSONB NOT NULL DEFAULT '{}',
			connected_to_lis BOOLEAN NOT NULL DEFAULT FALSE,
			is_used BOOLEAN NOT NULL DEFAULT FALSE
		)`,
	); err != nil {
		return fmt.Errorf("ensure devices table: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ALTER COLUMN id SET DEFAULT gen_random_uuid()`); err != nil {
		return fmt.Errorf("ensure devices.id default: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ADD COLUMN IF NOT EXISTS classificator UUID REFERENCES classificators(id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("ensure devices.classificator column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ADD COLUMN IF NOT EXISTS serial_number TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure devices.serial_number column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ADD COLUMN IF NOT EXISTS properties JSONB NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("ensure devices.properties column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ADD COLUMN IF NOT EXISTS connected_to_lis BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		return fmt.Errorf("ensure devices.connected_to_lis column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE devices ADD COLUMN IF NOT EXISTS is_used BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		return fmt.Errorf("ensure devices.is_used column: %w", err)
	}

	return nil
}

func loadExistingClassificatorIDs(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT id FROM classificators`)
	if err != nil {
		return nil, fmt.Errorf("load classificators: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan classificator id: %w", err)
		}
		ids[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classificators: %w", err)
	}

	return ids, nil
}

func normalizeJSON(raw json.RawMessage, fallback []byte) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return append(json.RawMessage(nil), fallback...)
	}

	return append(json.RawMessage(nil), raw...)
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
