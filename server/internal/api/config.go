package api

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"foxygen-vibe/server/internal/storage"
)

type authConfig struct {
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
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

func resolveAuthConfig() (authConfig, error) {
	fileEnv, err := loadDotEnv(".env")
	if err != nil {
		return authConfig{}, err
	}

	secret := getConfigValue(fileEnv, "JWT_SECRET")
	if secret == "" {
		secret = "dev-only-jwt-secret-change-me"
	}

	accessTokenTTL, err := resolveDuration(fileEnv, "ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return authConfig{}, err
	}

	refreshTokenTTL, err := resolveDuration(fileEnv, "REFRESH_TOKEN_TTL", 7*24*time.Hour)
	if err != nil {
		return authConfig{}, err
	}

	return authConfig{
		jwtSecret:       []byte(secret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}, nil
}

func resolveStorageConfig() (storage.Config, error) {
	fileEnv, err := loadDotEnv(".env")
	if err != nil {
		return storage.Config{}, err
	}

	useSSLValue := getConfigValue(fileEnv, "MINIO_USE_SSL")
	region := getConfigValue(fileEnv, "MINIO_REGION")
	if region == "" {
		region = getConfigValue(fileEnv, "MINIO_LOCATION")
	}
	config := storage.Config{
		Endpoint:        getConfigValue(fileEnv, "MINIO_ENDPOINT"),
		AccessKeyID:     getConfigValue(fileEnv, "MINIO_ACCESS_KEY"),
		SecretAccessKey: getConfigValue(fileEnv, "MINIO_SECRET_KEY"),
		Bucket:          getConfigValue(fileEnv, "MINIO_BUCKET"),
		Region:          region,
	}

	useSSL, err := resolveBool(fileEnv, "MINIO_USE_SSL", false)
	if err != nil {
		return storage.Config{}, err
	}
	config.UseSSL = useSSL

	if !config.Enabled() && useSSLValue == "" {
		return config, nil
	}

	if err := config.Validate(); err != nil {
		return storage.Config{}, err
	}

	return config, nil
}

func resolveDuration(fileEnv map[string]string, key string, fallback time.Duration) (time.Duration, error) {
	value := getConfigValue(fileEnv, key)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: parse duration: %w", key, err)
	}

	return duration, nil
}

func resolveBool(fileEnv map[string]string, key string, fallback bool) (bool, error) {
	value := getConfigValue(fileEnv, key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: parse bool: %w", key, err)
	}

	return parsed, nil
}
