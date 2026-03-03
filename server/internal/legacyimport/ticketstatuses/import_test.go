package ticketstatuses

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyTicketStatuses(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "ticket_1_a",
	        "status": "finished"
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyTicketStatuses(path)
	if err != nil {
		t.Fatalf("loadLegacyTicketStatuses: %v", err)
	}

	if len(items) != 6 {
		t.Fatalf("expected 6 statuses, got %d", len(items))
	}

	if items[0].Type != "assigned" || items[0].Title != "назначен" {
		t.Fatalf("unexpected first status %#v", items[0])
	}

	if items[len(items)-1].Type != "worksDone" || items[len(items)-1].Title != "работы завершены" {
		t.Fatalf("unexpected last status %#v", items[len(items)-1])
	}
}
