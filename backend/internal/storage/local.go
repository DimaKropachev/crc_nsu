package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Local struct{ dir string }

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Local{dir: abs}, nil
}
func (s *Local) Save(src io.Reader, ext string, maxBytes int64) (string, int64, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", 0, err
	}
	key := hex.EncodeToString(b[:]) + ext
	path := filepath.Join(s.dir, key)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", 0, err
	}
	// Close before removing: Windows cannot remove an open file.
	saved := false
	defer func() {
		_ = f.Close()
		if !saved {
			_ = os.Remove(path)
		}
	}()
	n, err := io.Copy(f, io.LimitReader(src, maxBytes+1))
	if err != nil {
		return "", 0, err
	}
	if n > maxBytes {
		return "", n, fmt.Errorf("file too large")
	}
	if n == 0 {
		return "", 0, fmt.Errorf("empty file")
	}
	if err := f.Sync(); err != nil {
		return "", 0, err
	}
	if err := f.Close(); err != nil {
		return "", 0, err
	}
	saved = true
	return key, n, nil
}
func (s *Local) Open(key string) (*os.File, error) {
	if filepath.Base(key) != key || key == "." || key == "" {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(s.dir, key))
}
func (s *Local) Delete(key string) error {
	if filepath.Base(key) != key || key == "." || key == "" {
		return os.ErrNotExist
	}
	err := os.Remove(filepath.Join(s.dir, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
