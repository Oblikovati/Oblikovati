// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"errors"
	"strings"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/addinhost"
)

// TestIncompatibleNotice covers the status-bar message for version-skipped add-ins:
// empty when none were skipped, the id + reason for one, and a count + ids for many.
func TestIncompatibleNotice(t *testing.T) {
	if got := incompatibleNotice(nil); got != "" {
		t.Errorf("no skips: notice = %q, want empty", got)
	}

	one := []addinhost.IncompatibleAddIn{
		{ID: "com.x.bridge", Reason: "add-in built against API 1.5, newer than host API 1.4"},
	}
	got := incompatibleNotice(one)
	for _, want := range []string{"com.x.bridge", "incompatible API version", "newer than host API 1.4"} {
		if !strings.Contains(got, want) {
			t.Errorf("single skip notice %q missing %q", got, want)
		}
	}

	many := []addinhost.IncompatibleAddIn{
		{ID: "com.x.a", Reason: "r1"},
		{ID: "com.x.b", Reason: "r2"},
	}
	got = incompatibleNotice(many)
	for _, want := range []string{"2 add-ins skipped", "com.x.a", "com.x.b"} {
		if !strings.Contains(got, want) {
			t.Errorf("multi skip notice %q missing %q", got, want)
		}
	}
}

// TestRegisterResultsSurfacesSkipAsNotice confirms a version-skipped add-in (no loadable
// libraries) sets the status-bar notice and adds nothing to the loaded set.
func TestRegisterResultsSurfacesSkipAsNotice(t *testing.T) {
	h := &addInHost{}
	session := app.NewSession()
	skipped := []addinhost.IncompatibleAddIn{{ID: "com.x.bridge", Reason: "newer than host"}}

	h.registerResults(session, nil, skipped, errors.New("read error"))

	if !strings.Contains(session.Notice(), "com.x.bridge") {
		t.Errorf("notice = %q, want it to name the skipped add-in", session.Notice())
	}
	if len(h.loaded) != 0 {
		t.Errorf("loaded = %d, want 0 (nothing loadable)", len(h.loaded))
	}
}

// TestLoadInstalledAddInsEmptyDir exercises the per-user install path against an empty
// directory: it must load nothing and leave the loaded set empty, without error.
func TestLoadInstalledAddInsEmptyDir(t *testing.T) {
	t.Setenv("OBK_USER_ADDINS_DIR", t.TempDir())
	h := &addInHost{}
	h.loadInstalledAddIns(app.NewSession())
	if len(h.loaded) != 0 {
		t.Errorf("loaded = %d from an empty user dir, want 0", len(h.loaded))
	}
}

// TestLoadAndRegisterEmptyDir covers the flat-directory load path with no add-ins present.
func TestLoadAndRegisterEmptyDir(t *testing.T) {
	h := &addInHost{}
	h.loadAndRegister(app.NewSession(), t.TempDir())
	if len(h.loaded) != 0 {
		t.Errorf("loaded = %d from an empty dir, want 0", len(h.loaded))
	}
}
