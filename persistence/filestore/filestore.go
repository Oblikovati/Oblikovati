// SPDX-License-Identifier: GPL-2.0-only

// Package filestore is the shared load/save core behind the per-user YAML config
// stores (dialogmemory, userprefs, viewstate, addinstate, keymap, options) — six
// packages used to carry byte-identical copies of it (M40 audit G3, #1651). It
// wraps yaml.v3 behind the project-owned yamlcodec seam (ADR-0020). The generic
// layer stays dumb I/O on purpose: defaults injection, field migration, and any
// other domain semantics live in the owning package.
package filestore

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/persistence/yamlcodec"
)

// FileStore persists one YAML document of type T at a fixed path, e.g.
//
//	store := filestore.New[Prefs](path)
//	prefs, found, err := store.Load()
type FileStore[T any] struct{ path string }

// New returns a store backed by the YAML file at path.
func New[T any](path string) *FileStore[T] { return &FileStore[T]{path: path} }

// Path returns the file the store reads and writes.
func (s *FileStore[T]) Path() string { return s.path }

// Load returns the stored document. A missing file is found=false with a zero T
// (fresh install, no error); a read or parse failure names the offending file.
func (s *FileStore[T]) Load() (T, bool, error) {
	var v T
	found, err := s.LoadInto(&v)
	if err != nil {
		var zero T // v may be partially filled by a failed unmarshal
		return zero, false, err
	}
	return v, found, nil
}

// LoadInto unmarshals the stored document over *v, so a caller can pre-fill
// defaults that absent YAML keys keep (the app/options pattern). A missing file
// leaves v untouched and reports found=false.
func (s *FileStore[T]) LoadInto(v *T) (bool, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("filestore: read %q: %w", s.path, err)
	}
	if err := yamlcodec.Unmarshal(raw, v); err != nil {
		return false, fmt.Errorf("filestore: parse %q: %w", s.path, err)
	}
	return true, nil
}

// Save writes v, creating the config directory on first use. The write is atomic
// (temp file + rename) so a crash mid-write never leaves a truncated config.
func (s *FileStore[T]) Save(v T) error {
	data, err := yamlcodec.Marshal(v)
	if err != nil {
		return fmt.Errorf("filestore: marshal for %q: %w", s.path, err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("filestore: create config dir for %q: %w", s.path, err)
	}
	return s.replaceAtomic(data)
}

// replaceAtomic stages data in a sibling temp file and renames it over the store's
// path — the same-directory rename is what makes the swap atomic on POSIX.
func (s *FileStore[T]) replaceAtomic(data []byte) error {
	dir, base := filepath.Split(s.path)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("filestore: create temp for %q: %w", s.path, err)
	}
	if err := fillTemp(tmp, data); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("filestore: write temp for %q: %w", s.path, err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("filestore: replace %q: %w", s.path, err)
	}
	return nil
}

// fillTemp writes data and widens the temp file's mode to the 0644 the stores have
// always written (os.CreateTemp creates 0600), closing on every path.
func fillTemp(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
