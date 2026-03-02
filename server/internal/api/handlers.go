package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Status   string `json:"status"`
		Database struct {
			Configured bool `json:"configured"`
			Connected  bool `json:"connected"`
		} `json:"database"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := response{Status: "ok"}
	payload.Database.Configured = s.databaseConfigured
	payload.Database.Connected = s.db != nil

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAccounts(w http.ResponseWriter, r *http.Request) {
	type createAccountRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type createAccountResponse struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Disabled bool   `json:"disabled"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.queries == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}

	defer r.Body.Close()

	var input createAccountRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)

	switch {
	case input.Username == "":
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	case input.Password == "":
		http.Error(w, "password is required", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		log.Printf("password hash failed: %v", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	account, err := s.queries.CreateAccount(ctx, appdb.CreateAccountParams{
		Username:     input.Username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}

		log.Printf("create account failed: %v", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, createAccountResponse{
		UserID:   account.UserID.String(),
		Username: account.Username,
		Disabled: account.Disabled,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	digest := sha256.Sum256(append(salt, []byte(password)...))

	return fmt.Sprintf(
		"sha256$%s$%s",
		hex.EncodeToString(salt),
		hex.EncodeToString(digest[:]),
	), nil
}
