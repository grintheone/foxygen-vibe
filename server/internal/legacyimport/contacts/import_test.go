package contacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyContacts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "contact_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "ref": "client-2",
	        "firstName": " Beta ",
	        "middleName": " Team ",
	        "lastName": " Lead ",
	        "position": " chief   nurse ",
	        "phone": " 123 ",
	        "email": " EXAMPLE@TEST.COM "
	      }
	    },
	    {
	      "doc": {
	        "_id": "client_1_ignore-me",
	        "title": "Ignored"
	      }
	    },
	    {
	      "doc": {
	        "_id": "contact_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "ref": "",
	        "firstName": "Solo Name",
	        "middleName": "",
	        "lastName": "",
	        "position": "",
	        "phone": "",
	        "email": ""
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyContacts(path)
	if err != nil {
		t.Fatalf("loadLegacyContacts: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(items))
	}

	if items[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected contacts sorted by id, got %q", items[0].ID)
	}

	if items[0].Name != "Solo Name" {
		t.Fatalf("expected single-field name preserved, got %q", items[0].Name)
	}

	if items[1].Name != "Beta Team Lead" {
		t.Fatalf("expected name to combine legacy fields, got %q", items[1].Name)
	}

	if items[1].Position != "chief nurse" {
		t.Fatalf("expected position normalized, got %q", items[1].Position)
	}

	if items[1].Phone != "123" {
		t.Fatalf("expected phone trimmed, got %q", items[1].Phone)
	}

	if items[1].Email != "example@test.com" {
		t.Fatalf("expected email normalized, got %q", items[1].Email)
	}

	if items[1].ClientID != "client-2" {
		t.Fatalf("expected client ref preserved, got %q", items[1].ClientID)
	}
}
