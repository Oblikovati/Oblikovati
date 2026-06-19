// SPDX-License-Identifier: GPL-2.0-only

package param

// Error-message format strings shared across the package's lookups, so the same
// "not found" / "read-only" wording is defined once (used with fmt.Errorf).
const (
	errNoDerivedTable = "param: no derived table with id %d"
	errNoGroup        = "param: no group named %q"
	errNoParameter    = "param: no parameter with id %d"
	errReadOnly       = "param: %s parameter %q is read-only"
)
