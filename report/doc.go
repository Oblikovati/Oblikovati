// SPDX-License-Identifier: GPL-2.0-only

// Package report submits a user bug report — a typed diagnostic [Payload] plus two
// screenshots — to the reporting.oblikovati.org endpoint, which queues it and opens a
// GitHub issue. Like the sibling update package, the network I/O lives behind a thin
// [httpDoer] seam so the flow is unit-testable without a live server, and connectivity
// failures map to [ErrOffline] so a missing network is a graceful skip, not a crash.
//
// The endpoint is an open POST, so each request carries an Authorization token equal to
// the CRC-32 (IEEE) of the exact JSON body (see [Token]); the server recomputes it over
// the bytes it received and rejects a mismatch. This is a cheap probe filter, not real
// authentication.
package report
