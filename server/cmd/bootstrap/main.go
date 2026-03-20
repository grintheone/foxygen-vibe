package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"foxygen-vibe/server/internal/dbinit"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	options, err := loadOptions()
	if err != nil {
		log.Fatal(err)
	}

	db, err := waitForDatabase(options.DatabaseURL, options.WaitTimeout, options.WaitInterval)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := dbinit.EnsureSchema(ctx, db, "db/schema/*.sql"); err != nil {
		log.Fatal(err)
	}

	if err := waitForStorage(options.StorageEndpoint, options.StorageUseSSL, options.StorageWaitTimeout, options.StorageWaitInterval); err != nil {
		log.Fatal(err)
	}

	if !options.ImportEnabled {
		log.Println("bootstrap import disabled; schema is ready")
		return
	}

	needsImport, err := dbinit.DatabaseNeedsImport(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	if !needsImport {
		log.Println("bootstrap import skipped; database already contains application data")
		return
	}

	if err := runImport(options); err != nil {
		log.Fatal(err)
	}

	log.Println("bootstrap import finished successfully")
}

type options struct {
	DatabaseURL           string
	WaitTimeout           time.Duration
	WaitInterval          time.Duration
	ImportEnabled         bool
	ImportSourcePath      string
	ImportDefaultPassword string
	ImportOnly            string
	ImportBinaryPath      string
	StorageEndpoint       string
	StorageUseSSL         bool
	StorageWaitTimeout    time.Duration
	StorageWaitInterval   time.Duration
}

func loadOptions() (options, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return options{}, err
	}
	if databaseURL == "" {
		return options{}, fmt.Errorf("database is not configured; set DATABASE_URL or DB_* variables")
	}

	waitTimeout, err := resolveDurationEnv("BOOTSTRAP_DB_WAIT_TIMEOUT", 2*time.Minute)
	if err != nil {
		return options{}, err
	}

	waitInterval, err := resolveDurationEnv("BOOTSTRAP_DB_WAIT_INTERVAL", 2*time.Second)
	if err != nil {
		return options{}, err
	}

	importEnabled, err := resolveBoolEnv("BOOTSTRAP_IMPORT_ENABLED", false)
	if err != nil {
		return options{}, err
	}

	storageWaitTimeout, err := resolveDurationEnv("BOOTSTRAP_STORAGE_WAIT_TIMEOUT", 1*time.Minute)
	if err != nil {
		return options{}, err
	}

	storageWaitInterval, err := resolveDurationEnv("BOOTSTRAP_STORAGE_WAIT_INTERVAL", 2*time.Second)
	if err != nil {
		return options{}, err
	}

	storageUseSSL, err := resolveBoolEnv("MINIO_USE_SSL", false)
	if err != nil {
		return options{}, err
	}

	importSourcePath := strings.TrimSpace(os.Getenv("BOOTSTRAP_IMPORT_SOURCE"))
	if importSourcePath == "" {
		importSourcePath = "/bootstrap/dump.json"
	}

	importBinaryPath := strings.TrimSpace(os.Getenv("BOOTSTRAP_IMPORT_BINARY"))
	if importBinaryPath == "" {
		importBinaryPath = "/app/import-dump"
	}

	importDefaultPassword := strings.TrimSpace(os.Getenv("BOOTSTRAP_IMPORT_DEFAULT_PASSWORD"))
	if importEnabled && importDefaultPassword == "" {
		return options{}, fmt.Errorf("BOOTSTRAP_IMPORT_DEFAULT_PASSWORD is required when BOOTSTRAP_IMPORT_ENABLED=true")
	}

	if importEnabled {
		if _, err := os.Stat(importSourcePath); err != nil {
			return options{}, fmt.Errorf("bootstrap import source %q is not available: %w", importSourcePath, err)
		}
	}

	return options{
		DatabaseURL:           databaseURL,
		WaitTimeout:           waitTimeout,
		WaitInterval:          waitInterval,
		ImportEnabled:         importEnabled,
		ImportSourcePath:      importSourcePath,
		ImportDefaultPassword: importDefaultPassword,
		ImportOnly:            strings.TrimSpace(os.Getenv("BOOTSTRAP_IMPORT_ONLY")),
		ImportBinaryPath:      importBinaryPath,
		StorageEndpoint:       strings.TrimSpace(os.Getenv("MINIO_ENDPOINT")),
		StorageUseSSL:         storageUseSSL,
		StorageWaitTimeout:    storageWaitTimeout,
		StorageWaitInterval:   storageWaitInterval,
	}, nil
}

func waitForDatabase(databaseURL string, timeout, interval time.Duration) (*pgxpool.Pool, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), minDuration(interval, 5*time.Second))
		db, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			err = db.Ping(ctx)
			if err == nil {
				cancel()
				return db, nil
			}
			db.Close()
		}
		cancel()

		lastErr = err
		log.Printf("waiting for database: %v", err)
		time.Sleep(interval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("database did not become ready before timeout")
	}

	return nil, fmt.Errorf("wait for database: %w", lastErr)
}

func waitForStorage(endpoint string, useSSL bool, timeout, interval time.Duration) error {
	if strings.TrimSpace(endpoint) == "" {
		return nil
	}

	dialAddress := storageDialAddress(endpoint, useSSL)
	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", dialAddress, minDuration(interval, 5*time.Second))
		if err == nil {
			_ = conn.Close()
			return nil
		}

		lastErr = err
		log.Printf("waiting for object storage at %s: %v", dialAddress, err)
		time.Sleep(interval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("object storage did not become ready before timeout")
	}

	return fmt.Errorf("wait for object storage: %w", lastErr)
}

func storageDialAddress(endpoint string, useSSL bool) string {
	if _, _, err := net.SplitHostPort(endpoint); err == nil {
		return endpoint
	}

	defaultPort := "80"
	if useSSL {
		defaultPort = "443"
	}

	return net.JoinHostPort(endpoint, defaultPort)
}

func runImport(options options) error {
	args := []string{
		"-source", options.ImportSourcePath,
		"-default-password", options.ImportDefaultPassword,
	}

	if options.ImportOnly != "" {
		args = append(args, "-only", options.ImportOnly)
	}

	cmd := exec.Command(options.ImportBinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	log.Printf("running bootstrap import from %s", options.ImportSourcePath)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run import command: %w", err)
	}

	return nil
}

func resolveDatabaseURL() (string, error) {
	fileEnv, err := loadDotEnv(".env")
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

func resolveDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: parse duration: %w", key, err)
	}

	return duration, nil
}

func resolveBoolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: parse bool: %w", key, err)
	}

	return parsed, nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}

	return b
}
