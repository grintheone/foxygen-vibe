package regions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyRegions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "region_1_22222222-2222-2222-2222-222222222222",
	        "title": "Beta"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_user-a",
	        "firstName": "Alex"
	      }
	    },
	    {
	      "doc": {
	        "_id": "region_1_11111111-1111-1111-1111-111111111111",
	        "title": " Alpha "
	      }
	    },
	    {
	      "doc": {
	        "_id": "region_1_33333333-3333-3333-3333-333333333333",
	        "title": ""
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyRegions(path)
	if err != nil {
		t.Fatalf("loadLegacyRegions: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(items))
	}

	if items[0].ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected regions to be sorted by legacy id, got %q", items[0].ID)
	}

	if items[0].Title != "Alpha" {
		t.Fatalf("expected region title to be trimmed, got %q", items[0].Title)
	}

	if items[1].ID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected second region id %q", items[1].ID)
	}
}
