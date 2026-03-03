package ticketreasons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyTicketReasons(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "ticketReason_1_beta",
	        "title": " Beta  Title ",
	        "past": " did  beta ",
	        "present": " doing beta ",
	        "future": " do beta "
	      }
	    },
	    {
	      "doc": {
	        "_id": "ticket_1_ignore-me",
	        "status": "closed"
	      }
	    },
	    {
	      "doc": {
	        "_id": "ticketReason_1_alpha",
	        "title": "Alpha",
	        "past": "",
	        "present": "",
	        "future": ""
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyTicketReasons(path)
	if err != nil {
		t.Fatalf("loadLegacyTicketReasons: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 reasons, got %d", len(items))
	}

	if items[0].ID != "alpha" {
		t.Fatalf("expected sorted ids, got %q", items[0].ID)
	}

	if items[1].Title != "Beta Title" || items[1].Past != "did beta" {
		t.Fatalf("expected normalized text, got %#v", items[1])
	}
}
