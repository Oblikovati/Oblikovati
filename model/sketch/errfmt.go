// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Shared error formats for 3D-sketch (de)serialization (used with fmt.Errorf).
const (
	errUnknownEntityRef = "references unknown entity id %d"
	errConstraintWrap   = "%s constraint: %w"
)
