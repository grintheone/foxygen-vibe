package main

import (
	"context"
	"crypto/sha1"
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
	ID       string             `json:"_id"`
	Bindings []legacyDeviceBind `json:"bindings"`
}

type legacyDeviceBind struct {
	Client string `json:"client"`
}

type legacyAgreement struct {
	ID       string
	ClientID string
	DeviceID string
}

type agreementImportStats struct {
	Found         int
	Imported      int
	MissingClient int
	MissingDevice int
}

const agreementNamespace = "foxygen-vibe/device-binding-agreements"

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

	items, err := loadLegacyAgreements(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy device bindings in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items[:min(10, len(items))] {
			log.Printf("dry-run agreement=%s device=%s client=%s", item.ID, item.DeviceID, item.ClientID)
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

	stats, err := importAgreements(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d imported=%d missing_client=%d missing_device=%d", stats.Found, stats.Imported, stats.MissingClient, stats.MissingDevice)
}

func loadLegacyAgreements(path string) ([]legacyAgreement, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyAgreement, 0)
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
			return nil, fmt.Errorf("parse device bindings %s: %w", meta.ID, err)
		}

		deviceID := trimLegacyPrefix(doc.ID)
		if deviceID == "" {
			continue
		}

		for _, binding := range doc.Bindings {
			clientID := strings.TrimSpace(binding.Client)
			if clientID == "" {
				continue
			}

			items = append(items, legacyAgreement{
				ID:       deterministicAgreementID(deviceID, clientID),
				ClientID: clientID,
				DeviceID: deviceID,
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DeviceID != items[j].DeviceID {
			return items[i].DeviceID < items[j].DeviceID
		}
		if items[i].ClientID != items[j].ClientID {
			return items[i].ClientID < items[j].ClientID
		}
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importAgreements(ctx context.Context, db *pgxpool.Pool, items []legacyAgreement) (agreementImportStats, error) {
	stats := agreementImportStats{Found: len(items)}

	if err := ensureAgreementsSchema(ctx, db); err != nil {
		return stats, err
	}

	clientIDs, err := loadIDSet(ctx, db, `SELECT id FROM clients`)
	if err != nil {
		return stats, fmt.Errorf("load clients: %w", err)
	}

	deviceIDs, err := loadIDSet(ctx, db, `SELECT id FROM devices`)
	if err != nil {
		return stats, fmt.Errorf("load devices: %w", err)
	}

	for _, item := range items {
		var clientID any
		if clientIDs[item.ClientID] {
			clientID = item.ClientID
		} else {
			stats.MissingClient++
		}

		var deviceID any
		if deviceIDs[item.DeviceID] {
			deviceID = item.DeviceID
		} else {
			stats.MissingDevice++
		}

		if clientID == nil || deviceID == nil {
			log.Printf("agreement=%s has unresolved binding device=%s client=%s; skipping", item.ID, item.DeviceID, item.ClientID)
			continue
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO agreements (
				id,
				actual_client,
				distributor,
				device,
				assigned_at,
				finished_at,
				is_active,
				on_warranty,
				type
			)
			 VALUES ($1, $2, NULL, $3, NULL, NULL, TRUE, TRUE, $4)
			 ON CONFLICT (id) DO UPDATE
			 SET actual_client = EXCLUDED.actual_client,
			     distributor = EXCLUDED.distributor,
			     device = EXCLUDED.device,
			     assigned_at = EXCLUDED.assigned_at,
			     finished_at = EXCLUDED.finished_at,
			     is_active = EXCLUDED.is_active,
			     on_warranty = EXCLUDED.on_warranty,
			     type = EXCLUDED.type`,
			item.ID,
			clientID,
			deviceID,
			"binding",
		); err != nil {
			return stats, fmt.Errorf("import agreement %s: %w", item.ID, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureAgreementsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS agreements (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			number INT GENERATED ALWAYS AS IDENTITY,
			actual_client UUID REFERENCES clients(id) ON DELETE SET NULL,
			distributor UUID REFERENCES clients(id) ON DELETE SET NULL,
			device UUID REFERENCES devices(id) ON DELETE SET NULL,
			assigned_at TIMESTAMP DEFAULT NULL,
			finished_at TIMESTAMP DEFAULT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			on_warranty BOOLEAN NOT NULL DEFAULT TRUE,
			type VARCHAR(128)
		)`,
	); err != nil {
		return fmt.Errorf("ensure agreements table: %w", err)
	}

	statements := []struct {
		query string
		label string
	}{
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS number INT GENERATED ALWAYS AS IDENTITY`, "number"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS actual_client UUID REFERENCES clients(id) ON DELETE SET NULL`, "actual_client"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS distributor UUID REFERENCES clients(id) ON DELETE SET NULL`, "distributor"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS device UUID REFERENCES devices(id) ON DELETE SET NULL`, "device"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMP DEFAULT NULL`, "assigned_at"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS finished_at TIMESTAMP DEFAULT NULL`, "finished_at"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE`, "is_active"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS on_warranty BOOLEAN NOT NULL DEFAULT TRUE`, "on_warranty"},
		{`ALTER TABLE agreements ADD COLUMN IF NOT EXISTS type VARCHAR(128)`, "type"},
	}

	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt.query); err != nil {
			return fmt.Errorf("ensure agreements.%s column: %w", stmt.label, err)
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

func deterministicAgreementID(deviceID, clientID string) string {
	sum := sha1.Sum([]byte(agreementNamespace + ":" + deviceID + ":" + clientID))
	b := sum[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	)
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
