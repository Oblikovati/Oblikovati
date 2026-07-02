// SPDX-License-Identifier: GPL-2.0-only

// Package dialogmemory persists the user's remembered dialog choices (M05-F09,
// #616): which balloon tips they suppressed ("don't show again") and which prompt
// answers they chose to keep — one YAML file in the OS config directory, beside the
// other per-user stores.
package dialogmemory

import (
	"oblikovati.org/persistence/filestore"
	"oblikovati.org/userconfig"
)

// Memory is the remembered choices: suppressed balloon-tip ids and prompt answers
// keyed by prompt id.
type Memory struct {
	SuppressedTips []string          `yaml:"suppressedTips,omitempty"`
	PromptAnswers  map[string]string `yaml:"promptAnswers,omitempty"`
}

// Store loads and saves the remembered choices.
type Store interface {
	Load() (Memory, error)
	Save(Memory) error
}

// FileStore persists the memory to a YAML file (the shared filestore core, #1651).
type FileStore struct{ file *filestore.FileStore[Memory] }

// DefaultPath is the per-user file: ~/.oblikovati/dialog-memory.yaml on Linux/macOS.
func DefaultPath() (string, error) {
	return userconfig.File("dialog-memory.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{file: filestore.New[Memory](path)}
}

// Load reads the remembered choices; a missing file is an empty memory.
func (s *FileStore) Load() (Memory, error) {
	m, _, err := s.file.Load()
	return m, err
}

// Save writes the remembered choices, creating the config directory on first use.
func (s *FileStore) Save(m Memory) error { return s.file.Save(m) }
