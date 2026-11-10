// Package legacytime normalizes timestamps from legacy ticket documents.
package legacytime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var moscow = time.FixedZone("MSK", 3*60*60)

// Parse preserves absolute timestamps and interprets offset-free legacy strings
// as Moscow wall time. The result is always UTC, ready for timestamp columns and
// RFC3339 serialization. Never subtract an offset from Unix milliseconds or Z.
func Parse(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var millis int64
	if err := json.Unmarshal(raw, &millis); err == nil {
		value := time.UnixMilli(millis).UTC()
		return &value, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, fmt.Errorf("unsupported timestamp payload %s", string(raw))
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if value, err := time.Parse(time.RFC3339Nano, text); err == nil {
		utc := value.UTC()
		return &utc, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if value, err := time.ParseInLocation(layout, text, moscow); err == nil {
			utc := value.UTC()
			return &utc, nil
		}
	}
	return nil, fmt.Errorf("unsupported time string %q", text)
}
