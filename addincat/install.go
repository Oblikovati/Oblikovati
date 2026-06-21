// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// State is an add-in's status relative to the host: available to install, installed and
// up to date, or installed with a newer version in the catalogue.
type State int

const (
	StateAvailable State = iota
	StateInstalled
	StateUpdateAvailable
)

// AddInStatus pairs a catalogue entry with the host's view of it: its state, the installed
// version (empty when not installed) and the latest catalogue version for the host's API.
type AddInStatus struct {
	Entry            Entry
	State            State
	InstalledVersion string
	LatestVersion    string
}

// InstalledAddIn is one add-in present in the user directory, read from its manifest.json.
type InstalledAddIn struct {
	Name    string
	Version string
	Dir     string
}

// Installer downloads and extracts bundles into the per-user add-ins directory, and reports
// what is installed. It is the host's only writer of that directory.
type Installer struct {
	dir      string
	client   *Client
	platform string
}

// NewInstaller returns an installer rooted at dir (typically UserAddInsDir) using client to
// download bundles, for the current host platform.
func NewInstaller(dir string, client *Client) *Installer {
	return &Installer{dir: dir, client: client, platform: Platform()}
}

// Install downloads the version's bundle for this platform, verifies its SHA-256, and
// extracts it into <dir>/<name>/, replacing any previous install (so it doubles as update).
func (i *Installer) Install(ctx context.Context, e Entry, v Version) error {
	bundle, ok := v.Bundles[i.platform]
	if !ok {
		return fmt.Errorf("addincat: %q %s has no bundle for %s", e.Name, v.Version, i.platform)
	}
	data, err := i.client.Download(ctx, bundle)
	if err != nil {
		return err
	}
	if err := verifyChecksum(data, bundle.SHA256); err != nil {
		return err
	}
	dest, err := i.addInDir(e.Name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("addincat: clear %q: %w", dest, err)
	}
	return extractZip(data, dest)
}

// Uninstall removes an add-in's directory.
func (i *Installer) Uninstall(name string) error {
	dest, err := i.addInDir(name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("addincat: uninstall %q: %w", name, err)
	}
	return nil
}

// Installed lists the add-ins present in the user directory (those with a readable
// manifest.json). A missing directory is an empty list, not an error.
func (i *Installer) Installed() ([]InstalledAddIn, error) {
	entries, err := os.ReadDir(i.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("addincat: read %q: %w", i.dir, err)
	}
	var out []InstalledAddIn
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		sub := filepath.Join(i.dir, de.Name())
		if m, ok := readInstalledManifest(sub); ok {
			out = append(out, InstalledAddIn{Name: m.ID, Version: m.Version, Dir: sub})
		}
	}
	return out, nil
}

// Status merges the catalogue (for the host's API major+minor) with what is installed: every
// catalogue entry plus any installed add-in absent from the catalogue, each tagged with its
// state and versions. It is what the catalogue window renders across its three tabs.
func (i *Installer) Status(catalogue []Entry, major, minor int) ([]AddInStatus, error) {
	installed, err := i.Installed()
	if err != nil {
		return nil, err
	}
	byName := map[string]string{}
	for _, a := range installed {
		byName[a.Name] = a.Version
	}
	seen := map[string]bool{}
	out := make([]AddInStatus, 0, len(catalogue))
	for _, e := range catalogue {
		seen[e.Name] = true
		latest := ""
		if v, ok := e.LatestFor(major, minor); ok {
			latest = v.Version
		}
		out = append(out, statusOf(e, byName[e.Name], latest))
	}
	for _, a := range installed {
		if !seen[a.Name] {
			out = append(out, AddInStatus{
				Entry:            Entry{Name: a.Name, DisplayName: a.Name},
				State:            StateInstalled,
				InstalledVersion: a.Version,
			})
		}
	}
	return out, nil
}

// statusOf classifies one catalogue entry given its installed and latest versions.
func statusOf(e Entry, installedVer, latestVer string) AddInStatus {
	s := AddInStatus{Entry: e, InstalledVersion: installedVer, LatestVersion: latestVer}
	switch {
	case installedVer == "":
		s.State = StateAvailable
	case latestVer != "" && compareSemver(installedVer, latestVer) < 0:
		s.State = StateUpdateAvailable
	default:
		s.State = StateInstalled
	}
	return s
}

// addInDir is <dir>/<name>, rejecting a name that is not a safe single path segment so an
// add-in can never escape the user directory.
func (i *Installer) addInDir(name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || filepath.Clean(name) != name {
		return "", fmt.Errorf("addincat: unsafe add-in name %q", name)
	}
	return filepath.Join(i.dir, name), nil
}

// installedManifest is the subset of a bundle's manifest.json the host reads to know what is
// installed and at which version.
type installedManifest struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// readInstalledManifest finds and parses the manifest.json under an installed add-in's
// directory (it may sit in a subfolder of the bundle). ok is false when none is found.
func readInstalledManifest(dir string) (installedManifest, bool) {
	var found installedManifest
	ok := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "manifest.json" {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		var m installedManifest
		if json.Unmarshal(b, &m) == nil && m.ID != "" {
			found, ok = m, true
			return fs.SkipAll
		}
		return nil
	})
	return found, ok
}

// verifyChecksum checks data's SHA-256 against want (lowercase hex). An empty want skips the
// check (the catalogue always supplies one; this keeps tests that omit it simple).
func verifyChecksum(data []byte, want string) error {
	if want == "" {
		return nil
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("addincat: bundle checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

// extractZip writes every file in a zip archive under dest, refusing any entry whose path
// would escape dest (a zip-slip guard).
func extractZip(data []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("addincat: bundle is not a valid zip: %w", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("addincat: create %q: %w", dest, err)
	}
	for _, f := range zr.File {
		if err := extractOne(f, dest); err != nil {
			return err
		}
	}
	return nil
}

// extractOne writes a single zip entry under dest, preserving its executable bit (add-in
// binaries must stay runnable).
func extractOne(f *zip.File, dest string) error {
	target, err := safeJoin(dest, f.Name)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("addincat: create dir for %q: %w", target, err)
	}
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("addincat: open entry %q: %w", f.Name, err)
	}
	defer func() { _ = rc.Close() }()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode()|0o600)
	if err != nil {
		return fmt.Errorf("addincat: create %q: %w", target, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, rc); err != nil { //nolint:gosec // size bounded by the downloaded bundle
		return fmt.Errorf("addincat: write %q: %w", target, err)
	}
	return nil
}

// safeJoin joins dest and a zip entry name, erroring if the entry is not a local path (it is
// absolute or escapes dest via ".."). filepath.IsLocal is the lexical zip-slip guard: it
// rejects exactly the entries that would write outside the install directory.
func safeJoin(dest, name string) (string, error) {
	if !filepath.IsLocal(name) {
		return "", fmt.Errorf("addincat: bundle entry %q escapes the install directory", name)
	}
	return filepath.Join(dest, name), nil
}
