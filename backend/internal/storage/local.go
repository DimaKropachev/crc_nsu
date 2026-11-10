package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Local struct{ dir string }

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Local{dir: abs}, nil
}

func (s *Local) Save(src io.Reader, ext string, maxBytes int64) (savedKey string, size int64, err error) {
	if ext != ".pdf" && ext != ".docx" {
		return "", 0, fmt.Errorf("unsupported resume extension")
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", 0, err
	}
	key := hex.EncodeToString(b[:]) + ext
	path := filepath.Join(s.dir, key)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: Random hex basename with allowlisted extension in private storage; O_EXCL rejects existing symlinks.
	if err != nil {
		return "", 0, err
	}
	// Close before removing: Windows cannot remove an open file.
	saved := false
	closed := false
	defer func() {
		if !closed {
			err = errors.Join(err, f.Close())
		}
		if !saved {
			err = errors.Join(err, os.Remove(path))
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
	err = f.Close()
	closed = true
	if err != nil {
		return "", 0, err
	}
	saved = true
	return key, n, nil
}

func (s *Local) Open(key string) (*os.File, error) {
	if !validStorageKey(key) {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(s.dir, key)) //nolint:gosec // G304: validStorageKey permits only generated hex basenames and PDF/DOCX extensions in private storage.
}

func (s *Local) Delete(key string) error {
	if !validStorageKey(key) {
		return os.ErrNotExist
	}
	err := os.Remove(filepath.Join(s.dir, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func validStorageKey(key string) bool {
	ext := filepath.Ext(key)
	if ext != ".pdf" && ext != ".docx" {
		return false
	}
	name := key[:len(key)-len(ext)]
	if len(name) != 32 {
		return false
	}
	_, err := hex.DecodeString(name)
	return err == nil
}
