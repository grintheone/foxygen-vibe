package tickets

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
		timeout = 10 * time.Minute
	}

	items, stats, err := loadLegacyTickets(sourcePath)
	if err != nil {
		return err
	}

	log.Printf(
		"discovered %d legacy tickets in %s (preserved_numbers=%d generated_numbers=%d mapped_fast=%d mapped_finished=%d mapped_reasons=%d)",
		len(items),
		sourcePath,
		stats.PreservedNumbers,
		stats.GeneratedNumbers,
		stats.MappedFastToInternal,
		stats.MappedFinishedStatus,
		stats.MappedReasonRepair+stats.MappedReasonMaintain,
	)

	if dryRun {
		for _, item := range items[:min(10, len(items))] {
			log.Printf("dry-run ticket=%s number=%v type=%q status=%q reason=%q", item.ID, item.Number, item.TicketType, item.Status, item.ReasonID)
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

	importStats, err := importTickets(ctx, db, items, stats)
	if err != nil {
		return err
	}

	log.Printf(
		"import complete: found=%d imported=%d preserved_numbers=%d generated_numbers=%d missing_client=%d missing_device=%d missing_author=%d missing_department=%d missing_assigned_by=%d missing_reason=%d missing_contact_person=%d missing_executor=%d missing_ticket_type=%d missing_status=%d",
		importStats.Found,
		importStats.Imported,
		importStats.PreservedNumbers,
		importStats.GeneratedNumbers,
		importStats.MissingClient,
		importStats.MissingDevice,
		importStats.MissingAuthor,
		importStats.MissingDepartment,
		importStats.MissingAssignedBy,
		importStats.MissingReason,
		importStats.MissingContactPerson,
		importStats.MissingExecutor,
		importStats.MissingTicketType,
		importStats.MissingStatus,
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
