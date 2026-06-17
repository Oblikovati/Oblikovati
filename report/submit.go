// SPDX-License-Identifier: GPL-2.0-only

package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
)

// httpDoer is the one method of *http.Client this package needs, named so tests can fake
// the transport without a live server (CLAUDE.md: wrap external I/O behind a thin seam).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// maxErrorBody caps how much of a rejection body we read into the error message.
const maxErrorBody = 4 << 10

// Submitter POSTs a [Payload] to the reporting endpoint with the CRC-32 authorization
// token the open endpoint expects.
type Submitter struct {
	endpoint string
	http     httpDoer
}

// NewSubmitter returns a submitter for endpoint over doer (typically an *http.Client with
// a timeout so a slow network never blocks the caller). An empty endpoint falls back to
// [DefaultEndpoint].
//
// Example:
//
//	sub := report.NewSubmitter("", &http.Client{Timeout: 15 * time.Second})
//	err := sub.Submit(ctx, payload)
func NewSubmitter(endpoint string, doer httpDoer) *Submitter {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Submitter{endpoint: endpoint, http: doer}
}

// Token is the authorization token for a request body: the lowercase, zero-padded hex
// CRC-32 (IEEE) of the exact bytes. The reporting service recomputes it over the body it
// receives, so both sides must hash the identical bytes — which is why Submit hashes the
// marshaled body it is about to send.
func Token(body []byte) string {
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE(body))
}

// Submit marshals p, POSTs it as JSON with the CRC token in the Authorization header, and
// maps a transport failure to [ErrOffline] so the caller can skip silently when offline. A
// non-2xx response is returned as an error carrying the status and (capped) body.
func (s *Submitter) Submit(ctx context.Context, p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("report: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("report: build request for %q: %w", s.endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", Token(body))
	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err) // no network / DNS / timeout
	}
	defer resp.Body.Close()
	return checkStatus(resp, s.endpoint)
}

// checkStatus accepts 200/202 and turns any other status into an error with the capped
// response body for context.
func checkStatus(resp *http.Response, endpoint string) error {
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	return fmt.Errorf("report: POST %q rejected: %s: %s", endpoint, resp.Status, bytes.TrimSpace(body))
}
