// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeDoer is a named httpDoer returning a canned response/error and recording the URL.
type fakeDoer struct {
	status int
	body   string
	err    error
	gotURL string
	gotHdr string
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.gotURL = req.URL.String()
	f.gotHdr = req.Header.Get("Accept")
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Status:     http.StatusText(f.status),
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func newSource(d *fakeDoer) *GitHubSource {
	s := NewGitHubSource(DefaultOwner, DefaultRepo, d)
	s.apiBase = "https://api.test"
	return s
}

func TestLatestStableParsesTag(t *testing.T) {
	d := &fakeDoer{status: 200, body: `{"tag_name":"v0.000200.1.0","html_url":"https://gh/r1"}`}
	rel, err := newSource(d).Latest(context.Background(), Stable)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.Version != "0.000200.1.0" || rel.HTMLURL != "https://gh/r1" || rel.Channel != Stable {
		t.Errorf("got %+v", rel)
	}
	if !strings.HasSuffix(d.gotURL, "/releases/latest") {
		t.Errorf("stable hit %q, want /releases/latest", d.gotURL)
	}
	if d.gotHdr != "application/vnd.github+json" {
		t.Errorf("Accept header = %q", d.gotHdr)
	}
}

func TestLatestNightlyParsesTitle(t *testing.T) {
	d := &fakeDoer{status: 200, body: `{"tag_name":"nightly","name":"Nightly 0.000200.1.0-nightly.20260615T030000","html_url":"https://gh/n"}`}
	rel, err := newSource(d).Latest(context.Background(), Nightly)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.Version != "0.000200.1.0-nightly.20260615T030000" {
		t.Errorf("nightly version = %q", rel.Version)
	}
	if !strings.HasSuffix(d.gotURL, "/releases/tags/nightly") {
		t.Errorf("nightly hit %q, want /releases/tags/nightly", d.gotURL)
	}
}

func TestLatestTransportErrorIsOffline(t *testing.T) {
	d := &fakeDoer{err: errors.New("dial tcp: no route to host")}
	_, err := newSource(d).Latest(context.Background(), Stable)
	if !errors.Is(err, ErrOffline) {
		t.Fatalf("want ErrOffline, got %v", err)
	}
}

func TestLatest404IsNoRelease(t *testing.T) {
	d := &fakeDoer{status: 404, body: `{"message":"Not Found"}`}
	_, err := newSource(d).Latest(context.Background(), Nightly)
	if !errors.Is(err, ErrNoRelease) {
		t.Fatalf("want ErrNoRelease, got %v", err)
	}
}

func TestLatest500IsError(t *testing.T) {
	d := &fakeDoer{status: 500, body: ``}
	_, err := newSource(d).Latest(context.Background(), Stable)
	if err == nil || errors.Is(err, ErrOffline) || errors.Is(err, ErrNoRelease) {
		t.Fatalf("a 5xx must be a hard error, got %v", err)
	}
}

func TestLatestMissingVersionIsError(t *testing.T) {
	d := &fakeDoer{status: 200, body: `{"tag_name":"nightly","name":""}`}
	if _, err := newSource(d).Latest(context.Background(), Nightly); err == nil {
		t.Fatal("a release with no parseable version must error")
	}
}
