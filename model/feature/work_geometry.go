// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// WorkRef is a stable, serializable reference to a work feature or origin element —
// the way work features (and revolve) name the geometry they are built on, mirroring
// Inventor's reference model (every work feature is defined relative to references,
// not absolute coordinates). Origin elements have well-known refs; user work features
// are referenced by their position in their collection ("plane/2", "axis/0"), which is
// deterministic because they are recreated in creation order on recompute/reopen — the
// same index-as-stable-ref scheme sketches and patterns already use.
type WorkRef string

// Well-known origin references — the static coordinate system every part document has
// from creation (Inventor's "Origin" folder): a center point, the X/Y/Z axes, and the
// XY/XZ/YZ planes. They are grounded coordinate-system elements.
const (
	OriginCenter  WorkRef = "origin/point/center"
	OriginXAxis   WorkRef = "origin/axis/x"
	OriginYAxis   WorkRef = "origin/axis/y"
	OriginZAxis   WorkRef = "origin/axis/z"
	OriginXYPlane WorkRef = "origin/plane/xy"
	OriginXZPlane WorkRef = "origin/plane/xz"
	OriginYZPlane WorkRef = "origin/plane/yz"
)

// workResolver resolves a WorkRef to its current geometry. [WorkGeometry] implements
// it; work-feature definitions resolve their references through it at recompute.
type workResolver interface {
	plane(WorkRef) (sketch.Plane, error)
	axis(WorkRef) (*WorkAxis, error)
	point(WorkRef) (math.Point3, error)
}

// WorkGeometry is a part's construction-geometry frame: the static origin coordinate
// system plus the user-created work planes, axes, points and coordinate systems. It is
// the single owner so references between work features (and from revolve) resolve in
// one place. A part component definition holds exactly one.
type WorkGeometry struct {
	planes  *WorkPlanes
	axes    *WorkAxes
	points  *WorkPoints
	ucs     *UserCoordinateSystems
	userSeq []userEntry // user work features in global creation order (for serialization)
}

// userEntry records a user work feature's collection and index in global creation
// order, so serialization replays them in dependency order — a work feature may
// reference an earlier one (e.g. a point on a user plane).
type userEntry struct {
	collection string
	index      int
}

// recordUser appends a freshly added user work feature to the creation-order log.
func (g *WorkGeometry) recordUser(collection string, index int) {
	g.userSeq = append(g.userSeq, userEntry{collection: collection, index: index})
}

// NewWorkGeometry builds the frame pre-populated with the grounded origin coordinate
// system (center point, X/Y/Z axes, XY/XZ/YZ planes) — the absolute document reference
// frame every other work feature is built relative to.
func NewWorkGeometry() *WorkGeometry {
	g := &WorkGeometry{}
	g.planes = newWorkPlanes(g)
	g.axes = newWorkAxes(g)
	g.points = newWorkPoints(g)
	g.ucs = NewUserCoordinateSystems()
	g.seedOrigin()
	return g
}

// WorkPlanes/WorkAxes/WorkPoints/UserCoordinateSystems expose the collections (origin
// elements first, then user features).
func (g *WorkGeometry) WorkPlanes() *WorkPlanes                       { return g.planes }
func (g *WorkGeometry) WorkAxes() *WorkAxes                           { return g.axes }
func (g *WorkGeometry) WorkPoints() *WorkPoints                       { return g.points }
func (g *WorkGeometry) UserCoordinateSystems() *UserCoordinateSystems { return g.ucs }

// OriginPlanes returns the three origin coordinate-system planes (XY/XZ/YZ) — the
// browser's Origin folder and the default sketch hosts.
func (g *WorkGeometry) OriginPlanes() []*WorkPlane {
	out := make([]*WorkPlane, 0, 3)
	for i := 0; i < g.planes.Count(); i++ {
		if p := g.planes.Item(i); p.coordinateSystem {
			out = append(out, p)
		}
	}
	return out
}

// Recompute re-derives every work feature in order (origin first, grounded; then user
// features, which may reference earlier ones).
func (g *WorkGeometry) Recompute() {
	for i := 0; i < g.points.Count(); i++ {
		g.points.Item(i).recompute(g)
	}
	for i := 0; i < g.axes.Count(); i++ {
		g.axes.Item(i).recompute(g)
	}
	for i := 0; i < g.planes.Count(); i++ {
		g.planes.Item(i).recompute(g)
	}
}

// seedOrigin creates the grounded origin coordinate-system elements with their
// well-known refs. Their geometry is fixed (the document's absolute frame).
func (g *WorkGeometry) seedOrigin() {
	center := math.P3(0, 0, 0)
	g.points.addOrigin(OriginCenter, "Center Point", center)
	g.axes.addOrigin(OriginXAxis, "X Axis", center, mustUnit(1, 0, 0))
	g.axes.addOrigin(OriginYAxis, "Y Axis", center, mustUnit(0, 1, 0))
	g.axes.addOrigin(OriginZAxis, "Z Axis", center, mustUnit(0, 0, 1))
	g.planes.addOrigin(OriginXYPlane, "XY Plane", sketch.XYPlane())
	g.planes.addOrigin(OriginXZPlane, "XZ Plane", sketch.XZPlane())
	g.planes.addOrigin(OriginYZPlane, "YZ Plane", sketch.YZPlane())
}

// plane/axis/point implement workResolver, resolving a ref to its current geometry.
func (g *WorkGeometry) plane(ref WorkRef) (sketch.Plane, error) {
	if i, ok := userIndex(ref, "plane"); ok {
		if i < 0 || i >= g.planes.Count() {
			return sketch.Plane{}, fmt.Errorf("work geometry: no work plane %q", ref)
		}
		return g.planes.Item(i).Plane(), nil
	}
	w, ok := g.planes.byKey[ref]
	if !ok {
		return sketch.Plane{}, fmt.Errorf("work geometry: unknown plane reference %q", ref)
	}
	return w.Plane(), nil
}

func (g *WorkGeometry) axis(ref WorkRef) (*WorkAxis, error) {
	if i, ok := userIndex(ref, "axis"); ok {
		if i < 0 || i >= g.axes.Count() {
			return nil, fmt.Errorf("work geometry: no work axis %q", ref)
		}
		return g.axes.Item(i), nil
	}
	w, ok := g.axes.byKey[ref]
	if !ok {
		return nil, fmt.Errorf("work geometry: unknown axis reference %q", ref)
	}
	return w, nil
}

func (g *WorkGeometry) point(ref WorkRef) (math.Point3, error) {
	if i, ok := userIndex(ref, "point"); ok {
		if i < 0 || i >= g.points.Count() {
			return math.Point3{}, fmt.Errorf("work geometry: no work point %q", ref)
		}
		return g.points.Item(i).Point(), nil
	}
	w, ok := g.points.byKey[ref]
	if !ok {
		return math.Point3{}, fmt.Errorf("work geometry: unknown point reference %q", ref)
	}
	return w.Point(), nil
}

// userIndex parses a "<collection>/<i>" user-feature reference into its index.
func userIndex(ref WorkRef, collection string) (int, bool) {
	prefix := collection + "/"
	s := string(ref)
	if !strings.HasPrefix(s, prefix) {
		return 0, false
	}
	i, err := strconv.Atoi(strings.TrimPrefix(s, prefix))
	if err != nil {
		return 0, false
	}
	return i, true
}

// userRef builds the stable reference for the i-th user feature in a collection.
func userRef(collection string, i int) WorkRef {
	return WorkRef(collection + "/" + strconv.Itoa(i))
}

func mustUnit(x, y, z float64) math.UnitVector3 {
	u, _ := math.NewUnitVector3(x, y, z)
	return u
}
