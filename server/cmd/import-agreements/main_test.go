package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyAgreements(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "device_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "bindings": [
	          {"client": " client-2 "}
	        ]
	      }
	    },
	    {
	      "doc": {
	        "_id": "device_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "bindings": [
	          {"client": "client-1"},
	          {"client": ""}
	        ]
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyAgreements(path)
	if err != nil {
		t.Fatalf("loadLegacyAgreements: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 agreements, got %d", len(items))
	}

	if items[0].DeviceID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || items[0].ClientID != "client-1" {
		t.Fatalf("expected sorted items, got %#v", items[0])
	}

	if items[1].ClientID != "client-2" {
		t.Fatalf("expected trimmed client id, got %#v", items[1])
	}

	if items[0].ID == items[1].ID {
		t.Fatalf("expected deterministic ids to differ")
	}
}
