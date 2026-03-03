package clients

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

type legacyClientDoc struct {
	ID               string          `json:"_id"`
	Title            string          `json:"title"`
	Address          string          `json:"address"`
	Region           string          `json:"region"`
	Location         json.RawMessage `json:"location"`
	LaboratorySystem *string         `json:"laboratorySystem"`
}

type legacyClient struct {
	ID               string
	Title            string
	Address          string
	RegionID         string
	Location         json.RawMessage
	LaboratorySystem *string
}

type clientImportStats struct {
	Found            int
	Imported         int
	WithoutRegion    int
	UnresolvedRegion int
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

	clients, err := loadLegacyClients(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy clients in %s", len(clients), sourcePath)

	if dryRun {
		for _, client := range clients {
			log.Printf("dry-run client=%s title=%q region=%q", client.ID, client.Title, client.RegionID)
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

	stats, err := importClients(ctx, db, clients)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"import complete: found=%d imported=%d without_region=%d unresolved_region=%d",
		stats.Found,
		stats.Imported,
		stats.WithoutRegion,
		stats.UnresolvedRegion,
	)
}

func loadLegacyClients(path string) ([]legacyClient, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	clients := make([]legacyClient, 0)
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
		if !strings.HasPrefix(meta.ID, "client_") {
			continue
		}

		var doc legacyClientDoc
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse client document %s: %w", meta.ID, err)
		}

		clientID := trimLegacyPrefix(doc.ID)
		title := normalizeWhitespace(doc.Title)
		if clientID == "" || title == "" {
			continue
		}

		var location json.RawMessage
		if len(doc.Location) > 0 && string(doc.Location) != "null" {
			location = append(json.RawMessage(nil), doc.Location...)
		}

		clients = append(clients, legacyClient{
			ID:               clientID,
			Title:            title,
			Address:          strings.TrimSpace(doc.Address),
			RegionID:         strings.TrimSpace(doc.Region),
			Location:         location,
			LaboratorySystem: normalizeNullableString(doc.LaboratorySystem),
		})
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ID < clients[j].ID
	})

	return clients, nil
}

func importClients(ctx context.Context, db *pgxpool.Pool, clients []legacyClient) (clientImportStats, error) {
	stats := clientImportStats{Found: len(clients)}

	if err := ensureClientsSchema(ctx, db); err != nil {
		return stats, err
	}

	regionIDs, err := loadExistingRegionIDs(ctx, db)
	if err != nil {
		return stats, err
	}

	for _, client := range clients {
		var regionID any
		switch {
		case client.RegionID == "":
			stats.WithoutRegion++
		case regionIDs[client.RegionID]:
			regionID = client.RegionID
		default:
			stats.UnresolvedRegion++
			log.Printf("client=%s has missing region=%s; importing with region=NULL", client.ID, client.RegionID)
		}

		var location any
		if len(client.Location) > 0 {
			location = []byte(client.Location)
		}

		var laboratorySystem any
		if client.LaboratorySystem != nil {
			laboratorySystem = *client.LaboratorySystem
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO clients (id, title, region, address, location, laboratory_system)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO UPDATE
			 SET title = EXCLUDED.title,
			     region = EXCLUDED.region,
			     address = EXCLUDED.address,
			     location = EXCLUDED.location,
			     laboratory_system = EXCLUDED.laboratory_system`,
			client.ID,
			client.Title,
			regionID,
			client.Address,
			location,
			laboratorySystem,
		); err != nil {
			return stats, fmt.Errorf("import client %s (%s): %w", client.ID, client.Title, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureClientsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS clients (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL,
			region UUID REFERENCES regions(id) ON DELETE SET NULL,
			address TEXT,
			location JSONB DEFAULT NULL,
			laboratory_system UUID DEFAULT NULL,
			manager UUID[] NOT NULL DEFAULT '{}'
		)`,
	); err != nil {
		return fmt.Errorf("ensure clients table: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ALTER COLUMN id SET DEFAULT gen_random_uuid()`,
	); err != nil {
		return fmt.Errorf("ensure clients.id default: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure clients.title column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS region UUID REFERENCES regions(id) ON DELETE SET NULL`,
	); err != nil {
		return fmt.Errorf("ensure clients.region column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS address TEXT`,
	); err != nil {
		return fmt.Errorf("ensure clients.address column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS location JSONB DEFAULT NULL`,
	); err != nil {
		return fmt.Errorf("ensure clients.location column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS laboratory_system UUID DEFAULT NULL`,
	); err != nil {
		return fmt.Errorf("ensure clients.laboratory_system column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE clients
		 ADD COLUMN IF NOT EXISTS manager UUID[] NOT NULL DEFAULT '{}'`,
	); err != nil {
		return fmt.Errorf("ensure clients.manager column: %w", err)
	}

	return nil
}

func loadExistingRegionIDs(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT id FROM regions`)
	if err != nil {
		return nil, fmt.Errorf("load regions: %w", err)
	}
	defer rows.Close()

	regionIDs := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan region id: %w", err)
		}
		regionIDs[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate regions: %w", err)
	}

	return regionIDs, nil
}

func normalizeNullableString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
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
