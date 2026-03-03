package contacts

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

type legacyContactDoc struct {
	ID         string `json:"_id"`
	Ref        string `json:"ref"`
	FirstName  string `json:"firstName"`
	MiddleName string `json:"middleName"`
	LastName   string `json:"lastName"`
	Position   string `json:"position"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
}

type legacyContact struct {
	ID       string
	ClientID string
	Name     string
	Position string
	Phone    string
	Email    string
}

type contactImportStats struct {
	Found           int
	Imported        int
	WithoutClient   int
	MissingClientID int
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

	contacts, err := loadLegacyContacts(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy contacts in %s", len(contacts), sourcePath)

	if dryRun {
		for _, contact := range contacts {
			log.Printf("dry-run contact=%s client=%q name=%q", contact.ID, contact.ClientID, contact.Name)
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

	stats, err := importContacts(ctx, db, contacts)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"import complete: found=%d imported=%d without_client=%d missing_client=%d",
		stats.Found,
		stats.Imported,
		stats.WithoutClient,
		stats.MissingClientID,
	)
}

func loadLegacyContacts(path string) ([]legacyContact, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, fmt.Errorf("parse dump: %w", err)
	}

	contacts := make([]legacyContact, 0)
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
		if !strings.HasPrefix(meta.ID, "contact_") {
			continue
		}

		var doc legacyContactDoc
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, fmt.Errorf("parse contact document %s: %w", meta.ID, err)
		}

		contactID := trimLegacyPrefix(doc.ID)
		if contactID == "" {
			continue
		}

		contacts = append(contacts, legacyContact{
			ID:       contactID,
			ClientID: strings.TrimSpace(doc.Ref),
			Name:     buildContactName(doc.FirstName, doc.MiddleName, doc.LastName),
			Position: normalizeWhitespace(doc.Position),
			Phone:    strings.TrimSpace(doc.Phone),
			Email:    strings.TrimSpace(strings.ToLower(doc.Email)),
		})
	}

	sort.Slice(contacts, func(i, j int) bool {
		return contacts[i].ID < contacts[j].ID
	})

	return contacts, nil
}

func importContacts(ctx context.Context, db *pgxpool.Pool, contacts []legacyContact) (contactImportStats, error) {
	stats := contactImportStats{Found: len(contacts)}

	if err := ensureContactsSchema(ctx, db); err != nil {
		return stats, err
	}

	clientIDs, err := loadExistingClientIDs(ctx, db)
	if err != nil {
		return stats, err
	}

	for _, contact := range contacts {
		var clientID any
		switch {
		case contact.ClientID == "":
			stats.WithoutClient++
		case clientIDs[contact.ClientID]:
			clientID = contact.ClientID
		default:
			stats.MissingClientID++
			log.Printf("contact=%s has missing client=%s; importing with client_id=NULL", contact.ID, contact.ClientID)
		}

		if _, err := db.Exec(
			ctx,
			`INSERT INTO contacts (id, name, position, phone, email, client_id)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO UPDATE
			 SET name = EXCLUDED.name,
			     position = EXCLUDED.position,
			     phone = EXCLUDED.phone,
			     email = EXCLUDED.email,
			     client_id = EXCLUDED.client_id`,
			contact.ID,
			contact.Name,
			contact.Position,
			contact.Phone,
			contact.Email,
			clientID,
		); err != nil {
			return stats, fmt.Errorf("import contact %s (%s): %w", contact.ID, contact.Name, err)
		}

		stats.Imported++
	}

	return stats, nil
}

func ensureContactsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS contacts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL DEFAULT '',
			position TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			client_id UUID REFERENCES clients(id) ON DELETE CASCADE
		)`,
	); err != nil {
		return fmt.Errorf("ensure contacts table: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ALTER COLUMN id SET DEFAULT gen_random_uuid()`,
	); err != nil {
		return fmt.Errorf("ensure contacts.id default: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure contacts.name column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ADD COLUMN IF NOT EXISTS position TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure contacts.position column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure contacts.phone column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("ensure contacts.email column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE contacts
		 ADD COLUMN IF NOT EXISTS client_id UUID REFERENCES clients(id) ON DELETE CASCADE`,
	); err != nil {
		return fmt.Errorf("ensure contacts.client_id column: %w", err)
	}

	return nil
}

func loadExistingClientIDs(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT id FROM clients`)
	if err != nil {
		return nil, fmt.Errorf("load clients: %w", err)
	}
	defer rows.Close()

	clientIDs := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan client id: %w", err)
		}
		clientIDs[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate clients: %w", err)
	}

	return clientIDs, nil
}

func buildContactName(firstName, middleName, lastName string) string {
	name := normalizeWhitespace(strings.Join([]string{
		strings.TrimSpace(firstName),
		strings.TrimSpace(middleName),
		strings.TrimSpace(lastName),
	}, " "))

	return name
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
