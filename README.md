# Foxygen Vibe

Minimal fullstack starter:

- `client/`: React + Tailwind (Vite)
- `server/`: Go API with PostgreSQL-ready environment wiring
- `docker-compose.yml`: local PostgreSQL service

## Run the server

1. Start PostgreSQL if you want database-backed responses:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && DATABASE_URL=postgres://postgres:postgres@localhost:5432/foxygen_vibe?sslmode=disable go run .`

The API runs now with stdlib only and exposes health/message endpoints immediately. `DATABASE_URL` is wired in so the next backend step can add a PostgreSQL driver and persistence.

## Run the client

1. Install dependencies:
   `cd client && npm install`
2. Start Vite:
   `npm run dev`

The Vite dev server proxies `/api/*` to `http://localhost:8080`.
