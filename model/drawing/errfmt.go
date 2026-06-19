// SPDX-License-Identifier: GPL-2.0-only

package drawing

// Shared error formats for view lookups and dimensioning (used with fmt.Errorf).
const (
	errNoViewNamed    = "drawing: no view named %q"
	errViewNoVertices = "drawing: view %q has no model vertices to dimension"
)
