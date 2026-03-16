package dbinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func EnsureSchema(ctx context.Context, db *pgxpool.Pool, schemaGlob string) error {
	schemaFiles, err := filepath.Glob(schemaGlob)
	if err != nil {
		return fmt.Errorf("list schema files: %w", err)
	}

	sort.Strings(schemaFiles)

	for _, schemaFile := range schemaFiles {
		schema, readErr := os.ReadFile(schemaFile)
		if readErr != nil {
			return fmt.Errorf("read schema %s: %w", schemaFile, readErr)
		}

		if _, execErr := db.Exec(ctx, string(schema)); execErr != nil {
			return fmt.Errorf("apply schema %s: %w", schemaFile, execErr)
		}
	}

	return nil
}

func DatabaseNeedsImport(ctx context.Context, db *pgxpool.Pool) (bool, error) {
	var hasData bool

	if err := db.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM accounts LIMIT 1)
			OR EXISTS (SELECT 1 FROM clients LIMIT 1)
			OR EXISTS (SELECT 1 FROM tickets LIMIT 1)
			OR EXISTS (SELECT 1 FROM agreements LIMIT 1)
	`).Scan(&hasData); err != nil {
		return false, fmt.Errorf("check existing application data: %w", err)
	}

	return !hasData, nil
}
