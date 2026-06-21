// SPDX-License-Identifier: GPL-2.0-only

// Package crcpost is the shared client for Oblikovati's CRC-authorized open ingest endpoints
// (bug reports, usage telemetry). Those services accept an open POST guarded only by a cheap
// probe filter: the Authorization header must equal the CRC-32 (IEEE) of the exact request
// body, which the server recomputes and rejects on mismatch. This package centralizes that
// one-line auth, the JSON POST, and the offline/status handling so report and usagestats do
// not each re-implement it (they differ only in their payload type and endpoint).
package crcpost

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
)

// Doer is the one method of *http.Client this package needs, named so tests can fake the
// transport without a live server (CLAUDE.md: wrap external I/O behind a thin seam).
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ErrOffline reports that the endpoint could not be reached (DNS, connection, or timeout).
// Callers treat it as a graceful skip — being offline is not a failure to shout about. The
// report and usagestats packages re-export this as their own ErrOffline.
var ErrOffline = errors.New("crcpost: endpoint unreachable")

// maxErrorBody caps how much of a rejection body we read into the error message.
const maxErrorBody = 4 << 10

// Token is the authorization token for a request body: the lowercase, zero-padded hex CRC-32
// (IEEE) of the exact bytes. The server recomputes it over the body it receives, so both
// sides must hash identical bytes — which is why Send hashes the exact body it transmits.
func Token(body []byte) string {
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE(body))
}

// Send POSTs body as JSON to endpoint with the CRC token in the Authorization header. A
// transport failure is wrapped as [ErrOffline]; a non-2xx response becomes an error carrying
// the status and (capped) body.
//
// Example:
//
//	err := crcpost.Send(ctx, &http.Client{Timeout: 8 * time.Second}, endpoint, jsonBytes)
func Send(ctx context.Context, doer Doer, endpoint string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("crcpost: build request for %q: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", Token(body))
	resp, err := doer.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err) // no network / DNS / timeout
	}
	defer func() { _ = resp.Body.Close() }()
	return checkStatus(resp, endpoint)
}

// checkStatus accepts 200/202 and turns any other status into an error with the capped
// response body for context.
func checkStatus(resp *http.Response, endpoint string) error {
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	return fmt.Errorf("crcpost: POST %q rejected: %s: %s", endpoint, resp.Status, bytes.TrimSpace(body))
}
