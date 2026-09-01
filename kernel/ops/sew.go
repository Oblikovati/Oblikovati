// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"strings"

	"oblikovati.org/kernel/geom"
	dset "oblikovati.org/kernel/ops/internal/disjoint"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Tolerant sewing of open shells (M07 PBI-084, Oblikovati/Oblikovati#300):
// boundary edges whose endpoints sit within the sew tolerance of each other —
// a real gap, not an exact coincidence — are pulled onto shared positions and
// welded into one edge. Edges that moved record the gap residual as a healed
// polyline (the M25 SnappedCurve machinery), so both faces tessellate the seam
// identically and the result stays watertight downstream.

// Sew closes an open shell's near-coincident boundary gaps and promotes the
// result to a solid. If gaps remain beyond the tolerance, it reports every
// unsewable boundary edge precisely (id, endpoints, nearest opposite gap)
// instead of returning a partially-sewn body.
//
// Example: quilt, _ := ops.Stitch(imported, 0, true, "import"); solid, err := ops.Sew(quilt, 1e-3)
func Sew(b *topo.Body, tolerance float64) (*topo.Body, error) {
	tol := tolerance
	if tol <= 0 {
		// Model-relative default gap (ADR-0042): generous next to the exact-coincidence
		// stitch grid, but scaled to the body so a sub-µm shell is not sewn shut.
		tol = ResolutionForBody(b).Sew()
	}
	open := BoundaryEdges(b)
	if len(open) == 0 {
		return Stitch([]*topo.Body{b}, 0, false, "sew") // already closed → promote to solid
	}
	body, w := sewWeld(b, boundarySnap(open, tol))
	if remaining := BoundaryEdges(body); len(remaining) > 0 {
		return nil, sewGapError(remaining, tol)
	}
	if r := Validate(body); !r.Valid {
		return nil, fmt.Errorf("sew closed the shell but the result is invalid: %s", strings.Join(r.Issues, "; "))
	}
	sewStampResiduals(w)
	return body, nil
}

// sewWeld re-welds every face of b with boundary endpoints snapped to their
// cluster centers, returning the rebuilt body and the weld (for residuals).
func sewWeld(b *topo.Body, snap *boundaryClusters) (*topo.Body, *weld) {
	w := newWeld(ResolutionForBody(b).Plane()) // model-relative coincidence grid (#1399)
	w.snap = snap.apply
	for _, f := range b.Faces() {
		w.addFace(f)
	}
	return w.build(false, "sew"), w
}

// boundaryClusters maps boundary endpoint positions (keyed on the fine stitch
// grid) to their cluster centroid; non-boundary positions pass through.
type boundaryClusters struct {
	grid    float64
	centers map[vKey]math.Point3
}

// boundarySnap clusters the open edges' endpoints: endpoints within tol of each
// other meld into one cluster whose centroid becomes the sewn position.
func boundarySnap(open []*topo.Edge, tol float64) *boundaryClusters {
	return endpointClusterSnap(boundaryEndpoints(open), tol)
}

// endpointClusterSnap unions endpoints within tol of each other and returns their cluster
// centroids. The spatial hash is built FIRST and candidacy flows through it — the retired
// O(m²) pre-pass compared every endpoint pair before the grid existed (#1607). Cells have
// edge tol, so any pair within tol sits in the same or an adjacent cell (its 3×3×3
// neighbourhood): the union set is exactly the brute scan's, the union-find components are
// therefore identical, and clusterCentroids accumulates members in index order regardless of
// union order — so the sewn positions are bit-identical.
func endpointClusterSnap(pts []math.Point3, tol float64) *boundaryClusters {
	cluster := make([]int, len(pts))
	for i := range cluster {
		cluster[i] = i
	}
	cells := endpointCells(pts, tol)
	for i := range pts {
		unionNearbyEndpoints(cells, pts, i, tol, cluster)
	}
	return clusterCentroids(pts, cluster)
}

// endpointCells hashes each endpoint into its tol-edge cube cell (Ericson §7.1 spatial
// hashing; the sparse map holds only occupied cells).
func endpointCells(pts []math.Point3, tol float64) map[[3]int64][]int32 {
	cells := make(map[[3]int64][]int32, len(pts))
	for i, p := range pts {
		c := endpointCell(p, tol)
		cells[c] = append(cells[c], int32(i))
	}
	return cells
}

func endpointCell(p math.Point3, tol float64) [3]int64 {
	return [3]int64{
		int64(stdmath.Floor(float64(p.X) / tol)),
		int64(stdmath.Floor(float64(p.Y) / tol)),
		int64(stdmath.Floor(float64(p.Z) / tol)),
	}
}

// unionNearbyEndpoints unions endpoint i with every within-tol endpoint j > i found in the
// 27 neighbouring cells (j < i pairs were already unioned when j was the query — the same
// each-pair-once discipline as the retired i<j double loop).
func unionNearbyEndpoints(cells map[[3]int64][]int32, pts []math.Point3, i int, tol float64, cluster []int) {
	c := endpointCell(pts[i], tol)
	for _, d := range sewCellNeighborhood {
		for _, j := range cells[[3]int64{c[0] + d[0], c[1] + d[1], c[2] + d[2]}] {
			if int(j) > i && float64(pts[i].DistanceTo(pts[int(j)])) <= tol {
				dset.Union(cluster, i, int(j))
			}
		}
	}
}

// sewCellNeighborhood is the 3×3×3 cell-offset stencil around an endpoint's own cell.
var sewCellNeighborhood = cubeNeighborhood()

func cubeNeighborhood() [][3]int64 {
	out := make([][3]int64, 0, 27)
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			out = append(out, [3]int64{dx, dy, -1}, [3]int64{dx, dy, 0}, [3]int64{dx, dy, 1})
		}
	}
	return out
}

// boundaryEndpoints collects each open edge's two endpoint positions.
func boundaryEndpoints(open []*topo.Edge) []math.Point3 {
	pts := make([]math.Point3, 0, 2*len(open))
	for _, e := range open {
		pts = append(pts, e.StartVertex().Point(), e.EndVertex().Point())
	}
	return pts
}

// clusterCentroids averages each cluster's members and indexes the centroid by
// every member's fine-grid cell, so apply() can look any member up.
func clusterCentroids(pts []math.Point3, cluster []int) *boundaryClusters {
	sums := map[int]math.Vector3{}
	counts := map[int]int{}
	for i, p := range pts {
		r := dset.Find(cluster, i)
		sums[r] = sums[r].Add(p.AsVector())
		counts[r]++
	}
	bc := &boundaryClusters{grid: ResolutionForPoints(pts).Plane(), centers: map[vKey]math.Point3{}} // model-relative (#1399)
	for i, p := range pts {
		r := dset.Find(cluster, i)
		c := sums[r].Scale(math.Scalar(1 / float64(counts[r]))).AsPoint()
		bc.centers[bc.cell(p)] = c
	}
	return bc
}

func (bc *boundaryClusters) cell(p math.Point3) vKey {
	w := weld{tol: bc.grid}
	return w.vkey(p)
}

// apply snaps a boundary endpoint to its cluster centroid; any other position
// passes through unchanged.
func (bc *boundaryClusters) apply(p math.Point3) math.Point3 {
	if c, ok := bc.centers[bc.cell(p)]; ok {
		return c
	}
	return p
}

// sewStampResiduals records the gap each welded edge absorbed: an edge whose
// endpoints moved gets a healed polyline (its curve samples, linearly tapered
// onto the snapped endpoints) so both adjacent faces tessellate the seam from
// the same points, plus the residual as its tolerance.
func sewStampResiduals(w *weld) {
	for key, e := range w.built {
		curve := w.curves[key]
		s0 := curve.PointAt(domainLo(curve))
		s1 := curve.PointAt(domainHi(curve))
		d0, d1 := alignSnapDeltas(e, s0, s1)
		residual := max(float64(d0.Length()), float64(d1.Length()))
		if residual <= w.tol {
			continue
		}
		e.SetSnappedCurve(orientToEdge(taperedPolyline(curve, d0, d1), e), residual)
	}
}

// orientToEdge flips a polyline running against the edge's start→end direction
// — discretizeEdge hands SnappedCurve to every face verbatim, so it must run in
// edge order even when the welded representative curve ran the other way.
func orientToEdge(poly []math.Point3, e *topo.Edge) []math.Point3 {
	first, last := poly[0], poly[len(poly)-1]
	s := e.StartVertex().Point()
	if first.DistanceTo(s) <= last.DistanceTo(s) {
		return poly
	}
	return probe.ReversedPoints(poly)
}

// alignSnapDeltas matches the curve's natural ends to the welded edge's
// vertices (the canonical edge key may run against the curve) and returns the
// displacement each curve end absorbed.
func alignSnapDeltas(e *topo.Edge, s0, s1 math.Point3) (math.Vector3, math.Vector3) {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	if s0.DistanceTo(a)+s1.DistanceTo(b) <= s0.DistanceTo(b)+s1.DistanceTo(a) {
		return s0.VectorTo(a), s1.VectorTo(b)
	}
	return s0.VectorTo(b), s1.VectorTo(a)
}

// sewSeamSamples is the healed-polyline density for a sewn edge.
const sewSeamSamples = 32

// taperedPolyline samples the curve and blends the endpoint displacements
// linearly across it, landing exactly on the snapped endpoints. A straight
// edge stays two points — its linear taper is still a straight line, so the
// dense samples would only fragment the face tessellation.
func taperedPolyline(c geom.Curve3, d0, d1 math.Vector3) []math.Point3 {
	samples := sewSeamSamples
	if _, straight := c.(geom.LineSegment); straight {
		samples = 1
	}
	lo, hi := c.Domain()
	out := make([]math.Point3, samples+1)
	for i := 0; i <= samples; i++ {
		t := float64(i) / float64(samples)
		p := c.PointAt(lo + (hi-lo)*t)
		shift := d0.Scale(math.Scalar(1 - t)).Add(d1.Scale(math.Scalar(t)))
		out[i] = p.TranslateBy(shift)
	}
	return out
}

func domainLo(c geom.Curve3) float64 { lo, _ := c.Domain(); return lo }
func domainHi(c geom.Curve3) float64 { _, hi := c.Domain(); return hi }

// sewGapError reports every boundary edge that survived the sew, with its
// endpoints and the nearest opposite boundary endpoint distance — the precise
// "could not be stitched" report PBI-084 requires.
func sewGapError(remaining []*topo.Edge, tol float64) error {
	lines := make([]string, len(remaining))
	for i, e := range remaining {
		s, t := e.StartVertex().Point(), e.EndVertex().Point()
		lines[i] = fmt.Sprintf("edge %d [%v → %v] gap %.3g", e.ID(), s, t, nearestGap(e, remaining))
	}
	return fmt.Errorf("sew: %d boundary edges exceed tolerance %g: %s",
		len(remaining), tol, strings.Join(lines, "; "))
}

// nearestGap is the smallest endpoint distance from e to any OTHER boundary
// edge — the gap a larger tolerance would have had to bridge.
func nearestGap(e *topo.Edge, all []*topo.Edge) float64 {
	best := stdmath.Inf(1)
	for _, o := range all {
		if o == e {
			continue
		}
		for _, p := range []math.Point3{e.StartVertex().Point(), e.EndVertex().Point()} {
			for _, q := range []math.Point3{o.StartVertex().Point(), o.EndVertex().Point()} {
				best = min(best, float64(p.DistanceTo(q)))
			}
		}
	}
	return best
}
