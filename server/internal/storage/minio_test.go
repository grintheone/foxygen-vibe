package storage

import "testing"

func TestConfigValidateRequiresCompleteSettings(t *testing.T) {
	t.Parallel()

	config := Config{
		Endpoint:    "localhost:9000",
		AccessKeyID: "minioadmin",
	}

	if err := config.Validate(); err == nil {
		t.Fatal("expected Validate to fail for incomplete config")
	}
}

func TestConfigEnabledDetectsConfiguredStorage(t *testing.T) {
	t.Parallel()

	if (Config{}).Enabled() {
		t.Fatal("expected empty config to be disabled")
	}

	if !(Config{Bucket: "foxygen-vibe"}).Enabled() {
		t.Fatal("expected config with any MinIO setting to be enabled")
	}
}

func TestTicketAttachmentObjectKeyKeepsSafeExtension(t *testing.T) {
	t.Parallel()

	key := TicketAttachmentObjectKey("ticket-1", "attachment-2", "photo.final.JPG")
	if key != "tickets/ticket-1/attachment-2.jpg" {
		t.Fatalf("expected sanitized object key, got %q", key)
	}
}
