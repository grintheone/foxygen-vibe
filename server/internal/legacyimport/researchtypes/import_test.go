package researchtypes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyResearchTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "researchType_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "title": " Beta   Type "
	      }
	    },
	    {
	      "doc": {
	        "_id": "manufacturer_1_ignore-me",
	        "title": "Ignored"
	      }
	    },
	    {
	      "doc": {
	        "_id": "researchType_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "title": "Alpha Type"
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyResearchTypes(path)
	if err != nil {
		t.Fatalf("loadLegacyResearchTypes: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 research types, got %d", len(items))
	}

	if items[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected items sorted by id, got %q", items[0].ID)
	}

	if items[1].Title != "Beta Type" {
		t.Fatalf("expected title normalized, got %q", items[1].Title)
	}
}
