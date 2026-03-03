package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type demoAccount struct {
	userID     string
	username   string
	password   string
	firstName  string
	lastName   string
	email      string
	department string
}

var demoAccounts = []demoAccount{
	{
		userID:     "11111111-1111-1111-1111-111111111111",
		username:   "mobile.lead",
		password:   "Alpha123!",
		firstName:  "Maya",
		lastName:   "Hernandez",
		email:      "maya.hernandez@foxygen.dev",
		department: "Mobile Engineering",
	},
	{
		userID:     "22222222-2222-2222-2222-222222222222",
		username:   "qa.runner",
		password:   "Beta123!",
		firstName:  "Jordan",
		lastName:   "Lee",
		email:      "jordan.lee@foxygen.dev",
		department: "Quality Assurance",
	},
	{
		userID:     "33333333-3333-3333-3333-333333333333",
		username:   "ops.viewer",
		password:   "Gamma123!",
		firstName:  "Priya",
		lastName:   "Nair",
		email:      "priya.nair@foxygen.dev",
		department: "Operations",
	},
}

func findDemoAccount(username string) (demoAccount, bool) {
	for _, account := range demoAccounts {
		if strings.EqualFold(account.username, username) {
			return account, true
		}
	}

	return demoAccount{}, false
}

func findDemoAccountByUserID(userID string) (demoAccount, bool) {
	for _, account := range demoAccounts {
		if account.userID == userID {
			return account, true
		}
	}

	return demoAccount{}, false
}

func (s *Server) ensureDemoAccounts(ctx context.Context) error {
	for _, account := range demoAccounts {
		stored, err := s.queries.GetAccountByUsername(ctx, account.username)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("load demo account %s: %w", account.username, err)
			}

			passwordHash, hashErr := hashPassword(account.password)
			if hashErr != nil {
				return fmt.Errorf("hash demo account %s: %w", account.username, hashErr)
			}

			stored, err = s.queries.CreateAccount(ctx, appdb.CreateAccountParams{
				Username:     account.username,
				PasswordHash: passwordHash,
			})
			if err != nil {
				var pgErr *pgconn.PgError
				if !(errors.As(err, &pgErr) && pgErr.Code == "23505") {
					return fmt.Errorf("create demo account %s: %w", account.username, err)
				}

				stored, err = s.queries.GetAccountByUsername(ctx, account.username)
				if err != nil {
					return fmt.Errorf("reload demo account %s: %w", account.username, err)
				}
			}
		}

		if _, err := s.db.Exec(
			ctx,
			`INSERT INTO users (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
			stored.UserID,
		); err != nil {
			return fmt.Errorf("ensure demo profile %s: %w", account.username, err)
		}

		if _, err := s.db.Exec(
			ctx,
			`INSERT INTO departments (title) VALUES ($1) ON CONFLICT (title) DO NOTHING`,
			account.department,
		); err != nil {
			return fmt.Errorf("ensure department for %s: %w", account.username, err)
		}

		if _, err := s.db.Exec(
			ctx,
			`UPDATE users
			SET first_name = $1,
			    last_name = $2,
			    email = $3,
			    department_id = (SELECT id FROM departments WHERE title = $4)
			WHERE user_id = $5`,
			account.firstName,
			account.lastName,
			account.email,
			account.department,
			stored.UserID,
		); err != nil {
			return fmt.Errorf("seed profile for %s: %w", account.username, err)
		}
	}

	return nil
}
