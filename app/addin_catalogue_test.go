// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"oblikovati.org/addincat"
	"oblikovati.org/api"
)

// fakeCatalogueSource is a named in-memory catalogueSource (CLAUDE.md: external I/O behind a
// fake, not an inline stub). It guards its recorded query because the session calls List on a
// worker goroutine.
type fakeCatalogueSource struct {
	mu      sync.Mutex
	entries []addincat.Entry
	err     error
	query   string
}

func (f *fakeCatalogueSource) List(_ context.Context, _, _ int, query string) ([]addincat.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.query = query
	return f.entries, f.err
}

func (f *fakeCatalogueSource) lastQuery() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.query
}

// fakeCatalogueInstaller records installs/uninstalls and returns a canned status list.
type fakeCatalogueInstaller struct {
	mu          sync.Mutex
	status      []addincat.AddInStatus
	installed   []string
	uninstalled []string
	installErr  error
}

func (f *fakeCatalogueInstaller) Install(_ context.Context, e addincat.Entry, _ addincat.Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.installErr != nil {
		return f.installErr
	}
	f.installed = append(f.installed, e.Name)
	return nil
}

func (f *fakeCatalogueInstaller) Uninstall(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uninstalled = append(f.uninstalled, name)
	return nil
}

func (f *fakeCatalogueInstaller) Status(catalogue []addincat.Entry, _, _ int) ([]addincat.AddInStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.status != nil {
		return f.status, nil
	}
	out := make([]addincat.AddInStatus, len(catalogue))
	for i, e := range catalogue {
		out[i] = addincat.AddInStatus{Entry: e, State: addincat.StateAvailable}
	}
	return out, nil
}

func (f *fakeCatalogueInstaller) installs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.installed...)
}

func (f *fakeCatalogueInstaller) uninstalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.uninstalled...)
}

// camEntry is a catalogue entry with a version for the host's exact API major+minor, so the
// host can install it.
func camEntry() addincat.Entry {
	return addincat.Entry{
		Name: "com.oblikovati.cam", DisplayName: "Oblikovati CAM", Description: "machining",
		Versions: []addincat.Version{{
			Version: "0.6.0", APIMajor: api.Major(), APIMinor: api.Minor(),
			Bundles: map[string]addincat.Bundle{addincat.Platform(): {URL: "u", SHA256: "s"}},
		}},
	}
}

// waitFor polls cond up to a deadline, failing the test if it never holds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestRefreshAddInCataloguePopulatesStatuses(t *testing.T) {
	t.Parallel()
	s := NewSession()
	src := &fakeCatalogueSource{entries: []addincat.Entry{camEntry()}}
	s.SetAddInCatalogue(src, &fakeCatalogueInstaller{})

	s.RefreshAddInCatalogue("cam")
	waitFor(t, func() bool { return len(s.AddInStatuses()) == 1 && !s.AddInCatalogueBusy() })

	if src.lastQuery() != "cam" {
		t.Errorf("query passed through = %q, want cam", src.lastQuery())
	}
	if s.AddInStatuses()[0].Entry.Name != "com.oblikovati.cam" {
		t.Errorf("status = %+v", s.AddInStatuses()[0])
	}
}

func TestRefreshRecordsError(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.SetAddInCatalogue(&fakeCatalogueSource{err: errors.New("offline")}, &fakeCatalogueInstaller{})
	s.RefreshAddInCatalogue("")
	waitFor(t, func() bool { return s.AddInCatalogueError() != "" && !s.AddInCatalogueBusy() })
	if s.AddInCatalogueError() == "" {
		t.Error("expected the offline error to be recorded")
	}
}

func TestInstallAddInRunsAndNotices(t *testing.T) {
	t.Parallel()
	s := NewSession()
	src := &fakeCatalogueSource{entries: []addincat.Entry{camEntry()}}
	inst := &fakeCatalogueInstaller{}
	s.SetAddInCatalogue(src, inst)

	s.RefreshAddInCatalogue("")
	waitFor(t, func() bool { return len(s.AddInStatuses()) == 1 })

	s.InstallAddIn("com.oblikovati.cam")
	waitFor(t, func() bool { return len(inst.installs()) == 1 })
	waitFor(t, func() bool {
		n := s.AddInCatalogueNotice()
		return n != "" && !s.AddInCatalogueBusy()
	})
	if inst.installs()[0] != "com.oblikovati.cam" {
		t.Errorf("installed %v, want com.oblikovati.cam", inst.installs())
	}
}

func TestInstallErrorRecorded(t *testing.T) {
	t.Parallel()
	s := NewSession()
	src := &fakeCatalogueSource{entries: []addincat.Entry{camEntry()}}
	inst := &fakeCatalogueInstaller{installErr: errors.New("checksum mismatch")}
	s.SetAddInCatalogue(src, inst)

	s.RefreshAddInCatalogue("")
	waitFor(t, func() bool { return len(s.AddInStatuses()) == 1 })

	s.InstallAddIn("com.oblikovati.cam")
	waitFor(t, func() bool { return s.AddInCatalogueError() != "" && !s.AddInCatalogueBusy() })
	if s.AddInCatalogueNotice() != "" {
		t.Error("a failed install must not leave a success notice")
	}
}

func TestUninstallAddInRuns(t *testing.T) {
	t.Parallel()
	s := NewSession()
	inst := &fakeCatalogueInstaller{}
	s.SetAddInCatalogue(&fakeCatalogueSource{}, inst)
	s.UninstallAddIn("com.oblikovati.cam")
	waitFor(t, func() bool { return len(inst.uninstalls()) == 1 })
}

func TestCatalogueNoOpWhenDisabled(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.AddInCatalogueEnabled() {
		t.Fatal("catalogue should be disabled before injection")
	}
	s.RefreshAddInCatalogue("") // must not panic or set busy
	if s.AddInCatalogueBusy() {
		t.Error("refresh should be a no-op while disabled")
	}
}

func TestInstallUnknownAddInIsNoOp(t *testing.T) {
	t.Parallel()
	s := NewSession()
	inst := &fakeCatalogueInstaller{}
	s.SetAddInCatalogue(&fakeCatalogueSource{}, inst)
	s.InstallAddIn("com.unknown") // not in any snapshot
	// Give any erroneous goroutine a chance to run, then assert nothing was installed.
	time.Sleep(20 * time.Millisecond)
	if len(inst.installs()) != 0 {
		t.Errorf("installed %v, want none for an unknown add-in", inst.installs())
	}
}
