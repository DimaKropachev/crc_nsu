package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"career-consultations-backend/internal/config"
	"career-consultations-backend/internal/models"
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

type memoryResume struct{ *bytes.Reader }

func (memoryResume) Close() error { return nil }

func TestValidateResume(t *testing.T) {
	makeDOCX := func(name, text string) []byte {
		t.Helper()
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		entry, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, text); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	for _, tc := range []struct {
		name  string
		ext   string
		data  []byte
		valid bool
	}{
		{"short PDF", ".pdf", []byte("%PDF-1.7"), true},
		{"empty PDF", ".pdf", nil, false},
		{"wrong PDF signature", ".pdf", []byte("not a PDF"), false},
		{"unsupported extension", ".txt", []byte("%PDF-1.7"), false},
		{"DOCX", ".docx", makeDOCX("word/document.xml", "<document/>"), true},
		{"missing document", ".docx", makeDOCX("other.xml", "<document/>"), false},
		{"oversized entry", ".docx", makeDOCX("word/document.xml", strings.Repeat("a", (10<<20)+1)), false},
		{"broken ZIP", ".docx", []byte("PKbroken"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := memoryResume{bytes.NewReader(tc.data)}
			mimeType, status, err := validateResume(f, tc.ext, int64(len(tc.data)))
			if (err == nil) != tc.valid {
				t.Fatalf("validateResume error = %v, valid = %v", err, tc.valid)
			}
			if tc.valid {
				if status != http.StatusOK || mimeType == "" {
					t.Fatalf("unexpected success: status=%d MIME=%q", status, mimeType)
				}
				remaining, err := io.ReadAll(f)
				if err != nil || !bytes.Equal(remaining, tc.data) {
					t.Fatalf("validation changed file position: %v", err)
				}
			}
		})
	}
}

func TestUploadResumeBoundsRequest(t *testing.T) {
	s := &Server{cfg: config.Config{MaxUploadBytes: 1024}, logger: zap.NewNop()}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", (1<<20)+2048)))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	r = r.WithContext(context.WithValue(r.Context(), principalKey, principal{User: models.User{Role: "student"}}))
	w := httptest.NewRecorder()
	s.uploadResume(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	if _, err := io.Copy(io.Discard, r.Body); err == nil {
		t.Fatal("request body was not bounded")
	}
}
