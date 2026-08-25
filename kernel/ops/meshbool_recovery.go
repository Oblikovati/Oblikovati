// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Faceted-curved-join recovery (ADR-0054). The analytic/planar boolean sometimes leaves
// a VALID but FACETED result for a curved join it has no analytic path for — the #2167
// cocylindrical arc-wall union facets both walls into mismatched grids (the visible
// seam). Provenance reconstruction rebuilds those on their exact surfaces. This upgrades
// such a result to the analytic B-rep, but only when reconstruction is a valid solid
// that recovers analytic faces the primary lost at the same volume — so a correct
// analytic primary is never disturbed and a case reconstruction cannot yet close stays
// on the faceted result.

// CodeBooleanAnalyticReconstruction marks a boolean the analytic path faceted and
// provenance reconstruction (ADR-0054) restored to analytic faces. Unlike the mesh
// fallback this is an IMPROVEMENT (a faceted valid solid → an analytic one), recorded so
// the pipeline can see reconstruction fired.
const CodeBooleanAnalyticReconstruction diag.Code = "boolean.analytic-reconstruction"

// recoverFacetedCurvedJoin returns an analytic reconstruction of a valid-but-faceted
// curved boolean, or nil when it does not apply or does not strictly improve the result.
func recoverFacetedCurvedJoin(op PartFeatureOperation, target, tool, primary *topo.Body, rec *diag.Recorder) *topo.Body {
	mop, ok := toMeshboolOp(op)
	if !ok || !facetedCurvedJoin(target, tool, primary) {
		return nil
	}
	if len(target.Faces())+len(tool.Faces()) > csgFallbackFaceLimit {
		return nil // reconstruction runs the exact mesh boolean; gate operand size as the fallbacks do
	}
	recon, ok := reconstructBoolean(target, tool, mop, DefaultQuality(), "boolean")
	if !ok || !validBooleanSolid(recon) {
		return nil
	}
	if analyticFaceCount(recon) <= analyticFaceCount(primary) || !volumesClose(recon, primary, 0.02) {
		return nil
	}
	rec.Recordf(CodeBooleanAnalyticReconstruction, diag.Info,
		"%s: analytic path faceted a curved join; rebuilt %d analytic faces from provenance (#2153)", op, analyticFaceCount(recon))
	return recon
}

// facetedCurvedJoin reports the cheap signature of a curved join the analytic path
// faceted: the operands carry analytic (non-planar) faces the primary result lost, AND
// the primary's face count EXPLODED past the operands' total. The explosion test
// distinguishes faceting (a curved wall becomes dozens of planar facets, #2167's 74
// planes) from a legitimate merge (two coaxial cylinders fuse into ONE clean wall,
// which also drops the curved-face count but keeps the face total small) — recovery
// must not disturb the latter.
func facetedCurvedJoin(target, tool, primary *topo.Body) bool {
	operandCurved := analyticFaceCount(target) + analyticFaceCount(tool)
	if operandCurved == 0 || analyticFaceCount(primary) >= operandCurved {
		return false
	}
	return len(primary.Faces()) > len(target.Faces())+len(tool.Faces())
}

// analyticFaceCount counts a body's non-planar (analytic curved) faces.
func analyticFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			n++
		}
	}
	return n
}

// volumesClose reports whether two bodies enclose the same volume within a relative
// tolerance (the faceted primary under-reports curved volume, so the tolerance covers
// the facet bias — a coarse guard against reconstruction producing wrong geometry).
func volumesClose(a, b *topo.Body, rel float64) bool {
	va := enclosedVolume(a)
	vb := enclosedVolume(b)
	return stdmath.Abs(va-vb) <= rel*stdmath.Max(stdmath.Abs(va), stdmath.Abs(vb))
}

// enclosedVolume is the volume a body encloses, integrated over its tessellation by the
// divergence theorem (positive for a closed, outward-oriented solid).
func enclosedVolume(b *topo.Body) float64 {
	vol := 0.0
	for _, tri := range bodyToSoup(b, DefaultQuality()) {
		p, q, r := tri[0].Round(), tri[1].Round(), tri[2].Round()
		vol += (p.X*(q.Y*r.Z-q.Z*r.Y) + p.Y*(q.Z*r.X-q.X*r.Z) + p.Z*(q.X*r.Y-q.Y*r.X)) / 6
	}
	return vol
}
