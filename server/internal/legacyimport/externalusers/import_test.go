package externalusers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyExternalUsers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "user_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "firstName": "Beta",
	        "lastName": "User"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_cccccccc-cccc-cccc-cccc-cccccccccccc",
	        "firstName": "Gamma",
	        "lastName": "User"
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_dddddddd-dddd-dddd-dddd-dddddddddddd",
	        "firstName": "Gamma",
	        "lastName": "User"
	      }
	    },
	    {
	      "doc": {
	        "_id": "externalUser_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "title": " Shared Identity "
	      }
	    },
	    {
	      "doc": {
	        "_id": "user_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "firstName": "Real",
	        "lastName": "User"
	      }
	    },
	    {
	      "doc": {
	        "_id": "externalUser_1_eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
	        "title": " Beta User "
	      }
	    },
	    {
	      "doc": {
	        "_id": "externalUser_1_ffffffff-ffff-ffff-ffff-ffffffffffff",
	        "title": "Gamma User"
	      }
	    },
	    {
	      "doc": {
	        "_id": "externalUser_1_99999999-9999-9999-9999-999999999999",
	        "title": "No Match"
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyExternalUsers(path)
	if err != nil {
		t.Fatalf("loadLegacyExternalUsers: %v", err)
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 external users, got %d", len(items))
	}

	if items[0].ID != "99999999-9999-9999-9999-999999999999" {
		t.Fatalf("expected items sorted by id, got %q", items[0].ID)
	}

	if items[1].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || items[1].LinkedUserID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected shared id to link directly, got %#v", items[1])
	}

	if items[2].ID != "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee" || items[2].LinkedUserID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("expected unique title match to link, got %#v", items[2])
	}

	if items[3].ID != "ffffffff-ffff-ffff-ffff-ffffffffffff" || items[3].LinkedUserID != "" {
		t.Fatalf("expected ambiguous title match to stay unlinked, got %#v", items[3])
	}
}
