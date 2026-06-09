// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"oblikovati.org/persistence/yamlcodec"
	"oblikovati.org/userconfig"
)

// FileSystem is the thin filesystem seam the [Store] depends on, so theme IO is testable
// with an in-memory fake and never touches a real disk in unit tests (CLAUDE.md). The
// real implementation is [OSFileSystem]. ReadDir must return an empty slice and nil error
// for a directory that does not exist yet (first run), so Load needs no special-casing.
type FileSystem interface {
	ReadDir(dir string) ([]string, error)     // base names of the directory's files
	ReadFile(path string) ([]byte, error)     // file contents
	WriteFile(path string, data []byte) error // create/overwrite, making parent dirs
	Remove(path string) error                 // delete a file
}

// Store loads and saves custom themes and the active-theme preference under a root
// directory (the app's per-user config dir). Built-in themes are never persisted; only
// the user's customs and which theme is selected.
type Store struct {
	root string
	fs   FileSystem
}

// NewStore builds a store rooted at dir, backed by fs.
func NewStore(dir string, fs FileSystem) *Store { return &Store{root: dir, fs: fs} }

// DefaultRoot is the shared per-user Oblikovati config directory the app stores themes
// under (~/.oblikovati on Linux/macOS, %AppData%\oblikovati on Windows) — see userconfig.
func DefaultRoot() (string, error) {
	return userconfig.Dir()
}

// themeFile is the on-disk YAML shape of one custom theme: a name and the full color
// snapshot as "#RRGGBBAA" hex. Kind is always custom (built-ins are not saved), so it is
// not stored.
type themeFile struct {
	Name   string            `yaml:"name"`
	Colors map[string]string `yaml:"colors"`
}

// prefsFile is the tiny preferences document holding the selected theme's name.
type prefsFile struct {
	ActiveTheme string `yaml:"activeTheme"`
}

// Load reads the active-theme preference and every custom theme file. A missing config
// dir (first run) yields no customs and an empty active name — [NewLibrary] then falls
// back to the Dark default. A single malformed theme file is skipped (its error is
// returned alongside the good ones) rather than aborting the whole load.
func (s *Store) Load() (customs []*Theme, active string, err error) {
	active = s.loadActive()
	names, derr := s.fs.ReadDir(s.themesDir())
	if derr != nil {
		return nil, active, fmt.Errorf("theme: read themes dir %q: %w", s.themesDir(), derr)
	}
	sort.Strings(names) // deterministic load order
	for _, name := range names {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		t, lerr := s.loadTheme(filepath.Join(s.themesDir(), name))
		if lerr != nil {
			err = lerr // remember the last bad file but keep loading the good ones
			continue
		}
		customs = append(customs, t)
	}
	return customs, active, err
}

// SaveTheme writes one custom theme to its file. It errors (rather than silently
// skipping) when asked to persist a built-in, which has no business on disk.
func (s *Store) SaveTheme(t *Theme) error {
	if t.Kind() != KindCustom {
		return fmt.Errorf("theme: refusing to save built-in %q", t.Name())
	}
	data, err := yamlcodec.Marshal(themeFile{Name: t.Name(), Colors: t.Palette().Hex()})
	if err != nil {
		return fmt.Errorf("theme: marshal %q: %w", t.Name(), err)
	}
	return s.fs.WriteFile(s.themePath(t.Name()), data)
}

// RemoveTheme deletes a custom theme's file.
func (s *Store) RemoveTheme(name string) error {
	return s.fs.Remove(s.themePath(name))
}

// SaveActive persists the selected theme's name to the preferences file.
func (s *Store) SaveActive(name string) error {
	data, err := yamlcodec.Marshal(prefsFile{ActiveTheme: name})
	if err != nil {
		return fmt.Errorf("theme: marshal preferences: %w", err)
	}
	return s.fs.WriteFile(s.prefsPath(), data)
}

// loadActive reads the active-theme name, returning "" when the preferences file is
// absent or unreadable (the library then defaults to Dark).
func (s *Store) loadActive() string {
	data, err := s.fs.ReadFile(s.prefsPath())
	if err != nil {
		return ""
	}
	var p prefsFile
	if yamlcodec.Unmarshal(data, &p) != nil {
		return ""
	}
	return p.ActiveTheme
}

// loadTheme decodes one custom theme file into a Theme, naming the file on any error.
func (s *Store) loadTheme(path string) (*Theme, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("theme: read %q: %w", path, err)
	}
	var tf themeFile
	if err := yamlcodec.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("theme: parse %q: %w", path, err)
	}
	if tf.Name == "" {
		return nil, fmt.Errorf("theme: %q has no name", path)
	}
	palette, err := NewPalette(tf.Colors)
	if err != nil {
		return nil, fmt.Errorf("theme: %q: %w", path, err)
	}
	return New(tf.Name, KindCustom, palette), nil
}

func (s *Store) themesDir() string { return filepath.Join(s.root, "themes") }
func (s *Store) prefsPath() string { return filepath.Join(s.root, "preferences.yaml") }
func (s *Store) themePath(n string) string {
	return filepath.Join(s.themesDir(), slug(n)+".yaml")
}

// slug turns a theme name into a safe file base: lower-case, non-alphanumerics collapsed
// to '-'. The on-disk name is cosmetic — the authoritative name is the file's `name`
// field — so a slug collision only risks one custom overwriting another, which the
// unique-name rule in [Library.Duplicate] already prevents at the source.
func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "theme"
	}
	return out
}
