package users

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(sourcePath, defaultPassword string, dryRun bool, timeout, perUserTimeout time.Duration) error {
	if strings.TrimSpace(sourcePath) == "" {
		return fmt.Errorf("missing required source path")
	}
	if strings.TrimSpace(defaultPassword) == "" {
		return fmt.Errorf("missing required default password")
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	if perUserTimeout <= 0 {
		perUserTimeout = 5 * time.Second
	}

	plan, err := loadLegacyImportPlan(sourcePath)
	if err != nil {
		return err
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
		return nil
	}

	db, ctx, cancel, err := openDatabase(timeout)
	if err != nil {
		return err
	}
	defer cancel()
	defer db.Close()

	stats, err := importUsers(ctx, db, plan, defaultPassword, perUserTimeout)
	if err != nil {
		return err
	}

	log.Printf("import complete: found=%d departments=%d created=%d updated=%d skipped=%d failures=%d", stats.Found, stats.Departments, stats.Created, stats.Updated, stats.Skipped, stats.Failures)
	return nil
}

func openDatabase(timeout time.Duration) (*pgxpool.Pool, context.Context, context.CancelFunc, error) {
	databaseURL, err := resolveDatabaseURL(".env")
	if err != nil {
		return nil, nil, nil, err
	}
	if databaseURL == "" {
		return nil, nil, nil, fmt.Errorf("database is not configured; set DATABASE_URL or DB_* in server/.env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		cancel()
		return nil, nil, nil, err
	}

	return db, ctx, cancel, nil
}
