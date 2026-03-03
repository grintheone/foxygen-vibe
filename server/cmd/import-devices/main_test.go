package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyDevices(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "device_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "classificator": "class-2",
	        "serialNumber": " 123 ",
	        "properties": {"k":"v"},
	        "connectedToLis": true,
	        "isUsed": true
	      }
	    },
	    {
	      "doc": {
	        "_id": "classificator_1_ignore-me",
	        "title": "Ignored"
	      }
	    },
	    {
	      "doc": {
	        "_id": "device_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "classificator": "class-1",
	        "serialNumber": "",
	        "properties": null,
	        "connectedToLis": false
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyDevices(path)
	if err != nil {
		t.Fatalf("loadLegacyDevices: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(items))
	}

	if items[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected items sorted by id, got %q", items[0].ID)
	}

	if string(items[0].Properties) != "{}" {
		t.Fatalf("expected null properties to default to empty object, got %s", string(items[0].Properties))
	}

	if items[0].IsUsed {
		t.Fatalf("expected missing isUsed to default false")
	}

	if items[1].SerialNumber != "123" {
		t.Fatalf("expected serial number trimmed, got %q", items[1].SerialNumber)
	}

	if !items[1].ConnectedToLis || !items[1].IsUsed {
		t.Fatalf("expected booleans preserved, got connected=%v isUsed=%v", items[1].ConnectedToLis, items[1].IsUsed)
	}
}
