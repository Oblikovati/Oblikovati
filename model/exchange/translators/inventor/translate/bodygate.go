// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"math"
	"sort"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

// meshEscapeTolerance is how far a rebuilt body may exceed the tessellation's extent before it is
// judged wrong. It absorbs tessellation chord error (the mesh slightly under-cuts a curved face)
// without admitting a real over-build; the smallest genuine failure measured on the ReelToReel
// corpus overshoots by ~1.6x, far outside this band.
const meshEscapeTolerance = 1.02

// gateBodyAgainstMesh drops the rebuilt feature tree when the body it produced does not fit inside
// Inventor's own stored tessellation of the part, and reports what it dropped.
//
// The tessellation is an INDEPENDENT in-file oracle: it is Inventor's render of the true body, not
// something our decoder derived, so it catches a feature we mis-decoded (a wrong extrude depth
// being the common case). CLAUDE.md ranks a wrong solid below an honest partial result — a 26x
// over-built plate corrupts every downstream consumer, whereas dropping to sketch-only is merely
// incomplete — so a body that fails this check is removed rather than shipped.
//
// Containment (not equality) is the test, because the tessellation also renders visible sketches
// and work planes as extra patches, which can only ever INFLATE its box. An inflated oracle makes
// the check more permissive, never falsely strict, so a correct body can't be rejected by it.
// Validated on the 175-part ReelToReel corpus: 16 wrong bodies caught (up to 26.3x), 0 of the
// volume-correct bodies rejected.
//
// That inflation is LARGE on sketch-heavy parts, which bounds how much this gate can ever catch.
// Measured against Inventor's own ComponentDefinition.RangeBox (COM, an oracle outside the file):
// MainFrameSingleHeadBlock's solid is 1.20 x 40.00 x 48.00 cm and BigChunkyPlate's is
// 3.00 x 40.53 x 48.00 — both matching our rebuild EXACTLY — while their stored tessellations
// measure 1.20 x 76.80 x 83.37 and 3.00 x 76.80 x 83.96. So on those parts the two big extents are
// ~1.8x the solid's, and nothing under that escapes. Note also that this only tests a body for
// EXCEEDING the mesh: an under-built body, or a correct-size one of wrong VOLUME, passes untouched
// (BigChunkyPlate ships at 1.051x). Filtering the graphics patches by provenance would give a real
// volume gate — see #18.
func gateBodyAgainstMesh(def *compdef.PartComponentDefinition, d *ipt.Document) []string {
	mesh, ok := meshExtents(d)
	if !ok {
		return nil // no tessellation stored: no oracle, so nothing is claimed either way
	}
	body, ok := bodyExtents(def)
	if !ok {
		return nil
	}
	if axis, over, escaped := escapingAxis(body, mesh); escaped {
		dropAllFeatures(def)
		def.Recompute()
		return []string{fmt.Sprintf(
			"dropped the rebuilt body: it exceeds Inventor's own tessellation by %.1fx on its %s extent "+
				"(body %.2f cm vs mesh %.2f cm) — a wrong solid is worse than none", over, axis, body[axisIndex(axis)], mesh[axisIndex(axis)])}
	}
	return nil
}

// escapingAxis returns the sorted-extent axis on which the body most exceeds the mesh, if any.
// Extents are compared SORTED because a sketch's plane orientation permutes which model axis a
// profile lands on: sorting compares the body's shape to the mesh's shape independently of pose.
func escapingAxis(body, mesh [3]float64) (name string, over float64, escaped bool) {
	names := [3]string{"smallest", "middle", "largest"}
	for i := 2; i >= 0; i-- {
		if mesh[i] > 0 && body[i] > mesh[i]*meshEscapeTolerance {
			return names[i], body[i] / mesh[i], true
		}
	}
	return "", 0, false
}

func axisIndex(name string) int {
	switch name {
	case "smallest":
		return 0
	case "middle":
		return 1
	}
	return 2
}

// meshExtents is the ascending-sorted side lengths of the bounding box of Inventor's stored
// tessellation, in cm.
func meshExtents(d *ipt.Document) ([3]float64, bool) {
	m := ipt.GraphicsMesh(d)
	if len(m.Verts) == 0 {
		return [3]float64{}, false
	}
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, v := range m.Verts {
		for i := 0; i < 3; i++ {
			lo[i], hi[i] = math.Min(lo[i], v[i]), math.Max(hi[i], v[i])
		}
	}
	return sortedSides(lo, hi), true
}

// bodyExtents is the ascending-sorted side lengths of the box spanning EVERY rebuilt body, in cm.
func bodyExtents(def *compdef.PartComponentDefinition) ([3]float64, bool) {
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		return [3]float64{}, false
	}
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, b := range bodies {
		rb := b.RangeBox()
		mn := [3]float64{rb.Min.X, rb.Min.Y, rb.Min.Z}
		mx := [3]float64{rb.Max.X, rb.Max.Y, rb.Max.Z}
		for i := 0; i < 3; i++ {
			lo[i], hi[i] = math.Min(lo[i], mn[i]), math.Max(hi[i], mx[i])
		}
	}
	return sortedSides(lo, hi), true
}

func sortedSides(lo, hi [3]float64) [3]float64 {
	s := []float64{hi[0] - lo[0], hi[1] - lo[1], hi[2] - lo[2]}
	sort.Float64s(s)
	return [3]float64{s[0], s[1], s[2]}
}

// dropAllFeatures removes every feature, leaving the decoded sketches and parameters standing.
// The features exist only to produce the body that was just judged wrong, so none survives it.
func dropAllFeatures(def *compdef.PartComponentDefinition) {
	fs := def.Features()
	for fs.Count() > 0 {
		fs.Remove(fs.Item(fs.Count() - 1).ID())
	}
}
