// SPDX-License-Identifier: GPL-2.0-only

package sketch

// The persisted 3D geometric constraint kinds (#1625, audit I2). 2D and 3D
// constraints share one ConstraintKind vocabulary where the relation is the
// same (coincident, parallel, …); the dimension is carried by which codec
// registry the kind is looked up in, exactly as EntityKind does (#1624).
const (
	Collinear3PointKind ConstraintKind = "collinear"
	EqualKind           ConstraintKind = "equal"
	ParallelToXAxisKind ConstraintKind = "parallelToXAxis"
	ParallelToYAxisKind ConstraintKind = "parallelToYAxis"
	ParallelToZAxisKind ConstraintKind = "parallelToZAxis"
	ParallelToXYKind    ConstraintKind = "parallelToXYPlane"
	ParallelToXZKind    ConstraintKind = "parallelToXZPlane"
	ParallelToYZKind    ConstraintKind = "parallelToYZPlane"
	SplineFitPointsKind ConstraintKind = "splineFitPoints"
	// HelicalJoinKind avoids colliding with the helical EntityKind; the
	// persisted spelling is the same "helical" in the constraint table.
	HelicalJoinKind ConstraintKind = "helical"
	BendKind        ConstraintKind = "bend"
)

// ConstraintKind per 3D type. ParallelToAxis3D/ParallelToPlane3D derive their
// kind from the constrained axis/normal — one type, three persisted kinds each
// (the pre-#1625 axisRowKind/planeRowKind spellings, kept for .obk stability).
func (c *Coincident3D) ConstraintKind() ConstraintKind    { return CoincidentKind }
func (c *Collinear3D) ConstraintKind() ConstraintKind     { return Collinear3PointKind }
func (c *Concentric3D) ConstraintKind() ConstraintKind    { return ConcentricKind }
func (c *Equal3D) ConstraintKind() ConstraintKind         { return EqualKind }
func (c *Parallel3D) ConstraintKind() ConstraintKind      { return ParallelKind }
func (c *Perpendicular3D) ConstraintKind() ConstraintKind { return PerpendicularKind }
func (c *Midpoint3D) ConstraintKind() ConstraintKind      { return MidpointKind }
func (c *Ground3D) ConstraintKind() ConstraintKind        { return GroundKind }
func (c *ParallelToAxis3D) ConstraintKind() ConstraintKind {
	switch {
	case c.Axis.X != 0:
		return ParallelToXAxisKind
	case c.Axis.Y != 0:
		return ParallelToYAxisKind
	default:
		return ParallelToZAxisKind
	}
}
func (c *ParallelToPlane3D) ConstraintKind() ConstraintKind {
	switch {
	case c.Normal.Z != 0:
		return ParallelToXYKind
	case c.Normal.Y != 0:
		return ParallelToXZKind
	default:
		return ParallelToYZKind
	}
}
func (c *Tangent3D) ConstraintKind() ConstraintKind         { return TangentKind }
func (c *Smooth3D) ConstraintKind() ConstraintKind          { return SmoothKind }
func (c *SplineFitPoints3D) ConstraintKind() ConstraintKind { return SplineFitPointsKind }
func (c *Helical3D) ConstraintKind() ConstraintKind         { return HelicalJoinKind }
func (c *Bend3D) ConstraintKind() ConstraintKind            { return BendKind }

// RelatedEntities per 3D type, in API enumeration order.
func (c *Coincident3D) RelatedEntities() []Entity    { return []Entity{c.A, c.B} }
func (c *Collinear3D) RelatedEntities() []Entity     { return []Entity{c.A, c.B, c.C} }
func (c *Concentric3D) RelatedEntities() []Entity    { return []Entity{c.Center1, c.Center2} }
func (c *Equal3D) RelatedEntities() []Entity         { return []Entity{c.E1, c.E2} }
func (c *Parallel3D) RelatedEntities() []Entity      { return []Entity{c.L1, c.L2} }
func (c *Perpendicular3D) RelatedEntities() []Entity { return []Entity{c.L1, c.L2} }
func (c *Midpoint3D) RelatedEntities() []Entity      { return []Entity{c.P, c.L} }
func (c *Ground3D) RelatedEntities() []Entity        { return []Entity{c.P} }
func (c *ParallelToAxis3D) RelatedEntities() []Entity {
	return []Entity{c.L}
}
func (c *ParallelToPlane3D) RelatedEntities() []Entity {
	return []Entity{c.L}
}
func (c *Tangent3D) RelatedEntities() []Entity { return []Entity{c.C1, c.C2} }
func (c *Smooth3D) RelatedEntities() []Entity  { return []Entity{c.C1, c.C2} }
func (c *SplineFitPoints3D) RelatedEntities() []Entity {
	return []Entity{c.Spline, c.P}
}
func (c *Helical3D) RelatedEntities() []Entity { return []Entity{c.H, c.C} }
func (c *Bend3D) RelatedEntities() []Entity {
	return []Entity{c.Arc, c.L1, c.L2}
}

// Every 3D constraint carries the capability — a lost method fails the build.
var (
	_ KindedConstraint = (*Coincident3D)(nil)
	_ KindedConstraint = (*Collinear3D)(nil)
	_ KindedConstraint = (*Concentric3D)(nil)
	_ KindedConstraint = (*Equal3D)(nil)
	_ KindedConstraint = (*Parallel3D)(nil)
	_ KindedConstraint = (*Perpendicular3D)(nil)
	_ KindedConstraint = (*Midpoint3D)(nil)
	_ KindedConstraint = (*Ground3D)(nil)
	_ KindedConstraint = (*ParallelToAxis3D)(nil)
	_ KindedConstraint = (*ParallelToPlane3D)(nil)
	_ KindedConstraint = (*Tangent3D)(nil)
	_ KindedConstraint = (*Smooth3D)(nil)
	_ KindedConstraint = (*SplineFitPoints3D)(nil)
	_ KindedConstraint = (*Helical3D)(nil)
	_ KindedConstraint = (*Bend3D)(nil)
)
