package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type legacyDump struct {
	Rows []legacyRow `json:"rows"`
}

type legacyRow struct {
	Doc json.RawMessage `json:"doc"`
}

type legacyDepartmentDoc struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
}

type legacyUserDoc struct {
	ID         string   `json:"_id"`
	FirstName  string   `json:"firstName"`
	LastName   string   `json:"lastName"`
	Department string   `json:"department"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
}

type legacyUser struct {
	SourceID         string
	FirstName        string
	LastName         string
	Email            string
	LegacyDepartment string
	DepartmentTitle  string
	RoleName         string
	Username         string
}

type importStats struct {
	Found       int
	Departments int
	Created     int
	Updated     int
	Skipped     int
	Failures    int
}

type importPlan struct {
	Departments map[string]string
	Users       []legacyUser
}

func main() {
	var (
		sourcePath      string
		defaultPassword string
		dryRun          bool
		commandTimeout  time.Duration
		perUserTimeout  time.Duration
	)

	flag.StringVar(&sourcePath, "source", "", "Path to the legacy CouchDB _all_docs JSON export")
	flag.StringVar(&defaultPassword, "default-password", "", "Temporary password to assign to imported users")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse and plan the import without writing to PostgreSQL")
	flag.DurationVar(&commandTimeout, "timeout", 2*time.Minute, "Overall import timeout")
	flag.DurationVar(&perUserTimeout, "per-user-timeout", 5*time.Second, "Timeout for each user transaction")
	flag.Parse()

	if strings.TrimSpace(sourcePath) == "" {
		log.Fatal("missing required -source")
	}
	if strings.TrimSpace(defaultPassword) == "" {
		log.Fatal("missing required -default-password")
	}

	plan, err := loadLegacyImportPlan(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("discovered %d legacy users and %d departments in %s", len(plan.Users), len(plan.Departments), sourcePath)

	if dryRun {
		for legacyID, title := range plan.Departments {
			log.Printf("dry-run department=%s title=%q", legacyID, title)
		}
		for _, user := range plan.Users {
			log.Printf("dry-run user=%s username=%s email=%q department=%q", user.SourceID, user.Username, user.Email, user.DepartmentTitle)
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

	stats, err := importUsers(ctx, db, plan, defaultPassword, perUserTimeout)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("import complete: found=%d departments=%d created=%d updated=%d skipped=%d failures=%d", stats.Found, stats.Departments, stats.Created, stats.Updated, stats.Skipped, stats.Failures)
}

func loadLegacyImportPlan(path string) (importPlan, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return importPlan{}, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return importPlan{}, fmt.Errorf("parse dump: %w", err)
	}

	departments := make(map[string]string)
	var docs []legacyUserDoc

	for _, row := range dump.Rows {
		if len(row.Doc) == 0 || string(row.Doc) == "null" {
			continue
		}

		var meta struct {
			ID string `json:"_id"`
		}
		if err := json.Unmarshal(row.Doc, &meta); err != nil {
			return importPlan{}, fmt.Errorf("parse document metadata: %w", err)
		}

		switch {
		case strings.HasPrefix(meta.ID, "department_"):
			var doc legacyDepartmentDoc
			if err := json.Unmarshal(row.Doc, &doc); err != nil {
				return importPlan{}, fmt.Errorf("parse department document %s: %w", meta.ID, err)
			}

			legacyID := trimLegacyPrefix(doc.ID)
			if legacyID != "" && strings.TrimSpace(doc.Title) != "" {
				departments[legacyID] = strings.TrimSpace(doc.Title)
			}
		case strings.HasPrefix(meta.ID, "user_"):
			var doc legacyUserDoc
			if err := json.Unmarshal(row.Doc, &doc); err != nil {
				return importPlan{}, fmt.Errorf("parse user document %s: %w", meta.ID, err)
			}
			docs = append(docs, doc)
		}
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].ID < docs[j].ID
	})

	used := make(map[string]int)
	users := make([]legacyUser, 0, len(docs))
	for _, doc := range docs {
		firstName := strings.TrimSpace(doc.FirstName)
		lastName := strings.TrimSpace(doc.LastName)
		email := strings.TrimSpace(strings.ToLower(doc.Email))
		departmentRef := strings.TrimSpace(doc.Department)
		departmentTitle := strings.TrimSpace(departments[departmentRef])
		username := uniqueUsername(fmt.Sprintf("user_%d", len(users)+1), used)

		users = append(users, legacyUser{
			SourceID:         doc.ID,
			FirstName:        firstName,
			LastName:         lastName,
			Email:            email,
			LegacyDepartment: departmentRef,
			DepartmentTitle:  departmentTitle,
			RoleName:         preferredRole(doc.Roles),
			Username:         username,
		})
	}

	return importPlan{
		Departments: departments,
		Users:       users,
	}, nil
}

func importUsers(ctx context.Context, db *pgxpool.Pool, plan importPlan, defaultPassword string, perUserTimeout time.Duration) (importStats, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return importStats{}, fmt.Errorf("hash default password: %w", err)
	}

	stats := importStats{Found: len(plan.Users)}
	if err := ensureImportSchema(ctx, db); err != nil {
		return stats, err
	}

	departmentIDs, err := importDepartments(ctx, db, plan.Departments)
	if err != nil {
		return stats, err
	}
	stats.Departments = len(departmentIDs)

	roleIDs, err := importRoles(ctx, db, plan.Users)
	if err != nil {
		return stats, err
	}

	for _, user := range plan.Users {
		if err := ctx.Err(); err != nil {
			return stats, err
		}

		userCtx, cancel := context.WithTimeout(ctx, perUserTimeout)
		created, err := importOneUser(userCtx, db, user, string(passwordHash), departmentIDs, roleIDs)
		cancel()

		switch {
		case err == nil:
			if created {
				stats.Created++
			} else {
				stats.Updated++
			}
		case errors.Is(err, pgx.ErrNoRows):
			stats.Skipped++
			log.Printf("skipped existing username=%s source=%s", user.Username, user.SourceID)
		default:
			stats.Failures++
			log.Printf("failed source=%s username=%s: %v", user.SourceID, user.Username, err)
		}
	}

	return stats, nil
}

func ensureImportSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`ALTER TABLE users
		 ADD COLUMN IF NOT EXISTS department UUID`,
	); err != nil {
		return fmt.Errorf("ensure users.department column: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`ALTER TABLE users
		 DROP COLUMN IF EXISTS department_id`,
	); err != nil {
		return fmt.Errorf("drop users.department_id column: %w", err)
	}

	return nil
}

func importDepartments(ctx context.Context, db *pgxpool.Pool, departments map[string]string) (map[string]string, error) {
	legacyIDs := make([]string, 0, len(departments))
	for legacyID := range departments {
		legacyIDs = append(legacyIDs, legacyID)
	}
	sort.Strings(legacyIDs)

	resolved := make(map[string]string, len(legacyIDs))
	for _, legacyID := range legacyIDs {
		title := strings.TrimSpace(departments[legacyID])
		if title == "" {
			continue
		}

		departmentID, err := findOrCreateDepartment(ctx, db, title)
		if err != nil {
			return nil, fmt.Errorf("import department %s (%s): %w", legacyID, title, err)
		}
		resolved[legacyID] = departmentID
	}

	return resolved, nil
}

func importRoles(ctx context.Context, db *pgxpool.Pool, users []legacyUser) (map[string]int, error) {
	roleNames := map[string]struct{}{
		"user": {},
	}

	for _, user := range users {
		roleNames[user.RoleName] = struct{}{}
	}

	ordered := make([]string, 0, len(roleNames))
	for roleName := range roleNames {
		ordered = append(ordered, roleName)
	}
	sort.Strings(ordered)

	resolved := make(map[string]int, len(ordered))
	for _, roleName := range ordered {
		roleID, err := findOrCreateRole(ctx, db, roleName)
		if err != nil {
			return nil, fmt.Errorf("import role %s: %w", roleName, err)
		}
		resolved[roleName] = roleID
	}

	return resolved, nil
}

func findOrCreateDepartment(ctx context.Context, db *pgxpool.Pool, title string) (string, error) {
	var id string
	err := db.QueryRow(ctx, `SELECT id FROM departments WHERE title = $1 ORDER BY id LIMIT 1`, title).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("query existing department: %w", err)
	}

	newID, err := newUUID()
	if err != nil {
		return "", fmt.Errorf("generate department id: %w", err)
	}

	err = db.QueryRow(ctx, `INSERT INTO departments (id, title) VALUES ($1, $2) RETURNING id`, newID, title).Scan(&id)
	if err == nil {
		return id, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		err = db.QueryRow(ctx, `SELECT id FROM departments WHERE title = $1 ORDER BY id LIMIT 1`, title).Scan(&id)
		if err == nil {
			return id, nil
		}
	}

	return "", fmt.Errorf("create department: %w", err)
}

func findOrCreateRole(ctx context.Context, db *pgxpool.Pool, roleName string) (int, error) {
	var id int
	err := db.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("query existing role: %w", err)
	}

	err = db.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM roles`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("compute next role id: %w", err)
	}

	_, err = db.Exec(ctx, `INSERT INTO roles (id, name, description) VALUES ($1, $2, $3)`, id, roleName, "")
	if err == nil {
		return id, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		err = db.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&id)
		if err == nil {
			return id, nil
		}
	}

	return 0, fmt.Errorf("create role: %w", err)
}

func importOneUser(ctx context.Context, db *pgxpool.Pool, user legacyUser, passwordHash string, departmentIDs map[string]string, roleIDs map[string]int) (bool, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var userID string
	created := true
	err = tx.QueryRow(
		ctx,
		`INSERT INTO accounts (username, password_hash)
		 VALUES ($1, $2)
		 ON CONFLICT (username) DO NOTHING
		 RETURNING user_id`,
		user.Username,
		passwordHash,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			created = false
			if err := tx.QueryRow(
				ctx,
				`SELECT user_id FROM accounts WHERE username = $1`,
				user.Username,
			).Scan(&userID); err != nil {
				return false, fmt.Errorf("load existing account: %w", err)
			}
		} else {
			return false, fmt.Errorf("create account: %w", err)
		}
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO users (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	); err != nil {
		return false, fmt.Errorf("create user profile: %w", err)
	}

	var departmentID any
	if user.LegacyDepartment != "" {
		if id, ok := departmentIDs[user.LegacyDepartment]; ok {
			departmentID = id
		}
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE users
		 SET first_name = $1,
		     last_name = $2,
		     email = $3,
		     department = $4
		 WHERE user_id = $5`,
		user.FirstName,
		user.LastName,
		user.Email,
		departmentID,
		userID,
	); err != nil {
		return false, fmt.Errorf("update user profile: %w", err)
	}

	roleID, ok := roleIDs[user.RoleName]
	if !ok {
		roleID = roleIDs["user"]
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO account_roles (user_id, role_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET role_id = EXCLUDED.role_id`,
		userID,
		roleID,
	); err != nil {
		return false, fmt.Errorf("upsert account role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}

	if created {
		log.Printf("created username=%s source=%s", user.Username, user.SourceID)
	} else {
		log.Printf("updated username=%s source=%s", user.Username, user.SourceID)
	}

	return created, nil
}

func preferredRole(roles []string) string {
	for _, role := range roles {
		role = strings.TrimSpace(strings.ToLower(role))
		switch role {
		case "":
			continue
		case "leader":
			return "admin"
		default:
			return role
		}
	}

	return "user"
}

func uniqueUsername(base string, used map[string]int) string {
	key := strings.ToLower(strings.TrimSpace(base))
	used[key]++
	if used[key] == 1 {
		return base
	}

	return fmt.Sprintf("%s-%d", base, used[key])
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

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	), nil
}
