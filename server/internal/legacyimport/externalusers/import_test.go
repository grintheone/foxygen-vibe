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
	        "_id": "user_1_12121212-1212-1212-1212-121212121212",
	        "firstName": "Ivan",
	        "lastName": "Petrov"
	      }
	    },
	    {
	      "doc": {
	        "_id": "externalUser_1_13131313-1313-1313-1313-131313131313",
	        "title": "Petrov Ivan"
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

	if len(items) != 5 {
		t.Fatalf("expected 5 external users, got %d", len(items))
	}

	if items[0].ID != "13131313-1313-1313-1313-131313131313" || items[0].LinkedUserID != "12121212-1212-1212-1212-121212121212" {
		t.Fatalf("expected reversed-name title match to link and sort first, got %#v", items[0])
	}

	if items[1].ID != "99999999-9999-9999-9999-999999999999" || items[1].LinkedUserID != "" {
		t.Fatalf("expected unmatched title to stay unlinked, got %#v", items[1])
	}

	if items[2].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || items[2].LinkedUserID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected shared id to link directly, got %#v", items[2])
	}

	if items[3].ID != "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee" || items[3].LinkedUserID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("expected unique title match to link, got %#v", items[3])
	}

	if items[4].ID != "ffffffff-ffff-ffff-ffff-ffffffffffff" || items[4].LinkedUserID != "" {
		t.Fatalf("expected ambiguous title match to stay unlinked, got %#v", items[4])
	}
}
