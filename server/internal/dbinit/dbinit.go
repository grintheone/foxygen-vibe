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
	for _, table := range []string{"accounts", "clients", "tickets", "agreements"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return false, fmt.Errorf("check existing application data: %w", err)
		}
		if !exists {
			continue
		}

		var hasRows bool
		query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM public.%s LIMIT 1)`, table)
		if err := db.QueryRow(ctx, query).Scan(&hasRows); err != nil {
			return false, fmt.Errorf("check existing application data: %w", err)
		}
		if hasRows {
			return false, nil
		}
	}

	return true, nil
}

func tableExists(ctx context.Context, db *pgxpool.Pool, table string) (bool, error) {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
