package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

var (
	errInvalidToken = errors.New("invalid token")
	errExpiredToken = errors.New("expired token")
)

type accessTokenClaims struct {
	Subject   string `json:"sub"`
	Username  string `json:"username"`
	TokenType string `json:"typ"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
}

func (s *Server) issueTokenPair(ctx context.Context, store accountStore, account appdb.Account) (tokenResponse, error) {
	return s.issueTokenPairForIdentity(ctx, store, uuidToString(account.UserID), account.Username)
}

func (s *Server) issueTokenPairForIdentity(ctx context.Context, store accountStore, userID string, username string) (tokenResponse, error) {
	issuedAt := time.Now().UTC()
	accessExpiresAt := issuedAt.Add(s.auth.accessTokenTTL)
	refreshExpiresAt := issuedAt.Add(s.auth.refreshTokenTTL)

	accessToken, err := signJWT(s.auth.jwtSecret, accessTokenClaims{
		Subject:   userID,
		Username:  username,
		TokenType: accessTokenType,
		ExpiresAt: accessExpiresAt.Unix(),
		IssuedAt:  issuedAt.Unix(),
	})
	if err != nil {
		return tokenResponse{}, err
	}

	var refreshToken string
	if store == nil {
		refreshToken, err = signJWT(s.auth.jwtSecret, accessTokenClaims{
			Subject:   userID,
			Username:  username,
			TokenType: refreshTokenType,
			ExpiresAt: refreshExpiresAt.Unix(),
			IssuedAt:  issuedAt.Unix(),
		})
		if err != nil {
			return tokenResponse{}, err
		}
	} else {
		var opaqueToken string
		opaqueToken, err = generateOpaqueToken()
		if err != nil {
			return tokenResponse{}, err
		}

		if _, err := store.CreateRefreshToken(ctx, appdb.CreateRefreshTokenParams{
			UserID:    accountIDToUUID(userID),
			TokenHash: hashOpaqueToken(opaqueToken),
			ExpiresAt: timestamptz(refreshExpiresAt),
		}); err != nil {
			return tokenResponse{}, err
		}

		refreshToken = opaqueToken
	}

	return tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.auth.accessTokenTTL / time.Second),
		UserID:       userID,
		Username:     username,
	}, nil
}

func signJWT(secret []byte, claims accessTokenClaims) (string, error) {
	header, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(header)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	unsigned := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", err
	}

	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, nil
}

func verifyJWT(secret []byte, token string, expectedType string) (accessTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return accessTokenClaims{}, errInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return accessTokenClaims{}, errInvalidToken
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return accessTokenClaims{}, errInvalidToken
	}

	if !hmac.Equal(signature, mac.Sum(nil)) {
		return accessTokenClaims{}, errInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return accessTokenClaims{}, errInvalidToken
	}

	var claims accessTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return accessTokenClaims{}, errInvalidToken
	}

	if claims.TokenType != expectedType || claims.Subject == "" || claims.ExpiresAt == 0 {
		return accessTokenClaims{}, errInvalidToken
	}

	if time.Now().UTC().Unix() >= claims.ExpiresAt {
		return accessTokenClaims{}, errExpiredToken
	}

	return claims, nil
}

func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func hashOpaqueToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
	}
}

func bearerToken(raw string) (string, error) {
	scheme, token, ok := strings.Cut(raw, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return "", errInvalidToken
	}

	return strings.TrimSpace(token), nil
}

func parseAuthorizationHeader(secret []byte, raw string) (accessTokenClaims, error) {
	token, err := bearerToken(raw)
	if err != nil {
		return accessTokenClaims{}, err
	}

	return verifyJWT(secret, token, accessTokenType)
}

func parseRefreshToken(secret []byte, raw string) (accessTokenClaims, error) {
	return verifyJWT(secret, raw, refreshTokenType)
}

func accountIDToUUID(raw string) pgtype.UUID {
	var value pgtype.UUID
	if err := value.Scan(raw); err != nil {
		return pgtype.UUID{}
	}

	return value
}

func validateStoredRefreshToken(record appdb.RefreshToken) error {
	if !record.ExpiresAt.Valid || time.Now().UTC().After(record.ExpiresAt.Time) {
		return errExpiredToken
	}
	if record.RotatedAt.Valid || record.RevokedAt.Valid {
		return errInvalidToken
	}

	return nil
}

func refreshConflict(rows int64) error {
	if rows == 0 {
		return errInvalidToken
	}

	return nil
}

func hashPasswordWithSalt(password string, salt []byte) string {
	digest := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(digest[:])
}

func parsePasswordHash(stored string) ([]byte, []byte, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 3 || parts[0] != "sha256" {
		return nil, nil, fmt.Errorf("invalid password hash")
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, nil, fmt.Errorf("decode password salt: %w", err)
	}

	expected, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, nil, fmt.Errorf("decode password digest: %w", err)
	}

	return salt, expected, nil
}

func verifyLegacyPassword(password string, stored string) bool {
	salt, expected, err := parsePasswordHash(stored)
	if err != nil {
		return false
	}

	digest, err := hex.DecodeString(hashPasswordWithSalt(password, salt))
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(digest[:], expected) == 1
}
