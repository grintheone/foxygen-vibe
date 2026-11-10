package legacytime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseMoscowAndAbsoluteTimestamps(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`"2026-11-09T12:34:56"`, "2026-11-09T09:34:56Z"},
		{`"2026-11-09 12:34:56"`, "2026-11-09T09:34:56Z"},
		{`"2026-11-09T12:34:56.123"`, "2026-11-09T09:34:56.123Z"},
		{`"2026-11-09"`, "2026-11-08T21:00:00Z"},
		{`"2026-11-09T12:34:56+03:00"`, "2026-11-09T09:34:56Z"},
		{`"2026-11-09T12:34:56+07:00"`, "2026-11-09T05:34:56Z"},
		{`"2026-11-09T09:34:56Z"`, "2026-11-09T09:34:56Z"},
		{`"2026-11-09T09:34:56.123456789Z"`, "2026-11-09T09:34:56.123456789Z"},
		{`1704164645000`, "2024-01-02T03:04:05Z"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			value, err := Parse(json.RawMessage(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if got := value.Format(time.RFC3339Nano); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
			// API and sidecar emit UTC strings: consuming that output again must not shift it.
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			again, err := Parse(encoded)
			if err != nil || !again.Equal(*value) {
				t.Fatalf("round trip changed instant: %v, %v", again, err)
			}
		})
	}
}

func TestParseMissingAndInvalidTimestamps(t *testing.T) {
	for _, raw := range []string{"", "null", `""`, `"  "`} {
		value, err := Parse(json.RawMessage(raw))
		if err != nil || value != nil {
			t.Fatalf("Parse(%q) = %v, %v", raw, value, err)
		}
	}
	for _, raw := range []string{`true`, `{}`, `"invalid"`, `"2026-02-30T12:00:00"`} {
		if _, err := Parse(json.RawMessage(raw)); err == nil {
			t.Fatalf("expected error for %s", raw)
		}
	}
}
