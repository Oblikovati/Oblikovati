// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/app"
)

// addInLib is the platform-specific handle to a loaded shared-library add-in. The
// Unix (dlopen) and Windows implementations satisfy it; loader.go stays portable.
type addInLib interface {
	id() string
	manifest() string
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

// LoadDir opens every shared-library add-in in dir and returns wrappers ready to
// register with session.AddIns(). A missing dir means "no add-ins installed" and
// yields (nil, nil). On the first failed load it returns what loaded so far plus
// the error, so the caller can decide whether a bad add-in is fatal.
func LoadDir(dir string) ([]*LoadedAddIn, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("addinhost: read add-in dir %q: %w", dir, err)
	}
	var out []*LoadedAddIn
	for _, e := range entries {
		if e.IsDir() || !isSharedLib(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		l, err := openLibrary(path)
		if err != nil {
			return out, err
		}
		out = append(out, &LoadedAddIn{lib: l, id: l.id(), path: path})
	}
	return out, nil
}

// Open loads a single shared-library add-in from path and wraps it for
// registration — the single-file counterpart of LoadDir, used by hot-reload to
// pick up a replaced .so/.dll without restarting the host.
func Open(path string) (*LoadedAddIn, error) {
	l, err := openLibrary(path)
	if err != nil {
		return nil, err
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
