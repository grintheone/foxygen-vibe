package pouchsync

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type lookupDoc struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
}

type ticketReasonDoc struct {
	ID      string `json:"_id"`
	Title   string `json:"title"`
	Past    string `json:"past"`
	Present string `json:"present"`
	Future  string `json:"future"`
}

type clientDoc struct {
	ID               string          `json:"_id"`
	Title            string          `json:"title"`
	Address          string          `json:"address"`
	Region           string          `json:"region"`
	Location         json.RawMessage `json:"location"`
	LaboratorySystem *string         `json:"laboratorySystem"`
}

type contactDoc struct {
	ID         string `json:"_id"`
	Ref        string `json:"ref"`
	FirstName  string `json:"firstName"`
	MiddleName string `json:"middleName"`
	LastName   string `json:"lastName"`
	Position   string `json:"position"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
}

type classificatorDoc struct {
	ID                      string          `json:"_id"`
	Title                   string          `json:"title"`
	Manufacturer            json.RawMessage `json:"manufacturer"`
	ResearchType            string          `json:"researchType"`
	RegistrationCertificate json.RawMessage `json:"registrationCertificate"`
	MaintenanceRegulations  json.RawMessage `json:"maintenanceRegulations"`
	Attachments             []string        `json:"attachments"`
	Images                  []string        `json:"images"`
}

type deviceDoc struct {
	ID             string          `json:"_id"`
	Classificator  string          `json:"classificator"`
	SerialNumber   string          `json:"serialNumber"`
	Properties     json.RawMessage `json:"properties"`
	ConnectedToLis bool            `json:"connectedToLis"`
	IsUsed         *bool           `json:"isUsed"`
}

type ticketTimestamp struct {
	Time  *time.Time
	Valid bool
}

type ticketInterval struct {
	Start ticketTimestamp
	End   ticketTimestamp
}

type ticketRecord struct {
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
	Urgent         bool
	DoubleSigned   bool
}

func applyRegion(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM regions WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc lookupDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id, title := trimLegacyPrefix(doc.ID), normalizeWhitespace(doc.Title)
	if id == "" || title == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO regions (id, title)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title`,
		id,
		title,
	)
	return err
}

func applyManufacturer(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM manufacturers WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc lookupDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id, title := trimLegacyPrefix(doc.ID), normalizeWhitespace(doc.Title)
	if id == "" || title == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO manufacturers (id, title)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title`,
		id,
		title,
	)
	return err
}

func applyResearchType(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM research_type WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc lookupDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id, title := trimLegacyPrefix(doc.ID), normalizeWhitespace(doc.Title)
	if id == "" || title == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO research_type (id, title)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title`,
		id,
		title,
	)
	return err
}

func applyTicketReason(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM ticket_reasons WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc ticketReasonDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id := trimLegacyPrefix(doc.ID)
	if id == "" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO ticket_reasons (id, title, past, present, future)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
		    past = EXCLUDED.past,
		    present = EXCLUDED.present,
		    future = EXCLUDED.future`,
		id,
		normalizeWhitespace(doc.Title),
		normalizeWhitespace(doc.Past),
		normalizeWhitespace(doc.Present),
		normalizeWhitespace(doc.Future),
	)
	return err
}

func applyClient(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM clients WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc clientDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id, title := trimLegacyPrefix(doc.ID), normalizeWhitespace(doc.Title)
	if id == "" || title == "" {
		return nil
	}

	var region any
	regionID := strings.TrimSpace(doc.Region)
	if regionID != "" && existsByID(ctx, tx, "regions", regionID) {
		region = regionID
	}

	var location any
	if len(doc.Location) > 0 && string(doc.Location) != "null" {
		location = []byte(doc.Location)
	}

	var laboratorySystem any
	if value := normalizeNullableString(doc.LaboratorySystem); value != nil {
		laboratorySystem = *value
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO clients (id, title, region, address, location, laboratory_system)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
		    region = EXCLUDED.region,
		    address = EXCLUDED.address,
		    location = EXCLUDED.location,
		    laboratory_system = EXCLUDED.laboratory_system`,
		id,
		title,
		region,
		strings.TrimSpace(doc.Address),
		location,
		laboratorySystem,
	)
	return err
}

func applyContact(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM contacts WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc contactDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id := trimLegacyPrefix(doc.ID)
	if id == "" {
		return nil
	}

	var clientID any
	ref := strings.TrimSpace(doc.Ref)
	if ref != "" && existsByID(ctx, tx, "clients", ref) {
		clientID = ref
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO contacts (id, name, position, phone, email, client_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    position = EXCLUDED.position,
		    phone = EXCLUDED.phone,
		    email = EXCLUDED.email,
		    client_id = EXCLUDED.client_id`,
		id,
		buildContactName(doc.FirstName, doc.MiddleName, doc.LastName),
		normalizeWhitespace(doc.Position),
		strings.TrimSpace(doc.Phone),
		strings.TrimSpace(strings.ToLower(doc.Email)),
		clientID,
	)
	return err
}

func applyClassificator(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM classificators WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc classificatorDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id, title := trimLegacyPrefix(doc.ID), normalizeWhitespace(doc.Title)
	if id == "" || title == "" {
		return nil
	}

	manufacturerID, err := parseManufacturerID(doc.Manufacturer)
	if err != nil {
		return fmt.Errorf("parse manufacturer: %w", err)
	}

	var manufacturer any
	if manufacturerID != "" && existsByID(ctx, tx, "manufacturers", manufacturerID) {
		manufacturer = manufacturerID
	}

	var researchType any
	researchTypeID := strings.TrimSpace(doc.ResearchType)
	if researchTypeID != "" && existsByID(ctx, tx, "research_type", researchTypeID) {
		researchType = researchTypeID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO classificators (
			id,
			title,
			manufacturer,
			research_type,
			registration_certificate,
			maintenance_regulations,
			attachments,
			images
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET title = EXCLUDED.title,
		    manufacturer = EXCLUDED.manufacturer,
		    research_type = EXCLUDED.research_type,
		    registration_certificate = EXCLUDED.registration_certificate,
		    maintenance_regulations = EXCLUDED.maintenance_regulations,
		    attachments = EXCLUDED.attachments,
		    images = EXCLUDED.images`,
		id,
		title,
		manufacturer,
		researchType,
		[]byte(normalizeJSON(doc.RegistrationCertificate, []byte(`{}`))),
		[]byte(normalizeJSON(doc.MaintenanceRegulations, []byte(`[]`))),
		cloneStrings(doc.Attachments),
		cloneStrings(doc.Images),
	)
	return err
}

func applyDevice(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM devices WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	var doc deviceDoc
	if err := decodeDoc(change, &doc); err != nil {
		return err
	}
	id := trimLegacyPrefix(doc.ID)
	if id == "" {
		return nil
	}

	var classificator any
	classificatorID := strings.TrimSpace(doc.Classificator)
	if classificatorID != "" && existsByID(ctx, tx, "classificators", classificatorID) {
		classificator = classificatorID
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO devices (id, classificator, serial_number, properties, connected_to_lis, is_used)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET classificator = EXCLUDED.classificator,
		    serial_number = EXCLUDED.serial_number,
		    properties = EXCLUDED.properties,
		    connected_to_lis = EXCLUDED.connected_to_lis,
		    is_used = EXCLUDED.is_used`,
		id,
		classificator,
		strings.TrimSpace(doc.SerialNumber),
		[]byte(normalizeJSON(doc.Properties, []byte(`{}`))),
		doc.ConnectedToLis,
		doc.IsUsed != nil && *doc.IsUsed,
	)
	return err
}

func applyTicket(ctx context.Context, tx pgx.Tx, change changeEvent) error {
	if change.Deleted {
		_, err := tx.Exec(ctx, `DELETE FROM tickets WHERE id = $1`, trimLegacyPrefix(change.ID))
		return err
	}

	item, err := parseTicketRecord(change)
	if err != nil {
		return err
	}
	if item.ID == "" {
		return nil
	}

	clientID := resolveOptionalID(ctx, tx, "clients", item.ClientID)
	deviceID := resolveOptionalID(ctx, tx, "devices", item.DeviceID)
	authorID, externalAuthorID := resolveTicketAuthor(ctx, tx, item.AuthorID)
	departmentID := resolveOptionalID(ctx, tx, "departments", item.DepartmentID)
	assignedByID := resolveOptionalID(ctx, tx, "accounts", item.AssignedByID, "user_id")
	reasonID := resolveOptionalID(ctx, tx, "ticket_reasons", item.ReasonID)
	contactID := resolveOptionalID(ctx, tx, "contacts", item.ContactPerson)
	executorID := resolveOptionalID(ctx, tx, "accounts", item.ExecutorID, "user_id")
	ticketType := resolveOptionalID(ctx, tx, "ticket_types", item.TicketType, "type")
	status := resolveOptionalID(ctx, tx, "ticket_statuses", item.Status, "type")

	if item.Number != nil {
		if err := upsertTicketWithNumber(
			ctx,
			tx,
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
			return err
		}
		return syncTicketNumberSequence(ctx, tx)
	}

	return upsertTicketWithoutNumber(
		ctx,
		tx,
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
	)
}

func parseTicketRecord(change changeEvent) (ticketRecord, error) {
	if len(change.Doc) == 0 || string(change.Doc) == "null" {
		return ticketRecord{}, fmt.Errorf("change %s does not include a document", change.ID)
	}

	var doc map[string]json.RawMessage
	if err := json.Unmarshal(change.Doc, &doc); err != nil {
		return ticketRecord{}, fmt.Errorf("parse ticket document %s: %w", change.ID, err)
	}

	id := trimLegacyPrefix(change.ID)
	if id == "" {
		return ticketRecord{}, nil
	}

	number, _, err := parseTicketNumber(doc["number"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse ticket number %s: %w", change.ID, err)
	}

	ticketType, _ := normalizeTicketType(parseString(doc["ticketType"]))
	executorID := parseString(doc["executor"])
	status, _ := normalizeTicketStatus(parseString(doc["status"]), executorID)
	reason, _ := normalizeTicketReason(parseString(doc["reason"]))

	createdAt, err := parseFlexibleTimestamp(doc["createdAt"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse createdAt %s: %w", change.ID, err)
	}
	assignedAt, err := parseFlexibleTimestamp(doc["assignedAt"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse assignedAt %s: %w", change.ID, err)
	}
	planned, err := parseInterval(doc["plannedInterval"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse plannedInterval %s: %w", change.ID, err)
	}
	assigned, err := parseInterval(doc["assignedInterval"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse assignedInterval %s: %w", change.ID, err)
	}
	actual, err := parseInterval(doc["actualInterval"])
	if err != nil {
		return ticketRecord{}, fmt.Errorf("parse actualInterval %s: %w", change.ID, err)
	}

	var closedAt *time.Time
	if status == "closed" && actual.End.Valid {
		closedAt = actual.End.Time
	}

	return ticketRecord{
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
		ExecutorID:     executorID,
		Status:         status,
		Result:         strings.TrimSpace(parseString(doc["result"])),
		Urgent:         parseBool(doc["urgent"]),
		DoubleSigned:   parseBool(doc["doubleSigned"]),
	}, nil
}

const upsertTicketWithNumberSQL = `INSERT INTO tickets (
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
	$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
	$21, $22, $23, $24, $25, '{}'::uuid[], NULL, $26
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
    double_signed = EXCLUDED.double_signed`

func upsertTicketWithNumber(
	ctx context.Context,
	tx pgx.Tx,
	item ticketRecord,
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
	_, err := tx.Exec(
		ctx,
		upsertTicketWithNumberSQL,
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
		item.Urgent,
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
		item.DoubleSigned,
	)
	if err != nil {
		return fmt.Errorf("upsert ticket %s with number: %w", item.ID, err)
	}
	return nil
}

func upsertTicketWithoutNumber(
	ctx context.Context,
	tx pgx.Tx,
	item ticketRecord,
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
	_, err := tx.Exec(
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
			$10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, '{}'::uuid[], NULL, $25
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
		item.Urgent,
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
		item.DoubleSigned,
	)
	if err != nil {
		return fmt.Errorf("upsert ticket %s without number: %w", item.ID, err)
	}
	return nil
}

func decodeDoc(change changeEvent, target any) error {
	if len(change.Doc) == 0 || string(change.Doc) == "null" {
		return fmt.Errorf("change %s does not include a document", change.ID)
	}
	if err := json.Unmarshal(change.Doc, target); err != nil {
		return fmt.Errorf("parse document %s: %w", change.ID, err)
	}
	return nil
}

func existsByID(ctx context.Context, tx pgx.Tx, table string, id string) bool {
	if columnRequiresUUID(table, "id") {
		normalized, ok := normalizeUUID(id)
		if !ok {
			return false
		}
		id = normalized
	}

	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE id = $1)`, table)
	if err := tx.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func resolveOptionalID(ctx context.Context, tx pgx.Tx, table string, value string, columns ...string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	column := "id"
	if len(columns) > 0 && strings.TrimSpace(columns[0]) != "" {
		column = strings.TrimSpace(columns[0])
	}
	if columnRequiresUUID(table, column) {
		normalized, ok := normalizeUUID(value)
		if !ok {
			return nil
		}
		value = normalized
	}
	if existsByColumn(ctx, tx, table, column, value) {
		return value
	}
	return nil
}

func resolveTicketAuthor(ctx context.Context, tx pgx.Tx, value string) (any, any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	normalized, ok := normalizeUUID(value)
	if !ok {
		return nil, nil
	}
	value = normalized
	if existsByColumn(ctx, tx, "accounts", "user_id", value) {
		return value, nil
	}

	var linkedUserID string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(linked_user_id::text, '') FROM external_users WHERE id = $1`, value).Scan(&linkedUserID); err != nil {
		return nil, nil
	}

	var authorID any
	if existsByColumn(ctx, tx, "accounts", "user_id", linkedUserID) {
		authorID = linkedUserID
	}

	return authorID, value
}

func existsByColumn(ctx context.Context, tx pgx.Tx, table string, column string, value string) bool {
	if columnRequiresUUID(table, column) {
		normalized, ok := normalizeUUID(value)
		if !ok {
			return false
		}
		value = normalized
	}

	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE %s = $1)`, table, column)
	if err := tx.QueryRow(ctx, query, value).Scan(&exists); err != nil {
		return false
	}
	return exists
}

var uuidColumns = map[string]map[string]struct{}{
	"accounts":       {"user_id": {}},
	"classificators": {"id": {}},
	"clients":        {"id": {}},
	"contacts":       {"id": {}},
	"departments":    {"id": {}},
	"devices":        {"id": {}},
	"external_users": {"id": {}, "linked_user_id": {}},
	"manufacturers":  {"id": {}},
	"regions":        {"id": {}},
	"research_type":  {"id": {}},
}

func columnRequiresUUID(table string, column string) bool {
	columns, ok := uuidColumns[table]
	if !ok {
		return false
	}
	_, ok = columns[column]
	return ok
}

func normalizeUUID(value string) (string, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}

func syncTicketNumberSequence(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		SELECT setval(
			pg_get_serial_sequence('tickets', 'number'),
			COALESCE((SELECT MAX(number) FROM tickets), 1),
			true
		)`,
	)
	if err != nil {
		return fmt.Errorf("sync tickets.number sequence: %w", err)
	}
	return nil
}

func normalizeNullableString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func buildContactName(firstName, middleName, lastName string) string {
	return normalizeWhitespace(strings.Join([]string{lastName, firstName, middleName}, " "))
}

func normalizeJSON(raw json.RawMessage, fallback []byte) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return append(json.RawMessage(nil), fallback...)
	}
	return append(json.RawMessage(nil), raw...)
}

func parseManufacturerID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value), nil
	}

	var object struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		return strings.TrimSpace(object.ID), nil
	}

	return "", fmt.Errorf("unsupported manufacturer value %s", string(raw))
}

func parseTicketNumber(raw json.RawMessage) (*int, bool, error) {
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

func parseBool(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}

	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		parsed, err := strconv.ParseBool(strings.TrimSpace(text))
		return err == nil && parsed
	}

	return false
}

func parseFlexibleTimestamp(raw json.RawMessage) (ticketTimestamp, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return ticketTimestamp{}, nil
	}

	var unixMillis int64
	if err := json.Unmarshal(raw, &unixMillis); err == nil {
		t := time.UnixMilli(unixMillis).UTC()
		return ticketTimestamp{Time: &t, Valid: true}, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		value = strings.TrimSpace(value)
		if value == "" {
			return ticketTimestamp{}, nil
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
				return ticketTimestamp{Time: &utc, Valid: true}, nil
			}
		}

		return ticketTimestamp{}, fmt.Errorf("unsupported time string %q", value)
	}

	return ticketTimestamp{}, fmt.Errorf("unsupported timestamp payload %s", string(raw))
}

func parseInterval(raw json.RawMessage) (ticketInterval, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return ticketInterval{}, nil
	}

	var decoded struct {
		Start json.RawMessage `json:"start"`
		End   json.RawMessage `json:"end"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ticketInterval{}, err
	}

	start, err := parseFlexibleTimestamp(decoded.Start)
	if err != nil {
		return ticketInterval{}, err
	}
	end, err := parseFlexibleTimestamp(decoded.End)
	if err != nil {
		return ticketInterval{}, err
	}

	return ticketInterval{Start: start, End: end}, nil
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

func normalizeTicketStatus(value string, executorID string) (string, bool) {
	status := strings.TrimSpace(value)
	if status == "assigned" && strings.TrimSpace(executorID) == "" {
		return "cancelled", false
	}

	switch status {
	case "finished":
		return "worksDone", true
	case "assigned", "cancelled", "closed", "created", "inWork", "worksDone":
		return status, false
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

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func trimLegacyPrefix(value string) string {
	parts := strings.Split(value, "_")
	if len(parts) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(parts[len(parts)-1])
}
