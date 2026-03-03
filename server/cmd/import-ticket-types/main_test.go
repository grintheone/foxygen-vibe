package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyTicketTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "ticket_1_a",
	        "ticketType": "fast"
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyTicketTypes(path)
	if err != nil {
		t.Fatalf("loadLegacyTicketTypes: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 types, got %d", len(items))
	}

	if items[0].Type != "external" || items[0].Title != "внешний" {
		t.Fatalf("unexpected first type %#v", items[0])
	}

	if items[1].Type != "internal" || items[1].Title != "внутренний" {
		t.Fatalf("unexpected types %#v", items)
	}
}
