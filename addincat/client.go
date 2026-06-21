// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// DefaultBaseURL is the public add-in catalogue service. Overridable (NewClient takes the
// URL) so a local instance can be targeted during development and tests.
const DefaultBaseURL = "https://addins.oblikovati.org"

// maxErrorBody caps how much of a rejection body we read into an error message.
const maxErrorBody = 4 << 10

// ErrOffline reports that the catalogue service could not be reached (DNS, connection or
// timeout). Callers treat it as a graceful skip — being offline is not a failure to shout
// about — mirroring report.ErrOffline.
var ErrOffline = errors.New("addincat: catalogue server unreachable")

// httpDoer is the one method of *http.Client this package needs, named so tests can fake the
// transport without a live server (CLAUDE.md: wrap external I/O behind a thin seam).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client queries the catalogue service and downloads bundles.
type Client struct {
	baseURL string
	http    httpDoer
}

// NewClient returns a client for baseURL over doer (typically an *http.Client with a timeout
// so a slow network never blocks the caller). An empty baseURL falls back to DefaultBaseURL.
func NewClient(baseURL string, doer httpDoer) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{baseURL: baseURL, http: doer}
}

// List returns the add-ins the catalogue offers for the host's API major+minor, optionally
// filtered by a free-text query. The service trims each entry to its latest matching version.
func (c *Client) List(ctx context.Context, major, minor int, query string) ([]Entry, error) {
	q := url.Values{}
	q.Set("api", strconv.Itoa(major)+"."+strconv.Itoa(minor))
	if query != "" {
		q.Set("q", query)
	}
	var body struct {
		AddIns []Entry `json:"addins"`
	}
	if err := c.getJSON(ctx, "/catalogue?"+q.Encode(), &body); err != nil {
		return nil, err
	}
	return body.AddIns, nil
}

// Fetch returns one add-in's full record (all versions).
func (c *Client) Fetch(ctx context.Context, name string) (Entry, error) {
	var e Entry
	err := c.getJSON(ctx, "/addins/"+url.PathEscape(name), &e)
	return e, err
}

// Download fetches a bundle's bytes. It does not verify the checksum — the installer does,
// once, after download (see Installer.Install).
func (c *Client) Download(ctx context.Context, b Bundle) ([]byte, error) {
	resp, err := c.get(ctx, b.URL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := okStatus(resp, b.URL); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("addincat: read bundle %q: %w", b.URL, err)
	}
	return data, nil
}

// getJSON GETs path (relative to the base URL) and decodes a JSON response into v.
func (c *Client) getJSON(ctx context.Context, path string, v any) error {
	resp, err := c.get(ctx, c.baseURL+path)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if err := okStatus(resp, path); err != nil {
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("addincat: decode %q response: %w", path, err)
	}
	return nil
}

// get issues a GET, mapping a transport failure to ErrOffline.
func (c *Client) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("addincat: build request for %q: %w", rawURL, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOffline, err)
	}
	return resp, nil
}

// okStatus turns a non-2xx response into an error carrying the status and (capped) body.
func okStatus(resp *http.Response, ref string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	return fmt.Errorf("addincat: GET %q: %s: %s", ref, resp.Status, body)
}
