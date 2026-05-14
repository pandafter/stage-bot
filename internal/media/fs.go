package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FSStore struct {
	baseDir    string
	publicBase string
}

func NewFSStore(baseDir, publicBase string) *FSStore {
	_ = os.MkdirAll(baseDir, 0755)
	return &FSStore{baseDir: baseDir, publicBase: strings.TrimRight(publicBase, "/")}
}

func (s *FSStore) Put(_ context.Context, key string, r io.Reader, _ string, _ int64) (string, error) {
	dest := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return s.URL(key), nil
}

func (s *FSStore) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(s.baseDir, filepath.FromSlash(key)))
}

func (s *FSStore) URL(key string) string {
	return fmt.Sprintf("%s/%s", s.publicBase, key)
}
