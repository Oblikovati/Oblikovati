// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"

	gmath "oblikovati.org/math"
)

// planarInlierFraction is the share of an entity's Z coordinates that must lie in one plane for
// the whole drawing to be imported as a flat 2D sketch rather than a Sketch3D. It is below 1 on
// purpose: real-world drawings (especially georeferenced survey DWGs) carry a handful of
// off-sheet or misdecoded entities whose Z is wildly off-plane — e.g. piracicaba.dwg (#1549),
// where 94% of 86k Z samples sit at exactly Z=0 yet ~50 strays reach ±1.8e7 units. A single
// outlier must not flip a flat 75k-entity street map onto the heavyweight (and incorrect) 3D
// path, so the test asks "is the bulk coplanar" instead of "is every entity coplanar". The
// minority that miss the plane are projected onto it by the 2D importer (their Z is dropped).
const planarInlierFraction = 0.9

// Planar reports whether the BULK of the drawing lies in one Z=constant plane (within tol) and
// returns that elevation. It routes an import to a 2D Sketch (with the returned elevation as the
// plane offset) versus a Sketch3D. The plane is the MEDIAN Z of the entity coordinates — a
// robust estimator (like the importer's median recenter) so a few off-plane outliers neither
// move the plane nor force the 3D path; planarity then holds when at least planarInlierFraction
// of the Z samples fall within tol of that median. An empty drawing is treated as planar at z=0.
func (d *Drawing) Planar(tol float64) (elevation float64, planar bool) {
	zs := d.zSamples()
	if len(zs) == 0 {
		return 0, true
	}
	median := gmath.Median(append([]float64{}, zs...)) // copy: Median sorts in place
	inliers := 0
	for _, z := range zs {
		if math.Abs(z-median) <= tol {
			inliers++
		}
	}
	if float64(inliers) >= planarInlierFraction*float64(len(zs)) {
		return median, true
	}
	return 0, false
}

// zSamples gathers every Z coordinate the planarity test inspects, across all entity types.
func (d *Drawing) zSamples() []float64 {
	var zs []float64
	for _, e := range d.Entities {
		zs = append(zs, entityZ(e)...)
	}
	return zs
}

// entityZ returns the Z coordinates an entity contributes to the planarity test.
//
//nolint:funlen // one-case-per-entity-type dispatch returning each type's Z coordinates.
func entityZ(e Entity) []float64 {
	switch g := e.(type) {
	case *Line:
		return []float64{g.Start[2], g.End[2]}
	case *Circle:
		return []float64{g.Center[2]}
	case *Arc:
		return []float64{g.Center[2]}
	case *Point:
		return []float64{g.Position[2]}
	case *Ellipse:
		return []float64{g.Center[2]}
	case *LwPolyline:
		return []float64{g.Elevation}
	case *Spline:
		zs := make([]float64, 0, len(g.ControlPoints)+len(g.FitPoints))
		for _, c := range g.ControlPoints {
			zs = append(zs, c[2])
		}
		for _, f := range g.FitPoints {
			zs = append(zs, f[2])
		}
		return zs
	default:
		return nil
	}
}
