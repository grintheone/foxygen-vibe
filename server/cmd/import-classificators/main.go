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

type legacyClassificatorDoc struct {
	ID                      string          `json:"_id"`
	Title                   string          `json:"title"`
	Manufacturer            json.RawMessage `json:"manufacturer"`
	ResearchType            string          `json:"researchType"`
	RegistrationCertificate json.RawMessage `json:"registrationCertificate"`
	MaintenanceRegulations  json.RawMessage `json:"maintenanceRegulations"`
	Attachments             []string        `json:"attachments"`
	Images                  []string        `json:"images"`
}

type legacyClassificator struct {
	ID                      string
	Title                   string
	ManufacturerID          string
	ResearchTypeID          string
	RegistrationCertificate json.RawMessage
	MaintenanceRegulations  json.RawMessage
	Attachments             []string
	Images                  []string
}

type classificatorImportStats struct {
	Found               int
	Imported            int
	WithoutManufacturer int
	WithoutResearchType int
	MissingManufacturer int
	MissingResearchType int
}

type manufacturerRef struct {
	ID string `json:"id"`
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

	items, err := loadLegacyClassificators(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy classificators in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items {
			log.Printf("dry-run classificator=%s manufacturer=%q research_type=%q title=%q", item.ID, item.ManufacturerID, item.ResearchTypeID, item.Title)
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

	stats, err := importClassificators(ctx, db, items)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"import complete: found=%d imported=%d without_manufacturer=%d without_research_type=%d missing_manufacturer=%d missing_research_type=%d",
		stats.Found,
		stats.Imported,
		stats.WithoutManufacturer,
		stats.WithoutResearchType,
		stats.MissingManufacturer,
		stats.MissingResearchType,
	)
}

func loadLegacyClassificators(path string) ([]legacyClassificator, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyClassificator, 0)
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
		if !strings.HasPrefix(meta.ID, "classificator_") {
			continue
		}

		var doc legacyClassificatorDoc
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse classificator document %s: %w", meta.ID, err)
		}

		id := trimLegacyPrefix(doc.ID)
		title := normalizeWhitespace(doc.Title)
		if id == "" || title == "" {
			continue
		}

		manufacturerID, err := parseManufacturerID(doc.Manufacturer)
		if err != nil {
			return nil, fmt.Errorf("parse classificator manufacturer %s: %w", meta.ID, err)
		}

		items = append(items, legacyClassificator{
			ID:                      id,
			Title:                   title,
			ManufacturerID:          manufacturerID,
			ResearchTypeID:          strings.TrimSpace(doc.ResearchType),
			RegistrationCertificate: normalizeJSON(doc.RegistrationCertificate, []byte(`{}`)),
			MaintenanceRegulations:  normalizeJSON(doc.MaintenanceRegulations, []byte(`[]`)),
			Attachments:             cloneStrings(doc.Attachments),
			Images:                  cloneStrings(doc.Images),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func importClassificators(ctx context.Context, db *pgxpool.Pool, items []legacyClassificator) (classificatorImportStats, error) {
	stats := classificatorImportStats{Found: len(items)}

	if err := ensureClassificatorsSchema(ctx, db); err != nil {
		return stats, err
	}

	manufacturerIDs, err := loadExistingIDs(ctx, db, "SELECT id FROM manufacturers")
	if err != nil {
		return stats, fmt.Errorf("load manufacturers: %w", err)
	}

	researchTypeIDs, err := loadExistingIDs(ctx, db, "SELECT id FROM research_type")
	if err != nil {
		return stats, fmt.Errorf("load research types: %w", err)
	}

	for _, item := range items {
		var manufacturerID any
		switch {
		case item.ManufacturerID == "":
			stats.WithoutManufacturer++
		case manufacturerIDs[item.ManufacturerID]:
			manufacturerID = item.ManufacturerID
		default:
			stats.MissingManufacturer++
			log.Printf("classificator=%s has missing manufacturer=%s; importing with manufacturer=NULL", item.ID, item.ManufacturerID)
		}

		var researchTypeID any
		switch {
		case item.ResearchTypeID == "":
			stats.WithoutResearchType++
		case researchTypeIDs[item.ResearchTypeID]:
			researchTypeID = item.ResearchTypeID
		default:
			stats.MissingResearchType++
			log.Printf("classificator=%s has missing research_type=%s; importing with research_type=NULL", item.ID, item.ResearchTypeID)
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO classificators (
				id,
				title,
				manufacturer,
				research_type,
				registration_certificate,
				maintenance_regulations,
				attachments,
				images
			)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (id) DO UPDATE
			 SET title = EXCLUDED.title,
			     manufacturer = EXCLUDED.manufacturer,
			     research_type = EXCLUDED.research_type,
			     registration_certificate = EXCLUDED.registration_certificate,
			     maintenance_regulations = EXCLUDED.maintenance_regulations,
			     attachments = EXCLUDED.attachments,
			     images = EXCLUDED.images`,
			item.ID,
			item.Title,
			manufacturerID,
			researchTypeID,
			[]byte(item.RegistrationCertificate),
			[]byte(item.MaintenanceRegulations),
			item.Attachments,
			item.Images,
		); err != nil {
			return stats, fmt.Errorf("import classificator %s (%s): %w", item.ID, item.Title, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureClassificatorsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS classificators (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title TEXT NOT NULL DEFAULT '',
			manufacturer UUID REFERENCES manufacturers(id) ON DELETE SET NULL,
			research_type UUID REFERENCES research_type(id) ON DELETE SET NULL,
			registration_certificate JSONB NOT NULL DEFAULT '{}',
			maintenance_regulations JSONB NOT NULL DEFAULT '{}',
			attachments TEXT[] NOT NULL DEFAULT '{}',
			images TEXT[] NOT NULL DEFAULT '{}'
		)`,
	); err != nil {
		return fmt.Errorf("ensure classificators table: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE classificators
		 ALTER COLUMN id SET DEFAULT gen_random_uuid()`,
	); err != nil {
		return fmt.Errorf("ensure classificators.id default: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("ensure classificators.title column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS manufacturer UUID REFERENCES manufacturers(id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("ensure classificators.manufacturer column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS research_type UUID REFERENCES research_type(id) ON DELETE SET NULL`); err != nil {
		return fmt.Errorf("ensure classificators.research_type column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS registration_certificate JSONB NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("ensure classificators.registration_certificate column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS maintenance_regulations JSONB NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("ensure classificators.maintenance_regulations column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS attachments TEXT[] NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("ensure classificators.attachments column: %w", err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE classificators ADD COLUMN IF NOT EXISTS images TEXT[] NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("ensure classificators.images column: %w", err)
	}

	return nil
}

func loadExistingIDs(ctx context.Context, db *pgxpool.Pool, query string) (map[string]bool, error) {
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

func parseManufacturerID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return strings.TrimSpace(direct), nil
	}

	var ref manufacturerRef
	if err := json.Unmarshal(raw, &ref); err == nil {
		return strings.TrimSpace(ref.ID), nil
	}

	return "", fmt.Errorf("unsupported manufacturer payload: %s", string(raw))
}

func normalizeJSON(raw json.RawMessage, fallback []byte) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return append(json.RawMessage(nil), fallback...)
	}

	return append(json.RawMessage(nil), raw...)
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	cloned := make([]string, 0, len(values))
	for _, value := range values {
		cloned = append(cloned, strings.TrimSpace(value))
	}

	return cloned
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
