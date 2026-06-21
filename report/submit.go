// SPDX-License-Identifier: GPL-2.0-only

package report

import (
	"context"
	"encoding/json"
	"fmt"

	"oblikovati.org/crcpost"
)

// Submitter POSTs a [Payload] to the reporting endpoint with the CRC-32 authorization token
// the open endpoint expects (see [crcpost]).
type Submitter struct {
	endpoint string
	http     crcpost.Doer
}

// NewSubmitter returns a submitter for endpoint over doer (typically an *http.Client with a
// timeout so a slow network never blocks the caller). An empty endpoint falls back to
// [DefaultEndpoint].
//
// Example:
//
//	sub := report.NewSubmitter("", &http.Client{Timeout: 15 * time.Second})
//	err := sub.Submit(ctx, payload)
func NewSubmitter(endpoint string, doer crcpost.Doer) *Submitter {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Submitter{endpoint: endpoint, http: doer}
}

// Token is the authorization token for a request body: the lowercase, zero-padded hex CRC-32
// (IEEE) of the exact bytes. Exposed for callers that need it; Submit applies it. See
// [crcpost.Token].
func Token(body []byte) string { return crcpost.Token(body) }

// Submit marshals p and POSTs it; a transport failure surfaces as [ErrOffline] so the caller
// can skip silently when offline, and a non-2xx response is returned as an error carrying the
// status and (capped) body.
func (s *Submitter) Submit(ctx context.Context, p Payload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("report: marshal payload: %w", err)
	}
	return crcpost.Send(ctx, s.http, s.endpoint, body)
}
