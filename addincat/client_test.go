// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// errDoer is an httpDoer whose every request fails — used to assert the offline mapping.
type errDoer struct{}

func (errDoer) Do(*http.Request) (*http.Response, error) { return nil, errors.New("dial tcp: refused") }

func TestListAndFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalogue":
			if r.URL.Query().Get("api") != "0.85" {
				t.Errorf("api param = %q, want 0.85", r.URL.Query().Get("api"))
			}
			_, _ = w.Write([]byte(`{"addins":[{"name":"com.oblikovati.cam","versions":[{"version":"0.6.0","apiMajor":0,"apiMinor":85}]}]}`))
		case "/addins/com.oblikovati.cam":
			_, _ = w.Write([]byte(`{"name":"com.oblikovati.cam","versions":[{"version":"0.6.0"},{"version":"0.5.0"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()
	c := NewClient(ts.URL, ts.Client())

	list, err := c.List(context.Background(), 0, 85, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "com.oblikovati.cam" {
		t.Errorf("List = %+v", list)
	}

	e, err := c.Fetch(context.Background(), "com.oblikovati.cam")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(e.Versions) != 2 {
		t.Errorf("Fetch versions = %d, want 2", len(e.Versions))
	}
}

func TestFetchNotFoundIsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"no such add-in"}`, http.StatusNotFound)
	}))
	defer ts.Close()
	if _, err := NewClient(ts.URL, ts.Client()).Fetch(context.Background(), "nope"); err == nil {
		t.Error("expected an error for 404")
	}
}

func TestTransportErrorIsOffline(t *testing.T) {
	c := NewClient("https://addins.test", errDoer{})
	_, err := c.List(context.Background(), 0, 85, "")
	if !errors.Is(err, ErrOffline) {
		t.Errorf("List error = %v, want ErrOffline", err)
	}
}

func TestNewClientDefaultsBaseURL(t *testing.T) {
	if NewClient("", nil).baseURL != DefaultBaseURL {
		t.Error("empty baseURL should fall back to DefaultBaseURL")
	}
}
