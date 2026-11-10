package storage

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) {
	return copy(p, "partial"), errors.New("read failed")
}

func TestSaveCleansUpFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  io.Reader
	}{
		{"empty", strings.NewReader("")},
		{"oversized", strings.NewReader("12345")},
		{"reader error", failingReader{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			s, err := NewLocal(dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := s.Save(tc.src, ".pdf", 4); err == nil {
				t.Fatal("expected save error")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("failed save left %d files", len(entries))
			}
		})
	}
}

func TestLocalRoundTrip(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key, size, err := s.Save(strings.NewReader("data"), ".pdf", 4)
	if err != nil {
		t.Fatal(err)
	}
	if size != 4 {
		t.Fatalf("size = %d", size)
	}
	f, err := s.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil || string(data) != "data" {
		t.Fatalf("read = %q, %v", data, err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open("../outside.pdf"); !os.IsNotExist(err) {
		t.Fatalf("traversal error = %v", err)
	}
}

func TestLocalRejectsInvalidNames(t *testing.T) {
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{"", ".exe", "/../outside.pdf", `\outside.pdf`} {
		if _, _, err := s.Save(strings.NewReader("data"), ext, 4); err == nil {
			t.Errorf("accepted extension %q", ext)
		}
	}
	for _, key := range []string{"", ".", "../outside.pdf", `..\outside.pdf`, strings.Repeat("z", 32) + ".pdf", strings.Repeat("a", 32) + ".exe"} {
		if _, err := s.Open(key); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Open(%q) error = %v", key, err)
		}
		if err := s.Delete(key); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Delete(%q) error = %v", key, err)
		}
	}
}
