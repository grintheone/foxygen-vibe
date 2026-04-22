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
	return EnsureSchemaWithSessionSettings(ctx, db, schemaGlob, nil)
}

func EnsureSchemaWithSessionSettings(ctx context.Context, db *pgxpool.Pool, schemaGlob string, sessionSettings map[string]string) error {
	schemaFiles, err := filepath.Glob(schemaGlob)
	if err != nil {
		return fmt.Errorf("list schema files: %w", err)
	}

	sort.Strings(schemaFiles)

	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire schema connection: %w", err)
	}
	defer conn.Release()

	for key, value := range sessionSettings {
		if _, execErr := conn.Exec(ctx, `SELECT set_config($1, $2, false)`, key, value); execErr != nil {
			return fmt.Errorf("set schema session setting %s: %w", key, execErr)
		}
	}

	for _, schemaFile := range schemaFiles {
		schema, readErr := os.ReadFile(schemaFile)
		if readErr != nil {
			return fmt.Errorf("read schema %s: %w", schemaFile, readErr)
		}

		if _, execErr := conn.Exec(ctx, string(schema)); execErr != nil {
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
