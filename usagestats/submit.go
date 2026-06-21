// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"context"
	"encoding/json"
	"fmt"

	"oblikovati.org/crcpost"
)

// Token is the authorization token for a request body: the lowercase, zero-padded hex CRC-32
// (IEEE) of the exact bytes. Exposed for callers that need to precompute it; Submit applies
// it automatically. See [crcpost.Token].
func Token(body []byte) string { return crcpost.Token(body) }

// Submitter POSTs a [Snapshot] to the telemetry endpoint with the CRC-32 authorization token
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
//	sub := usagestats.NewSubmitter("", &http.Client{Timeout: 8 * time.Second})
//	err := sub.Submit(ctx, snapshot)
func NewSubmitter(endpoint string, doer crcpost.Doer) *Submitter {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Submitter{endpoint: endpoint, http: doer}
}

// Submit marshals s and POSTs it; a transport failure surfaces as [ErrOffline] so the caller
// can skip silently when offline, and a non-2xx response is returned as an error.
func (sub *Submitter) Submit(ctx context.Context, s Snapshot) error {
	body, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("usagestats: marshal snapshot: %w", err)
	}
	return crcpost.Send(ctx, sub.http, sub.endpoint, body)
}
