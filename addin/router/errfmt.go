// SPDX-License-Identifier: GPL-2.0-only

package router

// Shared router literals: error-wrap formats, lookup misses, and the undo-transaction
// labels reused across the parameter method group (defined once for consistency).
const (
	errCtxWrap          = "%s: %w"
	errNoTransientBody  = "no transient body with handle %d"
	errNoSketch3DEntity = "sketch3d: no entity with id %d"
	errAddConstraintFmt = "sketch3d.addConstraint: %w"

	labelEditParameters        = "Edit Parameters"
	labelEditParameterGroups   = "Edit Parameter Groups"
	labelEditDerivedParameters = "Edit Derived Parameters"
)
