// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// Work features are parametric construction geometry — datum planes, axes and points
// defined by relationships to other geometry (Inventor's model: every work feature is
// built relative to references, resolved at recompute). Each holds a typed, serializable
// [planeDefinition]/etc. (not an opaque closure) so it round-trips. The origin elements
// are grounded coordinate-system members with fixed geometry; user features reference
// the origin or earlier features by [WorkRef].

// defaultOriginPlaneSize is the half-extent the origin planes display at.
const defaultOriginPlaneSize = 5.0

// planeDefinition computes a work plane from its references. Concrete kinds capture the
// references + parameters; eval resolves the references through the work resolver.
type planeDefinition interface {
	kindName() string
	refs() []WorkRef
	eval(r workResolver) (sketch.Plane, error)
}

// fixedPlaneDef is a grounded plane (an origin plane): fixed geometry, no references.
type fixedPlaneDef struct{ plane sketch.Plane }

func (d fixedPlaneDef) kindName() string                        { return "fixed" }
func (d fixedPlaneDef) refs() []WorkRef                         { return nil }
func (d fixedPlaneDef) eval(workResolver) (sketch.Plane, error) { return d.plane, nil }

// offsetPlaneDef is a plane parallel to a base plane, offset along its normal by a
// (parametric) distance — Inventor's PlaneAndOffset definition.
type offsetPlaneDef struct {
	base   WorkRef
	offset func() float64
}

func (d offsetPlaneDef) kindName() string { return "plane-offset" }
func (d offsetPlaneDef) refs() []WorkRef  { return []WorkRef{d.base} }
func (d offsetPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	base, err := r.plane(d.base)
	if err != nil {
		return sketch.Plane{}, err
	}
	origin := base.Origin().TranslateBy(base.Normal().AsVector().Scale(d.offset()))
	return sketch.NewPlane(origin, base.XAxis(), base.YAxis())
}

// threePointPlaneDef builds a plane through three referenced points.
type threePointPlaneDef struct{ a, b, c WorkRef }

func (d threePointPlaneDef) kindName() string { return "three-points" }
func (d threePointPlaneDef) refs() []WorkRef  { return []WorkRef{d.a, d.b, d.c} }
func (d threePointPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	a, err := r.point(d.a)
	if err != nil {
		return sketch.Plane{}, err
	}
	b, err := r.point(d.b)
	if err != nil {
		return sketch.Plane{}, err
	}
	c, err := r.point(d.c)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeThroughPoints(a, b, c)
}

// WorkPlane is a datum plane — an origin coordinate-system plane or a user work plane.
type WorkPlane struct {
	id               ID
	key              WorkRef
	name             string
	def              planeDefinition
	plane            sketch.Plane
	health           health.Health
	displaySize      float64
	coordinateSystem bool
	grounded         bool
	visible          bool
}

// ID/Key/Name identify the datum; Health reports its last recompute state.
func (w *WorkPlane) ID() ID                { return w.id }
func (w *WorkPlane) Key() WorkRef          { return w.key }
func (w *WorkPlane) Name() string          { return w.name }
func (w *WorkPlane) SetName(n string)      { w.name = n }
func (w *WorkPlane) Health() health.Health { return w.health }

// Plane returns the current datum plane, also usable directly as a sketch host.
func (w *WorkPlane) Plane() sketch.Plane { return w.plane }

// DisplaySize is the half-edge length of the square the plane is drawn/picked as.
func (w *WorkPlane) DisplaySize() float64 { return w.displaySize }

// IsCoordinateSystemElement reports whether this is one of the origin frame's planes;
// Grounded reports whether its geometry is fixed (the absolute document frame).
func (w *WorkPlane) IsCoordinateSystemElement() bool { return w.coordinateSystem }
func (w *WorkPlane) Grounded() bool                  { return w.grounded }

// Visible reports whether the datum is drawn in the viewport; SetVisible toggles it
// (Inventor's per-work-feature Visibility). User planes are visible by default.
func (w *WorkPlane) Visible() bool     { return w.visible }
func (w *WorkPlane) SetVisible(v bool) { w.visible = v }

// recompute re-derives the plane from its definition, going sick on failure (e.g.
// degenerate three points) rather than producing garbage.
func (w *WorkPlane) recompute(r workResolver) {
	p, err := w.def.eval(r)
	if err != nil {
		w.health = health.Sicken("work plane: " + err.Error())
		return
	}
	w.plane, w.health = p, health.Healthy
}

// WorkPlanes is the part's collection of datum planes (origin first, then user). It
// holds its owning [WorkGeometry] so Add* can resolve references at creation.
type WorkPlanes struct {
	g     *WorkGeometry
	items []*WorkPlane
	byID  map[ID]*WorkPlane
	byKey map[WorkRef]*WorkPlane
}

func newWorkPlanes(g *WorkGeometry) *WorkPlanes {
	return &WorkPlanes{g: g, byID: map[ID]*WorkPlane{}, byKey: map[WorkRef]*WorkPlane{}}
}

// addOrigin adds a grounded coordinate-system plane with a well-known reference.
func (c *WorkPlanes) addOrigin(key WorkRef, name string, plane sketch.Plane) {
	w := &WorkPlane{
		id: nextID(), key: key, name: name, def: fixedPlaneDef{plane: plane},
		displaySize: defaultOriginPlaneSize, coordinateSystem: true, grounded: true,
	}
	c.track(w)
}

// AddByPlaneAndOffset creates a user plane parallel to base, offset by the value the
// closure returns (typically a parameter), so the datum moves when that param changes.
func (c *WorkPlanes) AddByPlaneAndOffset(base WorkRef, offset func() float64) *WorkPlane {
	return c.addUser(offsetPlaneDef{base: base, offset: offset})
}

// AddByThreePoints creates a user plane through three referenced points.
func (c *WorkPlanes) AddByThreePoints(a, b, cc WorkRef) *WorkPlane {
	return c.addUser(threePointPlaneDef{a: a, b: b, c: cc})
}

// addUser adds a user datum plane, keying it by its position so references stay stable,
// and records it in the part's global creation order. All user planes start named
// "WorkPlane"; the app renames them uniquely (e.g. "Work Plane1") on creation.
func (c *WorkPlanes) addUser(def planeDefinition) *WorkPlane {
	w := &WorkPlane{
		id: nextID(), key: userRef("plane", len(c.items)), name: "WorkPlane", def: def,
		displaySize: defaultOriginPlaneSize, visible: true,
	}
	c.track(w)
	c.g.recordUser("plane", len(c.items)-1)
	return w
}

func (c *WorkPlanes) track(w *WorkPlane) {
	w.recompute(c.g)
	c.items = append(c.items, w)
	c.byID[w.id] = w
	c.byKey[w.key] = w
}

// Count/Item index the collection; ByID/ByKey look up.
func (c *WorkPlanes) Count() int            { return len(c.items) }
func (c *WorkPlanes) Item(i int) *WorkPlane { return c.items[i] }
func (c *WorkPlanes) ByID(id ID) (*WorkPlane, bool) {
	w, ok := c.byID[id]
	return w, ok
}

// planeThroughPoints builds an orthonormal sketch plane through a, b, c.
func planeThroughPoints(a, b, cp math.Point3) (sketch.Plane, error) {
	x, err := math.UnitVector3FromVector(a.VectorTo(b))
	if err != nil {
		return sketch.Plane{}, err
	}
	normal, err := math.UnitVector3FromVector(a.VectorTo(b).Cross(a.VectorTo(cp)))
	if err != nil {
		return sketch.Plane{}, err
	}
	y, err := math.UnitVector3FromVector(normal.Cross(x))
	if err != nil {
		return sketch.Plane{}, err
	}
	return sketch.NewPlane(a, x, y)
}
