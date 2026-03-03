package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyAttachments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "dump.json")

	content := `{
	  "rows": [
	    {
	      "doc": {
	        "_id": "ticket_1_bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	        "attachments": [
	          {
	            "id": " b-id ",
	            "name": " B Name ",
	            "mediaType": " image/jpeg ",
	            "ext": " jpg "
	          }
	        ]
	      }
	    },
	    {
	      "doc": {
	        "_id": "ticket_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	        "attachments": [
	          {
	            "id": "a-id",
	            "name": "A Name",
	            "mediaType": "application/pdf",
	            "ext": "pdf"
	          },
	          {
	            "id": "",
	            "name": "skip",
	            "mediaType": "image/jpeg",
	            "ext": "jpg"
	          }
	        ]
	      }
	    }
	  ]
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dump: %v", err)
	}

	items, err := loadLegacyAttachments(path)
	if err != nil {
		t.Fatalf("loadLegacyAttachments: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(items))
	}

	if items[0].RefID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || items[0].ID != "a-id" {
		t.Fatalf("expected sorted attachments, got %#v", items[0])
	}

	if items[1].ID != "b-id" || items[1].Name != "B Name" || items[1].MediaType != "image/jpeg" || items[1].Ext != "jpg" {
		t.Fatalf("expected trimmed fields, got %#v", items[1])
	}
}
