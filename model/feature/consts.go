// SPDX-License-Identifier: GPL-2.0-only

package feature

// Strings shared across the feature package: the bend-part kind id, the work-plane
// lookup error, the tangent-face slot label, and the generic "<context>: <err>" wrap.
const (
	kindBendPart     = "bend-part"
	errNoWorkPlane   = "work geometry: no work plane %q"
	labelTangentFace = "Tangent face"
	errCtxWrap       = "%s: %w"
)
