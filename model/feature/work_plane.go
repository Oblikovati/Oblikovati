// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// Work features are parametric construction geometry — datum planes, axes and
// points defined by relationships (offset, three-point, intersection…) that
// recompute when their driving inputs change and are valid feature/sketch inputs
// wherever no model face exists yet (modeling/01). Each holds an evaluate closure
// capturing its definition; Recompute re-derives the geometry and sets health.

// WorkPlane is a parametric datum plane.
type WorkPlane struct {
	id       ID
	name     string
	evaluate func() (sketch.Plane, error)
	plane    sketch.Plane
	health   health.Health
}

// ID/Name identify the datum; Health reports its last recompute state.
func (w *WorkPlane) ID() ID                { return w.id }
func (w *WorkPlane) Name() string          { return w.name }
func (w *WorkPlane) SetName(n string)      { w.name = n }
func (w *WorkPlane) Health() health.Health { return w.health }

// Plane returns the current datum plane, also usable directly as a sketch plane.
func (w *WorkPlane) Plane() sketch.Plane { return w.plane }

// Recompute re-derives the plane from its definition, going sick on failure (e.g.
// degenerate three points) rather than producing garbage.
func (w *WorkPlane) Recompute() {
	p, err := w.evaluate()
	if err != nil {
		w.health = health.Sicken("work plane: " + err.Error())
		return
	}
	w.plane, w.health = p, health.Healthy
}

// WorkPlanes is the collection of datum planes and their relationship constructors.
type WorkPlanes struct {
	items []*WorkPlane
	byID  map[ID]*WorkPlane
}

// NewWorkPlanes returns an empty collection.
func NewWorkPlanes() *WorkPlanes { return &WorkPlanes{byID: map[ID]*WorkPlane{}} }

// AddByPlaneAndOffset creates a plane parallel to base, offset along its normal by
// the value the offset closure returns (typically a parameter), so the datum moves
// when that parameter changes.
func (c *WorkPlanes) AddByPlaneAndOffset(base sketch.Plane, offset func() float64) *WorkPlane {
	eval := func() (sketch.Plane, error) {
		origin := base.Origin().TranslateBy(base.Normal().AsVector().Scale(offset()))
		return sketch.NewPlane(origin, base.XAxis(), base.YAxis())
	}
	return c.add("WorkPlane", eval)
}

// AddByThreePoints creates a plane through three (parametric) points, with its X
// axis toward the second point.
func (c *WorkPlanes) AddByThreePoints(a, b, cc func() math.Point3) *WorkPlane {
	eval := func() (sketch.Plane, error) {
		return planeThroughPoints(a(), b(), cc())
	}
	return c.add("WorkPlane", eval)
}

func (c *WorkPlanes) add(name string, eval func() (sketch.Plane, error)) *WorkPlane {
	w := &WorkPlane{id: nextID(), name: name, evaluate: eval}
	w.Recompute()
	c.items = append(c.items, w)
	c.byID[w.id] = w
	return w
}

// Count/Item index the collection; ByID looks up.
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
