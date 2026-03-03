package tickets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyTickets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "ticket_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "createdAt": "2024-01-02T03:04:05",
	        "number": "",
	        "client": "client-2",
	        "device": "device-2",
	        "ticketType": "fast",
	        "author": "user-2",
	        "plannedInterval": {"start": "2024-01-03T00:00:00", "end": "2024-01-04T00:00:00"},
	        "department": "dept-2",
	        "assignedBy": "user-3",
	        "assignedAt": 1704251045000,
	        "reason": "maintenance",
	        "description": " beta desc ",
	        "contactPerson": "contact-2",
	        "executor": "user-4",
	        "assignedInterval": {"start": "2024-01-05T00:00:00", "end": "2024-01-06T00:00:00"},
	        "status": "finished",
	        "actualInterval": {"start": 1704423845000, "end": 1704510245000},
	        "result": " done "
	      }
	    },
	    {
	      "doc": {
	        "_id": "ticket_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "createdAt": 1704164645000,
	        "number": "000000123",
	        "client": "client-1",
	        "device": "device-1",
	        "ticketType": "external",
	        "author": "user-1",
	        "plannedInterval": {"start": null, "end": null},
	        "department": "dept-1",
	        "reason": "repairs",
	        "description": "alpha",
	        "contactPerson": null,
	        "executor": null,
	        "status": "closed",
	        "actualInterval": {"start": 1704164645000, "end": 1704168245000},
	        "result": ""
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, stats, err := loadLegacyTickets(path)
	if err != nil {
		t.Fatalf("loadLegacyTickets: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(items))
	}

	if items[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected numbered ticket to sort first, got %q", items[0].ID)
	}

	if items[0].Number == nil || *items[0].Number != 123 {
		t.Fatalf("expected parsed ticket number, got %#v", items[0].Number)
	}

	if items[0].ReasonID != "repair" {
		t.Fatalf("expected repairs to map to repair, got %q", items[0].ReasonID)
	}

	if items[0].Status != "closed" || items[0].ClosedAt == nil {
		t.Fatalf("expected closed ticket to derive closed_at, got status=%q closed_at=%v", items[0].Status, items[0].ClosedAt)
	}

	if items[1].Number != nil {
		t.Fatalf("expected blank number to stay nil, got %#v", items[1].Number)
	}

	if items[1].TicketType != "internal" {
		t.Fatalf("expected fast to map to internal, got %q", items[1].TicketType)
	}

	if items[1].Status != "worksDone" {
		t.Fatalf("expected finished to map to worksDone, got %q", items[1].Status)
	}

	if items[1].ReasonID != "maintanence" {
		t.Fatalf("expected maintenance to map to maintanence, got %q", items[1].ReasonID)
	}

	if items[1].Description != "beta desc" || items[1].Result != "done" {
		t.Fatalf("expected text trimmed, got desc=%q result=%q", items[1].Description, items[1].Result)
	}

	if stats.PreservedNumbers != 1 || stats.GeneratedNumbers != 1 || stats.MappedFastToInternal != 1 || stats.MappedFinishedStatus != 1 {
		t.Fatalf("unexpected stats %#v", stats)
	}
}
