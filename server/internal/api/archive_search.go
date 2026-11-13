package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Each entity has its own limit so a common equipment name cannot hide clients.
const archiveSearchSQL = `
WITH clients_found AS (
 SELECT 'client' AS kind, c.id::text AS id, c.title AS title,
        concat_ws(' · ', NULLIF(c.address, ''), NULLIF(r.title, '')) AS subtitle
 FROM clients c LEFT JOIN regions r ON r.id = c.region
 WHERE NOT EXISTS (
   SELECT 1 FROM unnest($1::text[]) term
   WHERE strpos(lower(concat_ws(' ', c.title, c.address, r.title)), term) = 0
 )
 ORDER BY lower(c.title), c.id LIMIT 21
), devices_found AS (
 SELECT 'device' AS kind, d.id::text AS id,
        COALESCE(NULLIF(cls.title, ''), 'Оборудование') AS title,
        concat_ws(' · ', NULLIF(d.serial_number, ''), NULLIF(m.title, '')) AS subtitle
 FROM devices d
 LEFT JOIN classificators cls ON cls.id = d.classificator
 LEFT JOIN manufacturers m ON m.id = cls.manufacturer
 WHERE NOT EXISTS (
   SELECT 1 FROM unnest($1::text[]) term
   WHERE strpos(lower(concat_ws(' ', cls.title, d.serial_number, m.title, d.properties::text)), term) = 0
 )
 ORDER BY lower(cls.title), d.serial_number, d.id LIMIT 21
)
SELECT * FROM clients_found UNION ALL SELECT * FROM devices_found
`

type archiveSearchItem struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

func (s *Server) handleArchiveSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, err := parseAuthorizationHeader(s.auth.jwtSecret, r.Header.Get("Authorization")); err != nil {
		http.Error(w, "invalid access token", http.StatusUnauthorized)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) < 2 || utf8.RuneCountInString(query) > 100 {
		http.Error(w, "q must contain between 2 and 100 characters", http.StatusBadRequest)
		return
	}
	if s.db == nil {
		http.Error(w, "database not configured", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows, err := s.db.Query(ctx, archiveSearchSQL, strings.Fields(strings.ToLower(query)))
	if err != nil {
		log.Printf("archive search failed: %v", err)
		http.Error(w, "failed to search archives", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	items := make([]archiveSearchItem, 0)
	for rows.Next() {
		var item archiveSearchItem
		if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Subtitle); err != nil {
			log.Printf("scan archive search failed: %v", err)
			http.Error(w, "failed to search archives", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("iterate archive search failed: %v", err)
		http.Error(w, "failed to search archives", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
