// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/api"
	"oblikovati.org/api/contract"
	"oblikovati.org/app"
)

// addInLib is the platform-specific handle to a loaded shared-library add-in. The
// Unix (dlopen) and Windows implementations satisfy it; loader.go stays portable.
type addInLib interface {
	id() string
	manifest() string
	apiVersion() (major, minor int, present bool)
	activate() error
	deactivate() error
	notify(b []byte) error
	hasAutomation() bool
	automation(method string, req []byte) ([]byte, error)
	close() error
}

// LoadedAddIn adapts a shared-library add-in to app.AddIn, so the existing add-in
// lifecycle (session.AddIns().Register/Activate) drives a real .so/.dll. Notify
// forwards host events to the add-in over the C ABI.
type LoadedAddIn struct {
	lib  addInLib
	id   string
	path string
}

// A loaded add-in is the host's contract.AddInAutomation implementation (#1619):
// automation calls from other add-ins fan out through CallAutomation over the C ABI.
var _ contract.AddInAutomation = (*LoadedAddIn)(nil)

// ID is the add-in's stable id (from ObkAddInId).
func (a *LoadedAddIn) ID() string { return a.id }

// Manifest is the add-in's JSON manifest (from ObkAddInManifest).
func (a *LoadedAddIn) Manifest() string { return a.lib.manifest() }

// Path is the shared-library file the add-in was loaded from.
func (a *LoadedAddIn) Path() string { return a.path }

// Activate calls ObkAddInActivate; it blocks until the add-in returns, which may
// involve host-calls, so the session goroutine must be draining the Dispatcher.
func (a *LoadedAddIn) Activate(*app.Session) error { return a.lib.activate() }

// Deactivate calls ObkAddInDeactivate.
func (a *LoadedAddIn) Deactivate(*app.Session) error { return a.lib.deactivate() }

// Notify pushes a serialized host event to the add-in (ObkAddInNotify).
func (a *LoadedAddIn) Notify(b []byte) error { return a.lib.notify(b) }

// HasAutomation reports whether the add-in exports the optional ObkAddInAutomation
// entry (app.AddInAutomationProbe), so the registry lists hasAutomation truthfully.
func (a *LoadedAddIn) HasAutomation() bool { return a.lib.hasAutomation() }

// CallAutomation routes an automation request to the add-in's optional
// ObkAddInAutomation export (contract.AddInAutomation).
func (a *LoadedAddIn) CallAutomation(method string, args []byte) ([]byte, error) {
	return a.lib.automation(method, args)
}

// Close unloads the shared library (dlclose/FreeLibrary).
func (a *LoadedAddIn) Close() error { return a.lib.close() }

// IncompatibleAddIn records an add-in [LoadDir] refused to load because its
// compiled-against api version is incompatible with the host (a different major, or a
// minor newer than the host's). The caller surfaces Reason to the user (status bar).
type IncompatibleAddIn struct {
	ID     string // the add-in's id (best effort; its ObkAddInId still resolves)
	Path   string // the shared-library file
	Reason string // why it was refused, naming both versions
}

// LoadDir opens every shared-library add-in in dir and returns the loadable ones
// (ready to register with session.AddIns()) plus any it refused on version grounds.
// A missing dir means "no add-ins installed" and yields (nil, nil, nil). On the first
// hard load failure it returns what loaded so far plus the error, so the caller can
// decide whether a bad add-in is fatal; a version-incompatible add-in is NOT a hard
// failure — it is skipped, logged, and reported in the second return value.
func LoadDir(dir string) ([]*LoadedAddIn, []IncompatibleAddIn, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("addinhost: read add-in dir %q: %w", dir, err)
	}
	var out []*LoadedAddIn
	var skipped []IncompatibleAddIn
	for _, e := range entries {
		if e.IsDir() || !isSharedLib(e.Name()) {
			continue
		}
		if out, skipped, err = classifyLoad(filepath.Join(dir, e.Name()), out, skipped); err != nil {
			return out, skipped, err
		}
	}
	return out, skipped, nil
}

// LoadInstalledTree loads add-ins installed under a per-name tree — dir/<name>/… — where
// each <name> directory holds one extracted bundle (the shared library, possibly nested in a
// subfolder, alongside its manifest and dependencies). This is the layout the in-app
// installer writes to the per-user add-ins directory (#1164), distinct from LoadDir's flat
// directory of bare libraries. It loads the first shared library found in each add-in's
// directory with the same version handshake as LoadDir; a missing dir is empty, not an error.
func LoadInstalledTree(dir string) ([]*LoadedAddIn, []IncompatibleAddIn, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("addinhost: read installed add-in dir %q: %w", dir, err)
	}
	var out []*LoadedAddIn
	var skipped []IncompatibleAddIn
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lib, found := firstSharedLib(filepath.Join(dir, e.Name()))
		if !found {
			continue
		}
		if out, skipped, err = classifyLoad(lib, out, skipped); err != nil {
			return out, skipped, err
		}
	}
	return out, skipped, nil
}

// classifyLoad loads the library at path and folds the result into out/skipped: a loadable
// add-in is appended to out, a version-incompatible one to skipped, and a hard open failure
// is returned as an error (which the caller may treat as fatal). Shared by LoadDir and
// LoadInstalledTree so both classify a load identically.
func classifyLoad(path string, out []*LoadedAddIn, skipped []IncompatibleAddIn) ([]*LoadedAddIn, []IncompatibleAddIn, error) {
	loaded, skip, err := loadOne(path)
	switch {
	case err != nil:
		return out, skipped, err
	case skip != nil:
		return out, append(skipped, *skip), nil
	default:
		return append(out, loaded), skipped, nil
	}
}

// firstSharedLib returns the path of the first shared library found anywhere under dir, so an
// installed bundle's library is located whether it sits at the add-in's root or in a
// subfolder. ok is false when the subtree contains no shared library.
func firstSharedLib(dir string) (string, bool) {
	var found string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if isSharedLib(d.Name()) {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	return found, found != ""
}

// loadOne opens one shared library and either returns a loadable add-in or, when its
// api version is incompatible, a skip record naming why (the library is closed). A
// hard open failure is returned as an error the caller may treat as fatal.
func loadOne(path string) (*LoadedAddIn, *IncompatibleAddIn, error) {
	l, err := openLibrary(path)
	if err != nil {
		return nil, nil, err
	}
	if reason := compatibilityError(l); reason != nil {
		// Read the id before close: dlclose unmaps the library, so calling any
		// resolved export afterwards is use-after-unmap.
		id := l.id()
		slog.Warn("addinhost: not loading incompatible add-in", "id", id, "path", path, "reason", reason.Error())
		_ = l.close()
		return nil, &IncompatibleAddIn{ID: id, Path: path, Reason: reason.Error()}, nil
	}
	return &LoadedAddIn{lib: l, id: l.id(), path: path}, nil, nil
}

// compatibilityError refuses an add-in whose compiled-against api version is
// incompatible with this host's (ObkAddInApiMajor/ObkAddInApiMinor, see
// include/oblikovati_addin.h); nil means compatible.
func compatibilityError(l addInLib) error {
	major, minor, present := l.apiVersion()
	return checkCompatibility(major, minor, present, api.Major(), api.Minor())
}

// checkCompatibility is the pure version comparison behind compatibilityError, split
// out so it is unit testable without building a real c-shared add-in. The rule: a
// missing version export cannot be verified (refused); a different major is breaking
// (refused); a minor NEWER than the host needs API the host lacks (refused). An
// older-or-equal minor is fine — minor bumps are additive/backward-compatible.
func checkCompatibility(addinMajor, addinMinor int, present bool, hostMajor, hostMinor int) error {
	if !present {
		return fmt.Errorf("add-in does not report its API version (no ObkAddInApiMajor/Minor); cannot verify compatibility with host API %d.%d", hostMajor, hostMinor)
	}
	if addinMajor != hostMajor {
		return fmt.Errorf("add-in built against API major %d, host is API major %d", addinMajor, hostMajor)
	}
	if addinMinor > hostMinor {
		return fmt.Errorf("add-in built against API %d.%d, newer than host API %d.%d", addinMajor, addinMinor, hostMajor, hostMinor)
	}
	return nil
}

// Open loads a single shared-library add-in from path and wraps it for
// registration — the single-file counterpart of LoadDir, used by hot-reload to
// pick up a replaced .so/.dll without restarting the host.
func Open(path string) (*LoadedAddIn, error) {
	l, err := openLibrary(path)
	if err != nil {
		return nil, err
	}
	if reason := compatibilityError(l); reason != nil {
		_ = l.close()
		return nil, fmt.Errorf("addinhost: %s: %w", path, reason)
	}
	return &LoadedAddIn{lib: l, id: l.id(), path: path}, nil
}

// isSharedLib reports whether name has a loadable shared-library extension.
func isSharedLib(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".so", ".dylib", ".dll":
		return true
	default:
		return false
	}
}
