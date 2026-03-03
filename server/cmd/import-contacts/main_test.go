package main

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

	contacts, err := loadLegacyContacts(path)
	if err != nil {
		t.Fatalf("loadLegacyContacts: %v", err)
	}

	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}

	if contacts[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected contacts sorted by id, got %q", contacts[0].ID)
	}

	if contacts[0].Name != "Solo Name" {
		t.Fatalf("expected single-field name preserved, got %q", contacts[0].Name)
	}

	if contacts[1].Name != "Beta Team Lead" {
		t.Fatalf("expected name to combine legacy fields, got %q", contacts[1].Name)
	}

	if contacts[1].Position != "chief nurse" {
		t.Fatalf("expected position normalized, got %q", contacts[1].Position)
	}

	if contacts[1].Phone != "123" {
		t.Fatalf("expected phone trimmed, got %q", contacts[1].Phone)
	}

	if contacts[1].Email != "example@test.com" {
		t.Fatalf("expected email normalized, got %q", contacts[1].Email)
	}

	if contacts[1].ClientID != "client-2" {
		t.Fatalf("expected client ref preserved, got %q", contacts[1].ClientID)
	}
}
