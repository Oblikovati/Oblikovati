// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// The signal for a trim the mesher could not honour (ADR-0061 stage 5).
//
// meshSeamCrossingFace ends at the full-domain grid for a face whose boundary wraps a seam that none of
// the wrapping meshers recognised. That grid is the WHOLE surface: a trimmed face meshed with it comes
// back covering material the face does not have, and — since the trim's own boundary is not in the mesh
// — the neighbouring faces meet nothing along it. A ring meeting a ball is the smallest example: the
// lens patch of the torus is a band wrapping the tube between two section curves, and the grid returned
// the entire torus, 227 mm³ where the face carries 24.
//
// The mesh still ships, because a wrong covering beats a missing face in a viewport, but the ground
// rules do not allow it to ship SILENTLY: "a fallback, approximation, or dropped element is a
// diag.Defect that reaches feature health, the API and the UI". Until a chart-driven mesher covers
// these trims, this is what says so.

// CodeTrimIgnoredFullDomain marks a TRIMMED face meshed with the surface's whole parametric domain,
// because no mesher recognised the shape its boundary makes on that surface. Everything integrated from
// the mesh — the face's area, the body's volume and mass properties — is then the untrimmed surface's,
// not the face's, and the face's own boundary is absent so its neighbours cannot meet it.
//
// It is NOT raised for an untrimmed face: a bare sphere, torus or cylinder side legitimately IS its
// whole domain, and the same grid is exactly right there.
const CodeTrimIgnoredFullDomain diag.Code = "tessellate.trim-ignored-full-domain"

// recordIgnoredTrim flags a face whose trim the full-domain grid discarded. loops is the face's own
// boundary-loop count: zero means the face is the whole surface and the grid is correct.
func recordIgnoredTrim(m *Mesh, s geom.Surface, loops int, refused string) *Mesh {
	if m == nil || loops == 0 {
		return m
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeTrimIgnoredFullDomain,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("a trimmed %T face bounded by %d loop(s) was meshed over the surface's whole "+
			"domain: %s, so the mesh covers material the face does not carry and omits the face's own "+
			"boundary", s, loops, ignoredTrimCause(refused)),
	})
	return m
}

// ignoredTrimCause names WHY the face reached the whole domain: a mesher that recognised it and gave it
// up on its own conditioning says which shape it could not describe, and everything else fell through
// unrecognised. A reader who has to guess between the two cannot act on the report.
func ignoredTrimCause(refused string) string {
	if refused == "" {
		return "no mesher recognised its boundary on this surface"
	}
	return "the mesher that recognised it refused the shape — " + refused
}
