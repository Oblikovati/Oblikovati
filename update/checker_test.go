// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"context"
	"errors"
	"testing"
)

// fakeSource is a named ReleaseSource returning a canned release or error per channel.
type fakeSource struct {
	rel    Release
	err    error
	gotCh  Channel
	called bool
}

func (f *fakeSource) Latest(_ context.Context, ch Channel) (Release, error) {
	f.called, f.gotCh = true, ch
	return f.rel, f.err
}

func TestCheckDevBuildNeverQueries(t *testing.T) {
	src := &fakeSource{}
	res, err := NewChecker(src).Check(context.Background(), "dev")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if src.called {
		t.Error("a dev build must not touch the network")
	}
	if !res.Skipped || res.SkipReason != "development build" {
		t.Errorf("got %+v, want skipped development build", res)
	}
}

func TestCheckOfflineIsGracefulSkip(t *testing.T) {
	src := &fakeSource{err: ErrOffline}
	res, err := NewChecker(src).Check(context.Background(), "0.0.20260614120000")
	if err != nil {
		t.Fatalf("offline must not be an error, got %v", err)
	}
	if !res.Skipped || res.SkipReason != "offline" {
		t.Errorf("got %+v, want skipped offline", res)
	}
}

func TestCheckNoReleaseIsGracefulSkip(t *testing.T) {
	src := &fakeSource{err: ErrNoRelease}
	res, _ := NewChecker(src).Check(context.Background(), "0.0.20260614120000")
	if !res.Skipped || res.SkipReason != "no published release" {
		t.Errorf("got %+v, want skipped no published release", res)
	}
}

func TestCheckUnexpectedErrorPropagates(t *testing.T) {
	boom := errors.New("github 503")
	src := &fakeSource{err: boom}
	if _, err := NewChecker(src).Check(context.Background(), "0.0.20260614120000"); !errors.Is(err, boom) {
		t.Fatalf("want the underlying error, got %v", err)
	}
}

func TestCheckUpdateAvailableUsesChannel(t *testing.T) {
	src := &fakeSource{rel: Release{Version: "0.0.20260615030000-nightly", HTMLURL: "https://x/r"}}
	res, err := NewChecker(src).Check(context.Background(), "0.0.20260614120000-nightly")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if src.gotCh != Nightly {
		t.Errorf("queried channel %v, want Nightly", src.gotCh)
	}
	if !res.UpdateAvailable || res.Latest.HTMLURL != "https://x/r" {
		t.Errorf("got %+v, want update available with the release URL", res)
	}
}

func TestCheckUpToDate(t *testing.T) {
	src := &fakeSource{rel: Release{Version: "0.0.20260614120000"}}
	res, _ := NewChecker(src).Check(context.Background(), "0.0.20260614120000")
	if res.UpdateAvailable || res.Skipped {
		t.Errorf("got %+v, want no update and not skipped", res)
	}
}
