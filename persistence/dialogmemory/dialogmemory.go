// SPDX-License-Identifier: GPL-2.0-only

// Package dialogmemory persists the user's remembered dialog choices (M05-F09,
// #616): which balloon tips they suppressed ("don't show again") and which prompt
// answers they chose to keep — one YAML file in the OS config directory, beside the
// other per-user stores.
package dialogmemory

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/userconfig"
	"oblikovati.org/yamlcodec"
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

// FileStore persists the memory to a YAML file.
type FileStore struct{ path string }

// DefaultPath is the per-user file: ~/.oblikovati/dialog-memory.yaml on Linux/macOS.
func DefaultPath() (string, error) {
	return userconfig.File("dialog-memory.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// Load reads the remembered choices; a missing file is an empty memory.
func (s *FileStore) Load() (Memory, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return Memory{}, nil
	}
	if err != nil {
		return Memory{}, fmt.Errorf("dialogmemory: read %q: %w", s.path, err)
	}
	var m Memory
	if err := yamlcodec.Unmarshal(raw, &m); err != nil {
		return Memory{}, fmt.Errorf("dialogmemory: parse %q: %w", s.path, err)
	}
	return m, nil
}

// Save writes the remembered choices, creating the config directory on first use.
func (s *FileStore) Save(m Memory) error {
	data, err := yamlcodec.Marshal(m)
	if err != nil {
		return fmt.Errorf("dialogmemory: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("dialogmemory: create config dir for %q: %w", s.path, err)
	}
	return os.WriteFile(s.path, data, 0o644)
}
