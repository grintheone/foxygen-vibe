package api

import (
	"testing"
	"time"
)

func TestMoscowArchiveDate(t *testing.T) {
	date, err := parseArchiveDate("2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	if got := date.Format(time.RFC3339); got != "2026-10-31T21:00:00Z" {
		t.Fatalf("archive midnight = %s, want Moscow midnight in UTC", got)
	}
}

func TestMoscowTicketDateInput(t *testing.T) {
	for _, value := range []string{"2026-11-06", "2026-11-06T12:00:00+03:00"} {
		date, err := parseTicketDateInput(value)
		if err != nil {
			t.Fatal(err)
		}
		if got := date.Format(time.RFC3339); got != "2026-11-06T09:00:00Z" {
			t.Fatalf("parseTicketDateInput(%q) = %s", value, got)
		}
	}
}
