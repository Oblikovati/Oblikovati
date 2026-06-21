// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"oblikovati.org/addincat"
	"oblikovati.org/api"
)

// Add-In Catalogue surface (#1164). The session owns the browse/install state and runs every
// catalogue network call and install on a goroutine, never on the frame loop: the UI reads
// the cached, mutex-guarded snapshot each frame, and a worker refreshes it. A freshly
// installed add-in is not loaded in-process (a Go c-shared cannot be hot-loaded), so an
// install/uninstall leaves a "restart to apply" notice rather than activating immediately.

// catalogueOpTimeout bounds a catalogue network call or download so a slow service never
// strands the worker goroutine.
const catalogueOpTimeout = 60 * time.Second

// catalogueSource lists the add-ins offered for a host API version — the query seam, satisfied
// by *addincat.Client and faked in tests (CLAUDE.md: external I/O behind a thin interface).
type catalogueSource interface {
	List(ctx context.Context, major, minor int, query string) ([]addincat.Entry, error)
}

// catalogueInstaller installs, removes and classifies add-ins in the per-user directory —
// satisfied by *addincat.Installer and faked in tests.
type catalogueInstaller interface {
	Install(ctx context.Context, e addincat.Entry, v addincat.Version) error
	Uninstall(name string) error
	Status(catalogue []addincat.Entry, major, minor int) ([]addincat.AddInStatus, error)
}

// addInCatalogue is the session's catalogue state: the injected seams and the last snapshot
// the UI renders, guarded by a mutex because a worker goroutine writes it.
type addInCatalogue struct {
	mu        sync.Mutex
	source    catalogueSource
	installer catalogueInstaller
	statuses  []addincat.AddInStatus
	busy      bool
	errMsg    string
	notice    string
}

// SetAddInCatalogue injects the catalogue query + install seams (the head wires the real
// addincat client/installer; tests pass fakes). It enables the Add-In Catalogue surface.
func (s *Session) SetAddInCatalogue(source catalogueSource, installer catalogueInstaller) {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	s.addInCat.source = source
	s.addInCat.installer = installer
}

// AddInCatalogueEnabled reports whether the catalogue seams have been injected.
func (s *Session) AddInCatalogueEnabled() bool {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	return s.addInCat.source != nil && s.addInCat.installer != nil
}

// RefreshAddInCatalogue re-queries the catalogue for the host's API version and recomputes
// install status on a goroutine. It is a no-op while a catalogue operation is already running
// or before the seams are injected.
func (s *Session) RefreshAddInCatalogue(query string) {
	if !s.beginCatalogueOp() {
		return
	}
	source, installer := s.addInCat.source, s.addInCat.installer
	major, minor := api.Major(), api.Minor()
	go func() {
		statuses, err := refreshCatalogue(source, installer, major, minor, query)
		s.finishCatalogueRefresh(statuses, err)
	}()
}

// InstallAddIn installs (or updates to) the latest catalogue version of the named add-in for
// the host API, on a goroutine. The name must be one shown by the catalogue.
func (s *Session) InstallAddIn(name string) {
	entry, version, ok := s.latestForInstall(name)
	if !ok || !s.beginCatalogueOp() {
		return
	}
	installer := s.addInCat.installer
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), catalogueOpTimeout)
		defer cancel()
		err := installer.Install(ctx, entry, version)
		s.finishCatalogueAction(err, fmt.Sprintf("Installed %s %s — restart Oblikovati to load it", name, version.Version))
	}()
}

// UninstallAddIn removes the named add-in from the per-user directory on a goroutine.
func (s *Session) UninstallAddIn(name string) {
	if !s.beginCatalogueOp() {
		return
	}
	installer := s.addInCat.installer
	go func() {
		err := installer.Uninstall(name)
		s.finishCatalogueAction(err, fmt.Sprintf("Uninstalled %s — restart Oblikovati to remove it", name))
	}()
}

// AddInStatuses returns a copy of the last catalogue snapshot for rendering.
func (s *Session) AddInStatuses() []addincat.AddInStatus {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	return append([]addincat.AddInStatus(nil), s.addInCat.statuses...)
}

// AddInCatalogueBusy reports whether a catalogue operation is in flight (the UI disables
// actions and shows progress while true).
func (s *Session) AddInCatalogueBusy() bool {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	return s.addInCat.busy
}

// AddInCatalogueError returns the last catalogue error message, or "" when the last op
// succeeded.
func (s *Session) AddInCatalogueError() string {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	return s.addInCat.errMsg
}

// AddInCatalogueNotice returns the last install/uninstall notice (e.g. the restart prompt).
func (s *Session) AddInCatalogueNotice() string {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	return s.addInCat.notice
}

// beginCatalogueOp marks a catalogue operation started, returning false when the seams are
// missing or an operation is already running (so concurrent clicks cannot pile up).
func (s *Session) beginCatalogueOp() bool {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	if s.addInCat.source == nil || s.addInCat.installer == nil || s.addInCat.busy {
		return false
	}
	s.addInCat.busy = true
	s.addInCat.errMsg = ""
	return true
}

// latestForInstall resolves the catalogue entry + latest host-compatible version for name
// from the current snapshot.
func (s *Session) latestForInstall(name string) (addincat.Entry, addincat.Version, bool) {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	for _, st := range s.addInCat.statuses {
		if st.Entry.Name == name {
			v, ok := st.Entry.LatestFor(api.Major(), api.Minor())
			return st.Entry, v, ok
		}
	}
	return addincat.Entry{}, addincat.Version{}, false
}

// finishCatalogueRefresh stores a refresh result and clears the busy flag.
func (s *Session) finishCatalogueRefresh(statuses []addincat.AddInStatus, err error) {
	s.addInCat.mu.Lock()
	defer s.addInCat.mu.Unlock()
	s.addInCat.busy = false
	if err != nil {
		s.addInCat.errMsg = err.Error()
		return
	}
	s.addInCat.statuses = statuses
}

// finishCatalogueAction records an install/uninstall outcome: on success it sets the notice
// and re-refreshes so the snapshot reflects the change; on failure it records the error.
func (s *Session) finishCatalogueAction(err error, successNotice string) {
	s.addInCat.mu.Lock()
	s.addInCat.busy = false
	if err != nil {
		s.addInCat.errMsg = err.Error()
		s.addInCat.mu.Unlock()
		return
	}
	s.addInCat.notice = successNotice
	s.addInCat.mu.Unlock()
	s.RefreshAddInCatalogue("")
}

// refreshCatalogue lists the catalogue and computes install status — the worker body, kept
// free of session locking so it is unit-testable on its own.
func refreshCatalogue(source catalogueSource, installer catalogueInstaller, major, minor int, query string) ([]addincat.AddInStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), catalogueOpTimeout)
	defer cancel()
	entries, err := source.List(ctx, major, minor, query)
	if err != nil {
		return nil, err
	}
	return installer.Status(entries, major, minor)
}
