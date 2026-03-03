package classificators

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(sourcePath string, dryRun bool, timeout time.Duration) error {
	if strings.TrimSpace(sourcePath) == "" {
		return fmt.Errorf("missing required source path")
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	items, err := loadLegacyClassificators(sourcePath)
	if err != nil {
		return err
	}

	log.Printf("discovered %d legacy classificators in %s", len(items), sourcePath)

	if dryRun {
		for _, item := range items {
			log.Printf("dry-run classificator=%s manufacturer=%q research_type=%q title=%q", item.ID, item.ManufacturerID, item.ResearchTypeID, item.Title)
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

	stats, err := importClassificators(ctx, db, items)
	if err != nil {
		return err
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
