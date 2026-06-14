// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/options"
	"oblikovati.org/update"
)

func TestUpdateCheckRequestIsOneShot(t *testing.T) {
	s := NewSession()
	if s.TakeUpdateCheckRequest() {
		t.Fatal("no request pending initially")
	}
	s.RequestUpdateCheck()
	if !s.TakeUpdateCheckRequest() {
		t.Fatal("the request should be taken once")
	}
	if s.TakeUpdateCheckRequest() {
		t.Fatal("the request must clear after being taken")
	}
}

func TestShowUpdateResultSetsNoticeWhenAvailable(t *testing.T) {
	s := NewSession()
	s.ShowUpdateResult(update.Result{
		UpdateAvailable: true,
		Latest:          update.Release{Version: "0.0.20260615030000", HTMLURL: "https://gh/r"},
	})
	if s.PendingUpdate() == nil {
		t.Fatal("a result should be pending for the window")
	}
	if s.Notice() == "" {
		t.Error("an available update should drop a status-bar notice")
	}
	s.DismissUpdate()
	if s.PendingUpdate() != nil {
		t.Error("DismissUpdate should clear the window")
	}
}

func TestShowUpdateResultNoNoticeWhenUpToDate(t *testing.T) {
	s := NewSession()
	s.ShowUpdateResult(update.Result{UpdateAvailable: false})
	if s.Notice() != "" {
		t.Errorf("an up-to-date check must not nag the status bar, got %q", s.Notice())
	}
}

func TestOpenLatestReleasePageUsesOpener(t *testing.T) {
	s := NewSession()
	opener := &FakeURLOpener{}
	s.SetURLOpener(opener)
	if err := s.OpenLatestReleasePage(); err == nil {
		t.Fatal("opening with no pending release should fail")
	}
	s.ShowUpdateResult(update.Result{
		UpdateAvailable: true,
		Latest:          update.Release{HTMLURL: "https://gh/release"},
	})
	if err := s.OpenLatestReleasePage(); err != nil {
		t.Fatalf("OpenLatestReleasePage: %v", err)
	}
	if len(opener.opened) != 1 || opener.opened[0] != "https://gh/release" {
		t.Errorf("opened = %v, want the release URL", opener.opened)
	}
}

func TestSetUpdateChecksEnabledPersists(t *testing.T) {
	s := NewSession()
	store := &FakeOptionsStore{stored: options.Defaults()}
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}
	if !s.UpdateChecksEnabled() {
		t.Fatal("default should be enabled")
	}
	if err := s.SetUpdateChecksEnabled(false); err != nil {
		t.Fatalf("SetUpdateChecksEnabled: %v", err)
	}
	if s.UpdateChecksEnabled() {
		t.Error("preference should be off after disabling")
	}
	if store.stored.Updates.CheckOnStartup {
		t.Error("the disabled preference should have been persisted")
	}
}
