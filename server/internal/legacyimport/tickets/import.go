package tickets

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type legacyDump struct {
	Rows []legacyRow `json:"rows"`
}

type legacyRow struct {
	Doc json.RawMessage `json:"doc"`
}

type legacyTimestamp struct {
	Time  *time.Time
	Valid bool
}

type legacyInterval struct {
	Start legacyTimestamp
	End   legacyTimestamp
}

type legacyTicket struct {
	ID             string
	Number         *int
	CreatedAt      *time.Time
	AssignedAt     *time.Time
	WorkStartedAt  *time.Time
	WorkFinishedAt *time.Time
	PlannedStart   *time.Time
	PlannedEnd     *time.Time
	AssignedStart  *time.Time
	AssignedEnd    *time.Time
	ClosedAt       *time.Time
	ClientID       string
	DeviceID       string
	TicketType     string
	AuthorID       string
	DepartmentID   string
	AssignedByID   string
	ReasonID       string
	Description    string
	ContactPerson  string
	ExecutorID     string
	Status         string
	Result         string
}

type ticketImportStats struct {
	Found                int
	Imported             int
	PreservedNumbers     int
	GeneratedNumbers     int
	MissingClient        int
	MissingDevice        int
	MissingAuthor        int
	MissingDepartment    int
	MissingAssignedBy    int
	MissingReason        int
	MissingContactPerson int
	MissingExecutor      int
	MissingTicketType    int
	MissingStatus        int
	MappedFastToInternal int
	MappedFinishedStatus int
	MappedReasonRepair   int
	MappedReasonMaintain int
}

func main() {
	var (
		sourcePath     string
		dryRun         bool
		commandTimeout time.Duration
	)

	flag.StringVar(&sourcePath, "source", "", "Path to the legacy CouchDB _all_docs JSON export")
	flag.BoolVar(&dryRun, "dry-run", false, "Parse and plan the import without writing to PostgreSQL")
	flag.DurationVar(&commandTimeout, "timeout", 10*time.Minute, "Overall import timeout")
	flag.Parse()

	if strings.TrimSpace(sourcePath) == "" {
		log.Fatal("missing required -source")
	}

	items, stats, err := loadLegacyTickets(sourcePath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"discovered %d legacy tickets in %s (preserved_numbers=%d generated_numbers=%d mapped_fast=%d mapped_finished=%d mapped_reasons=%d)",
		len(items),
		sourcePath,
		stats.PreservedNumbers,
		stats.GeneratedNumbers,
		stats.MappedFastToInternal,
		stats.MappedFinishedStatus,
		stats.MappedReasonRepair+stats.MappedReasonMaintain,
	)

	if dryRun {
		for _, item := range items[:min(10, len(items))] {
			log.Printf("dry-run ticket=%s number=%v type=%q status=%q reason=%q", item.ID, item.Number, item.TicketType, item.Status, item.ReasonID)
		}
		log.Printf("dry-run complete")
		return
	}

	databaseURL, err := resolveDatabaseURL(".env")
	if err != nil {
		log.Fatal(err)
	}
	if databaseURL == "" {
		log.Fatal("database is not configured; set DATABASE_URL or DB_* in server/.env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	importStats, err := importTickets(ctx, db, items, stats)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"import complete: found=%d imported=%d preserved_numbers=%d generated_numbers=%d missing_client=%d missing_device=%d missing_author=%d missing_department=%d missing_assigned_by=%d missing_reason=%d missing_contact_person=%d missing_executor=%d missing_ticket_type=%d missing_status=%d",
		importStats.Found,
		importStats.Imported,
		importStats.PreservedNumbers,
		importStats.GeneratedNumbers,
		importStats.MissingClient,
		importStats.MissingDevice,
		importStats.MissingAuthor,
		importStats.MissingDepartment,
		importStats.MissingAssignedBy,
		importStats.MissingReason,
		importStats.MissingContactPerson,
		importStats.MissingExecutor,
		importStats.MissingTicketType,
		importStats.MissingStatus,
	)
}

func loadLegacyTickets(path string) ([]legacyTicket, ticketImportStats, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, ticketImportStats{}, fmt.Errorf("read dump: %w", err)
	}

	var dump legacyDump
	if err := json.Unmarshal(content, &dump); err != nil {
		return nil, ticketImportStats{}, fmt.Errorf("parse dump: %w", err)
	}

	items := make([]legacyTicket, 0)
	stats := ticketImportStats{}
	for _, row := range dump.Rows {
		if len(row.Doc) == 0 || string(row.Doc) == "null" {
			continue
		}

		var meta struct {
			ID string `json:"_id"`
		}
		if err := json.Unmarshal(row.Doc, &meta); err != nil {
			return nil, stats, fmt.Errorf("parse document metadata: %w", err)
		}
		if !strings.HasPrefix(meta.ID, "ticket_") {
			continue
		}

		var doc map[string]json.RawMessage
		if err := json.Unmarshal(row.Doc, &doc); err != nil {
			return nil, stats, fmt.Errorf("parse ticket document %s: %w", meta.ID, err)
		}

		id := trimLegacyPrefix(meta.ID)
		if id == "" {
			continue
		}

		number, hadNumber, err := parseLegacyTicketNumber(doc["number"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse ticket number %s: %w", meta.ID, err)
		}
		if hadNumber {
			stats.PreservedNumbers++
		} else {
			stats.GeneratedNumbers++
		}

		ticketType, mappedFast := normalizeTicketType(parseString(doc["ticketType"]))
		if mappedFast {
			stats.MappedFastToInternal++
		}

		status, mappedFinished := normalizeTicketStatus(parseString(doc["status"]))
		if mappedFinished {
			stats.MappedFinishedStatus++
		}

		reason, reasonMap := normalizeTicketReason(parseString(doc["reason"]))
		switch reasonMap {
		case "repair":
			stats.MappedReasonRepair++
		case "maint":
			stats.MappedReasonMaintain++
		}

		createdAt, err := parseFlexibleTimestamp(doc["createdAt"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse createdAt %s: %w", meta.ID, err)
		}
		assignedAt, err := parseFlexibleTimestamp(doc["assignedAt"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse assignedAt %s: %w", meta.ID, err)
		}
		planned, err := parseInterval(doc["plannedInterval"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse plannedInterval %s: %w", meta.ID, err)
		}
		assigned, err := parseInterval(doc["assignedInterval"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse assignedInterval %s: %w", meta.ID, err)
		}
		actual, err := parseInterval(doc["actualInterval"])
		if err != nil {
			return nil, stats, fmt.Errorf("parse actualInterval %s: %w", meta.ID, err)
		}

		var closedAt *time.Time
		if status == "closed" && actual.End.Valid {
			closedAt = actual.End.Time
		}

		items = append(items, legacyTicket{
			ID:             id,
			Number:         number,
			CreatedAt:      createdAt.Time,
			AssignedAt:     assignedAt.Time,
			WorkStartedAt:  actual.Start.Time,
			WorkFinishedAt: actual.End.Time,
			PlannedStart:   planned.Start.Time,
			PlannedEnd:     planned.End.Time,
			AssignedStart:  assigned.Start.Time,
			AssignedEnd:    assigned.End.Time,
			ClosedAt:       closedAt,
			ClientID:       parseString(doc["client"]),
			DeviceID:       parseString(doc["device"]),
			TicketType:     ticketType,
			AuthorID:       parseString(doc["author"]),
			DepartmentID:   parseString(doc["department"]),
			AssignedByID:   parseString(doc["assignedBy"]),
			ReasonID:       reason,
			Description:    strings.TrimSpace(parseString(doc["description"])),
			ContactPerson:  parseString(doc["contactPerson"]),
			ExecutorID:     parseString(doc["executor"]),
			Status:         status,
			Result:         strings.TrimSpace(parseString(doc["result"])),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Number != nil && items[j].Number != nil && *items[i].Number != *items[j].Number {
			return *items[i].Number < *items[j].Number
		}
		if items[i].Number != nil && items[j].Number == nil {
			return true
		}
		if items[i].Number == nil && items[j].Number != nil {
			return false
		}
		return items[i].ID < items[j].ID
	})

	stats.Found = len(items)
	return items, stats, nil
}

func importTickets(ctx context.Context, db *pgxpool.Pool, items []legacyTicket, stats ticketImportStats) (ticketImportStats, error) {
	if err := ensureTicketsSchema(ctx, db); err != nil {
		return stats, err
	}

	clientIDs, err := loadIDSet(ctx, db, `SELECT id FROM clients`)
	if err != nil {
		return stats, fmt.Errorf("load clients: %w", err)
	}
	deviceIDs, err := loadIDSet(ctx, db, `SELECT id FROM devices`)
	if err != nil {
		return stats, fmt.Errorf("load devices: %w", err)
	}
	accountIDs, err := loadIDSet(ctx, db, `SELECT user_id FROM accounts`)
	if err != nil {
		return stats, fmt.Errorf("load accounts: %w", err)
	}
	externalUserLinks, err := loadOptionalIDMap(ctx, db, `SELECT id, COALESCE(linked_user_id::text, '') FROM external_users`)
	if err != nil {
		return stats, fmt.Errorf("load external users: %w", err)
	}
	departmentIDs, err := loadIDSet(ctx, db, `SELECT id FROM departments`)
	if err != nil {
		return stats, fmt.Errorf("load departments: %w", err)
	}
	reasonIDs, err := loadIDSet(ctx, db, `SELECT id FROM ticket_reasons`)
	if err != nil {
		return stats, fmt.Errorf("load ticket reasons: %w", err)
	}
	contactIDs, err := loadIDSet(ctx, db, `SELECT id FROM contacts`)
	if err != nil {
		return stats, fmt.Errorf("load contacts: %w", err)
	}
	typeIDs, err := loadIDSet(ctx, db, `SELECT type FROM ticket_types`)
	if err != nil {
		return stats, fmt.Errorf("load ticket types: %w", err)
	}
	statusIDs, err := loadIDSet(ctx, db, `SELECT type FROM ticket_statuses`)
	if err != nil {
		return stats, fmt.Errorf("load ticket statuses: %w", err)
	}

	for _, item := range items {
		clientID, ok := resolveOptional(item.ClientID, clientIDs)
		if !ok && item.ClientID != "" {
			stats.MissingClient++
		}
		deviceID, ok := resolveOptional(item.DeviceID, deviceIDs)
		if !ok && item.DeviceID != "" {
			stats.MissingDevice++
		}
		authorID, externalAuthorID, ok := resolveTicketAuthor(item.AuthorID, accountIDs, externalUserLinks)
		if !ok && item.AuthorID != "" {
			stats.MissingAuthor++
		}
		departmentID, ok := resolveOptional(item.DepartmentID, departmentIDs)
		if !ok && item.DepartmentID != "" {
			stats.MissingDepartment++
		}
		assignedByID, ok := resolveOptional(item.AssignedByID, accountIDs)
		if !ok && item.AssignedByID != "" {
			stats.MissingAssignedBy++
		}
		reasonID, ok := resolveOptional(item.ReasonID, reasonIDs)
		if !ok && item.ReasonID != "" {
			stats.MissingReason++
		}
		contactID, ok := resolveOptional(item.ContactPerson, contactIDs)
		if !ok && item.ContactPerson != "" {
			stats.MissingContactPerson++
		}
		executorID, ok := resolveOptional(item.ExecutorID, accountIDs)
		if !ok && item.ExecutorID != "" {
			stats.MissingExecutor++
		}
		ticketType, ok := resolveOptional(item.TicketType, typeIDs)
		if !ok && item.TicketType != "" {
			stats.MissingTicketType++
		}
		status, ok := resolveOptional(item.Status, statusIDs)
		if !ok && item.Status != "" {
			stats.MissingStatus++
		}

		if item.Number != nil {
			if err := upsertTicketWithNumber(
				ctx,
				db,
				item,
				clientID,
				deviceID,
				ticketType,
				authorID,
				externalAuthorID,
				departmentID,
				assignedByID,
				reasonID,
				contactID,
				executorID,
				status,
			); err != nil {
				return stats, err
			}
		} else {
			if err := upsertTicketWithoutNumber(
				ctx,
				db,
				item,
				clientID,
				deviceID,
				ticketType,
				authorID,
				externalAuthorID,
				departmentID,
				assignedByID,
				reasonID,
				contactID,
				executorID,
				status,
			); err != nil {
				return stats, err
			}
		}

		stats.Imported++
	}

	if _, err := db.Exec(
		ctx,
		`SELECT setval(
			pg_get_serial_sequence('tickets', 'number'),
			COALESCE((SELECT MAX(number) FROM tickets), 1),
			true
		)`,
	); err != nil {
		return stats, fmt.Errorf("sync tickets.number sequence: %w", err)
	}

	return stats, nil
}

func upsertTicketWithNumber(
	ctx context.Context,
	db *pgxpool.Pool,
	item legacyTicket,
	clientID any,
	deviceID any,
	ticketType any,
	authorID any,
	externalAuthorID any,
	departmentID any,
	assignedByID any,
	reasonID any,
	contactID any,
	executorID any,
	status any,
) error {
	_, err := db.Exec(
		ctx,
		`INSERT INTO tickets (
			id,
			number,
			created_at,
			assigned_at,
			workstarted_at,
			workfinished_at,
			planned_start,
			planned_end,
			assigned_start,
			assigned_end,
			urgent,
			closed_at,
			client,
			device,
			ticket_type,
			author,
			external_author,
			department,
			assigned_by,
			reason,
			description,
			contact_person,
			executor,
			status,
			result,
			used_materials,
			reference_ticket,
			double_signed
		)
		OVERRIDING SYSTEM VALUE
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			FALSE, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, '{}'::uuid[], NULL, FALSE
		)
		ON CONFLICT (id) DO UPDATE
		SET created_at = EXCLUDED.created_at,
		    assigned_at = EXCLUDED.assigned_at,
		    workstarted_at = EXCLUDED.workstarted_at,
		    workfinished_at = EXCLUDED.workfinished_at,
		    planned_start = EXCLUDED.planned_start,
		    planned_end = EXCLUDED.planned_end,
		    assigned_start = EXCLUDED.assigned_start,
		    assigned_end = EXCLUDED.assigned_end,
		    urgent = EXCLUDED.urgent,
		    closed_at = EXCLUDED.closed_at,
		    client = EXCLUDED.client,
		    device = EXCLUDED.device,
		    ticket_type = EXCLUDED.ticket_type,
		    author = EXCLUDED.author,
		    external_author = EXCLUDED.external_author,
		    department = EXCLUDED.department,
		    assigned_by = EXCLUDED.assigned_by,
		    reason = EXCLUDED.reason,
		    description = EXCLUDED.description,
		    contact_person = EXCLUDED.contact_person,
		    executor = EXCLUDED.executor,
		    status = EXCLUDED.status,
		    result = EXCLUDED.result,
		    used_materials = EXCLUDED.used_materials,
		    reference_ticket = EXCLUDED.reference_ticket,
		    double_signed = EXCLUDED.double_signed`,
		item.ID,
		*item.Number,
		item.CreatedAt,
		item.AssignedAt,
		item.WorkStartedAt,
		item.WorkFinishedAt,
		item.PlannedStart,
		item.PlannedEnd,
		item.AssignedStart,
		item.AssignedEnd,
		item.ClosedAt,
		clientID,
		deviceID,
		ticketType,
		authorID,
		externalAuthorID,
		departmentID,
		assignedByID,
		reasonID,
		item.Description,
		contactID,
		executorID,
		status,
		item.Result,
	)
	if err != nil {
		return fmt.Errorf("import ticket %s with number: %w", item.ID, err)
	}

	return nil
}

func upsertTicketWithoutNumber(
	ctx context.Context,
	db *pgxpool.Pool,
	item legacyTicket,
	clientID any,
	deviceID any,
	ticketType any,
	authorID any,
	externalAuthorID any,
	departmentID any,
	assignedByID any,
	reasonID any,
	contactID any,
	executorID any,
	status any,
) error {
	_, err := db.Exec(
		ctx,
		`INSERT INTO tickets (
			id,
			created_at,
			assigned_at,
			workstarted_at,
			workfinished_at,
			planned_start,
			planned_end,
			assigned_start,
			assigned_end,
			urgent,
			closed_at,
			client,
			device,
			ticket_type,
			author,
			external_author,
			department,
			assigned_by,
			reason,
			description,
			contact_person,
			executor,
			status,
			result,
			used_materials,
			reference_ticket,
			double_signed
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			FALSE, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23, '{}'::uuid[], NULL, FALSE
		)
		ON CONFLICT (id) DO UPDATE
		SET created_at = EXCLUDED.created_at,
		    assigned_at = EXCLUDED.assigned_at,
		    workstarted_at = EXCLUDED.workstarted_at,
		    workfinished_at = EXCLUDED.workfinished_at,
		    planned_start = EXCLUDED.planned_start,
		    planned_end = EXCLUDED.planned_end,
		    assigned_start = EXCLUDED.assigned_start,
		    assigned_end = EXCLUDED.assigned_end,
		    urgent = EXCLUDED.urgent,
		    closed_at = EXCLUDED.closed_at,
		    client = EXCLUDED.client,
		    device = EXCLUDED.device,
		    ticket_type = EXCLUDED.ticket_type,
		    author = EXCLUDED.author,
		    external_author = EXCLUDED.external_author,
		    department = EXCLUDED.department,
		    assigned_by = EXCLUDED.assigned_by,
		    reason = EXCLUDED.reason,
		    description = EXCLUDED.description,
		    contact_person = EXCLUDED.contact_person,
		    executor = EXCLUDED.executor,
		    status = EXCLUDED.status,
		    result = EXCLUDED.result,
		    used_materials = EXCLUDED.used_materials,
		    reference_ticket = EXCLUDED.reference_ticket,
		    double_signed = EXCLUDED.double_signed`,
		item.ID,
		item.CreatedAt,
		item.AssignedAt,
		item.WorkStartedAt,
		item.WorkFinishedAt,
		item.PlannedStart,
		item.PlannedEnd,
		item.AssignedStart,
		item.AssignedEnd,
		item.ClosedAt,
		clientID,
		deviceID,
		ticketType,
		authorID,
		externalAuthorID,
		departmentID,
		assignedByID,
		reasonID,
		item.Description,
		contactID,
		executorID,
		status,
		item.Result,
	)
	if err != nil {
		return fmt.Errorf("import ticket %s without number: %w", item.ID, err)
	}

	return nil
}

func ensureTicketsSchema(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS external_users (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			linked_user_id UUID REFERENCES accounts(user_id) ON DELETE SET NULL
		)`,
	); err != nil {
		return fmt.Errorf("ensure external_users table for tickets: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS tickets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			number INT GENERATED ALWAYS AS IDENTITY,
			created_at TIMESTAMP DEFAULT (NOW() AT TIME ZONE 'UTC'),
			assigned_at TIMESTAMP DEFAULT NULL,
			workstarted_at TIMESTAMP DEFAULT NULL,
			workfinished_at TIMESTAMP DEFAULT NULL,
			planned_start TIMESTAMP DEFAULT NULL,
			planned_end TIMESTAMP DEFAULT NULL,
			assigned_start TIMESTAMP DEFAULT NULL,
			assigned_end TIMESTAMP DEFAULT NULL,
			urgent BOOLEAN NOT NULL DEFAULT FALSE,
			closed_at TIMESTAMP DEFAULT NULL,
			client UUID REFERENCES clients(id) ON DELETE SET NULL,
			device UUID REFERENCES devices(id) ON DELETE SET NULL,
			ticket_type VARCHAR(128) REFERENCES ticket_types(type) ON DELETE SET NULL,
			author UUID REFERENCES accounts(user_id) ON DELETE SET NULL,
			external_author UUID REFERENCES external_users(id) ON DELETE SET NULL,
			department UUID REFERENCES departments(id) ON DELETE SET NULL,
			assigned_by UUID REFERENCES accounts(user_id) ON DELETE SET NULL DEFAULT NULL,
			reason VARCHAR(128) REFERENCES ticket_reasons(id) ON DELETE SET NULL,
			description TEXT NOT NULL DEFAULT '',
			contact_person UUID REFERENCES contacts(id) ON DELETE SET NULL,
			executor UUID REFERENCES accounts(user_id) ON DELETE SET NULL,
			status VARCHAR(128) REFERENCES ticket_statuses(type) ON DELETE SET NULL,
			result TEXT NOT NULL DEFAULT '',
			used_materials UUID[] NOT NULL DEFAULT '{}',
			reference_ticket UUID REFERENCES tickets(id) DEFAULT NULL,
			double_signed BOOLEAN NOT NULL DEFAULT FALSE
		)`,
	); err != nil {
		return fmt.Errorf("ensure tickets table: %w", err)
	}

	statements := []struct {
		query string
		label string
	}{
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS number INT GENERATED ALWAYS AS IDENTITY`, "number"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT (NOW() AT TIME ZONE 'UTC')`, "created_at"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS assigned_at TIMESTAMP DEFAULT NULL`, "assigned_at"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS workstarted_at TIMESTAMP DEFAULT NULL`, "workstarted_at"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS workfinished_at TIMESTAMP DEFAULT NULL`, "workfinished_at"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS planned_start TIMESTAMP DEFAULT NULL`, "planned_start"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS planned_end TIMESTAMP DEFAULT NULL`, "planned_end"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS assigned_start TIMESTAMP DEFAULT NULL`, "assigned_start"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS assigned_end TIMESTAMP DEFAULT NULL`, "assigned_end"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS urgent BOOLEAN NOT NULL DEFAULT FALSE`, "urgent"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP DEFAULT NULL`, "closed_at"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS client UUID REFERENCES clients(id) ON DELETE SET NULL`, "client"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS device UUID REFERENCES devices(id) ON DELETE SET NULL`, "device"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS ticket_type VARCHAR(128) REFERENCES ticket_types(type) ON DELETE SET NULL`, "ticket_type"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS author UUID REFERENCES accounts(user_id) ON DELETE SET NULL`, "author"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS external_author UUID REFERENCES external_users(id) ON DELETE SET NULL`, "external_author"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS department UUID REFERENCES departments(id) ON DELETE SET NULL`, "department"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS assigned_by UUID REFERENCES accounts(user_id) ON DELETE SET NULL DEFAULT NULL`, "assigned_by"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS reason VARCHAR(128) REFERENCES ticket_reasons(id) ON DELETE SET NULL`, "reason"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT ''`, "description"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS contact_person UUID REFERENCES contacts(id) ON DELETE SET NULL`, "contact_person"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS executor UUID REFERENCES accounts(user_id) ON DELETE SET NULL`, "executor"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS status VARCHAR(128) REFERENCES ticket_statuses(type) ON DELETE SET NULL`, "status"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS result TEXT NOT NULL DEFAULT ''`, "result"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS used_materials UUID[] NOT NULL DEFAULT '{}'`, "used_materials"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS reference_ticket UUID REFERENCES tickets(id) DEFAULT NULL`, "reference_ticket"},
		{`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS double_signed BOOLEAN NOT NULL DEFAULT FALSE`, "double_signed"},
	}

	for _, stmt := range statements {
		if _, err := db.Exec(ctx, stmt.query); err != nil {
			return fmt.Errorf("ensure tickets.%s column: %w", stmt.label, err)
		}
	}

	return nil
}

func loadIDSet(ctx context.Context, db *pgxpool.Pool, query string) (map[string]bool, error) {
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func loadOptionalIDMap(ctx context.Context, db *pgxpool.Pool, query string) (map[string]string, error) {
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make(map[string]string)
	for rows.Next() {
		var id string
		var linked string
		if err := rows.Scan(&id, &linked); err != nil {
			return nil, err
		}
		values[id] = linked
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func resolveOptional(value string, allowed map[string]bool) (any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	if allowed[value] {
		return value, true
	}
	return nil, false
}

func resolveTicketAuthor(value string, accountIDs map[string]bool, externalUserLinks map[string]string) (any, any, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil, true
	}
	if accountIDs[value] {
		return value, nil, true
	}

	linkedUserID, ok := externalUserLinks[value]
	if !ok {
		return nil, nil, false
	}

	var authorID any
	if accountIDs[linkedUserID] {
		authorID = linkedUserID
	}

	return authorID, value, true
}

func parseLegacyTicketNumber(raw json.RawMessage) (*int, bool, error) {
	value := strings.TrimSpace(parseString(raw))
	if value == "" {
		return nil, false, nil
	}

	if idx := strings.IndexFunc(value, func(r rune) bool {
		return r < '0' || r > '9'
	}); idx > 0 {
		value = value[:idx]
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return nil, false, err
	}

	return &number, true, nil
}

func parseString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}

	return ""
}

func parseFlexibleTimestamp(raw json.RawMessage) (legacyTimestamp, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return legacyTimestamp{}, nil
	}

	var unixMillis int64
	if err := json.Unmarshal(raw, &unixMillis); err == nil {
		t := time.UnixMilli(unixMillis).UTC()
		return legacyTimestamp{Time: &t, Valid: true}, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		value = strings.TrimSpace(value)
		if value == "" {
			return legacyTimestamp{}, nil
		}

		layouts := []string{
			"2006-01-02T15:04:05",
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, value); err == nil {
				utc := t.UTC()
				return legacyTimestamp{Time: &utc, Valid: true}, nil
			}
		}

		return legacyTimestamp{}, fmt.Errorf("unsupported time string %q", value)
	}

	return legacyTimestamp{}, fmt.Errorf("unsupported timestamp payload %s", string(raw))
}

func parseInterval(raw json.RawMessage) (legacyInterval, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return legacyInterval{}, nil
	}

	var decoded struct {
		Start json.RawMessage `json:"start"`
		End   json.RawMessage `json:"end"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return legacyInterval{}, err
	}

	start, err := parseFlexibleTimestamp(decoded.Start)
	if err != nil {
		return legacyInterval{}, err
	}
	end, err := parseFlexibleTimestamp(decoded.End)
	if err != nil {
		return legacyInterval{}, err
	}

	return legacyInterval{Start: start, End: end}, nil
}

func normalizeTicketType(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "fast":
		return "internal", true
	case "internal", "external":
		return strings.TrimSpace(value), false
	default:
		return "", false
	}
}

func normalizeTicketStatus(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "finished":
		return "worksDone", true
	case "assigned", "cancelled", "closed", "created", "inWork", "worksDone":
		return strings.TrimSpace(value), false
	default:
		return "", false
	}
}

func normalizeTicketReason(value string) (string, string) {
	switch strings.TrimSpace(value) {
	case "repairs":
		return "repair", "repair"
	case "maintenance", "mainenance":
		return "maintanence", "maint"
	default:
		return strings.TrimSpace(value), ""
	}
}

func trimLegacyPrefix(value string) string {
	parts := strings.Split(value, "_")
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func resolveDatabaseURL(dotEnvPath string) (string, error) {
	fileEnv, err := loadDotEnv(dotEnvPath)
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
