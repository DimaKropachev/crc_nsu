package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		valid bool
	}{
		{"object", `{"enabled":true}`, true},
		{"unknown field", `{"unknown":true}`, false},
		{"second object", `{"enabled":true}{}`, false},
		{"malformed", `{"enabled":`, false},
		{"empty", "", false},
		{"exact limit", `{}` + strings.Repeat(" ", (1<<20)-2), true},
		// A LimitReader would silently hide everything beyond the limit as EOF.
		{"oversized whitespace", `{}` + strings.Repeat(" ", 1<<20), false},
		{"hidden second object", `{}` + strings.Repeat(" ", (1<<20)-2) + `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			var dst struct {
				Enabled bool `json:"enabled"`
			}
			err := decodeJSON(r, &dst)
			if (err == nil) != tc.valid {
				t.Fatalf("decodeJSON error = %v, valid = %v", err, tc.valid)
			}
		})
	}
}
