package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyClients(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "client_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "title": " Beta   Hospital ",
	        "address": " 123 Main St ",
	        "region": "region-2",
	        "location": [{"lat": 1.2, "lng": 3.4}],
	        "laboratorySystem": null
	      }
	    },
	    {
	      "doc": {
	        "_id": "region_1_region-1",
	        "title": "Ignored"
	      }
	    },
	    {
	      "doc": {
	        "_id": "client_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "title": "Alpha Clinic",
	        "address": "",
	        "region": "",
	        "location": null,
	        "laboratorySystem": " 11111111-1111-1111-1111-111111111111 "
	      }
	    },
	    {
	      "doc": {
	        "_id": "client_1_cccccccc-cccc-cccc-cccc-cccccccccccc",
	        "title": "   ",
	        "address": "skip me",
	        "region": "region-3"
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	clients, err := loadLegacyClients(path)
	if err != nil {
		t.Fatalf("loadLegacyClients: %v", err)
	}

	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}

	if clients[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected clients sorted by id, got %q", clients[0].ID)
	}

	if clients[1].Title != "Beta Hospital" {
		t.Fatalf("expected title whitespace normalized, got %q", clients[1].Title)
	}

	if clients[1].Address != "123 Main St" {
		t.Fatalf("expected address trimmed, got %q", clients[1].Address)
	}

	if clients[1].RegionID != "region-2" {
		t.Fatalf("expected region to be preserved, got %q", clients[1].RegionID)
	}

	if clients[0].LaboratorySystem == nil || *clients[0].LaboratorySystem != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected laboratory system to be trimmed, got %#v", clients[0].LaboratorySystem)
	}

	if len(clients[1].Location) == 0 {
		t.Fatalf("expected location payload to be preserved")
	}
}
