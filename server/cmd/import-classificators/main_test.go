package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyClassificators(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "classificator_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "title": " Beta   Device ",
	        "manufacturer": {"id": "maker-2", "title": "Ignored"},
	        "researchType": "research-2",
	        "registrationCertificate": {"number": "1"},
	        "maintenanceRegulations": [{"kind": "yearly"}],
	        "attachments": [" a ", "b"],
	        "images": [" c "]
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
	        "_id": "classificator_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "title": "Alpha Device",
	        "manufacturer": "maker-1",
	        "researchType": "",
	        "registrationCertificate": null,
	        "maintenanceRegulations": null,
	        "attachments": [],
	        "images": []
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyClassificators(path)
	if err != nil {
		t.Fatalf("loadLegacyClassificators: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 classificators, got %d", len(items))
	}

	if items[0].ID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected items sorted by id, got %q", items[0].ID)
	}

	if items[0].ManufacturerID != "maker-1" {
		t.Fatalf("expected string manufacturer preserved, got %q", items[0].ManufacturerID)
	}

	if string(items[0].RegistrationCertificate) != "{}" {
		t.Fatalf("expected null registration certificate to fall back to empty object, got %s", string(items[0].RegistrationCertificate))
	}

	if string(items[0].MaintenanceRegulations) != "[]" {
		t.Fatalf("expected null maintenance regulations to fall back to empty array, got %s", string(items[0].MaintenanceRegulations))
	}

	if items[1].Title != "Beta Device" {
		t.Fatalf("expected title normalized, got %q", items[1].Title)
	}

	if items[1].ManufacturerID != "maker-2" {
		t.Fatalf("expected object manufacturer id extracted, got %q", items[1].ManufacturerID)
	}

	if items[1].ResearchTypeID != "research-2" {
		t.Fatalf("expected research type preserved, got %q", items[1].ResearchTypeID)
	}

	if len(items[1].Attachments) != 2 || items[1].Attachments[0] != "a" {
		t.Fatalf("expected attachments trimmed, got %#v", items[1].Attachments)
	}

	if len(items[1].Images) != 1 || items[1].Images[0] != "c" {
		t.Fatalf("expected images trimmed, got %#v", items[1].Images)
	}
}
