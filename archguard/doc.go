// SPDX-License-Identifier: GPL-2.0-only

// Package archguard holds repository-level architecture invariants that are enforced
// as ordinary Go tests, so a violation fails CI without shipping any code in the
// binary. It intentionally has no production symbols — see the _test.go files for the
// rules (e.g. "/source must never depend on an add-in", ADR-0016/0018).
package archguard
