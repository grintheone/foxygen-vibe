package api

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
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
