// SPDX-License-Identifier: GPL-2.0-only

package sketch

// ConstraintKind names what a geometric constraint IS — the stable identifier
// its persistence codec and creation factory are registered under (#1625,
// audit I2). The string values are the .obk recipe vocabulary (ADR-0020) and
// must never change once shipped; they intentionally stay decoupled from the
// wire enum api/types.GeometricConstraintKind, which follows API SemVer and is
// coarser (circularTangent enumerates as the wire "tangent") — the router maps
// between the two at its boundary, exactly as EntityKind does (#1624).
type ConstraintKind string

// The persisted 2D geometric constraint kinds.
const (
	CoincidentKind      ConstraintKind = "coincident"
	HorizontalKind      ConstraintKind = "horizontal"
	VerticalKind        ConstraintKind = "vertical"
	PointOnLineKind     ConstraintKind = "pointOnLine"
	MidpointKind        ConstraintKind = "midpoint"
	PointOnCircleKind   ConstraintKind = "pointOnCircle"
	ParallelKind        ConstraintKind = "parallel"
	PerpendicularKind   ConstraintKind = "perpendicular"
	CollinearKind       ConstraintKind = "collinear"
	EqualLengthKind     ConstraintKind = "equalLength"
	ConcentricKind      ConstraintKind = "concentric"
	EqualRadiusKind     ConstraintKind = "equalRadius"
	CircularTangentKind ConstraintKind = "circularTangent"
	TangentKind         ConstraintKind = "tangent"
	SymmetryKind        ConstraintKind = "symmetry"
	FixKind             ConstraintKind = "fix"
	SmoothKind          ConstraintKind = "smooth"
	GroundKind          ConstraintKind = "ground"
	OffsetKind          ConstraintKind = "offset"
	PatternLinkKind     ConstraintKind = "patternLink"
	TextBoxAnchorKind   ConstraintKind = "textBox"
	CustomKind          ConstraintKind = "custom"
)

// KindedConstraint is the capability the serializer and the router enumerate
// consume: what kind of constraint am I, and which entities do I relate — so
// no consumer re-derives either with a type switch (#1625, audit I2; the
// switch-per-consumer structure shipped #1574's enumerable-but-not-creatable
// Symmetry and keeps the #1416 save-failure class open).
type KindedConstraint interface {
	Constraint
	// ConstraintKind returns the constraint's persisted kind.
	ConstraintKind() ConstraintKind
	// RelatedEntities returns the entities the constraint relates, in the
	// order the API enumerates them.
	RelatedEntities() []Entity
}

// ConstraintKind identifies each 2D constraint for codec and factory dispatch
// (and, mapped at the router boundary, for enumeration). Each concrete type
// declares its own — like Kind() on entities — so no consumer needs a switch.
func (c *CoincidentConstraint) ConstraintKind() ConstraintKind      { return CoincidentKind }
func (c *HorizontalConstraint) ConstraintKind() ConstraintKind      { return HorizontalKind }
func (c *VerticalConstraint) ConstraintKind() ConstraintKind        { return VerticalKind }
func (c *PointOnLineConstraint) ConstraintKind() ConstraintKind     { return PointOnLineKind }
func (c *MidpointConstraint) ConstraintKind() ConstraintKind        { return MidpointKind }
func (c *PointOnCircleConstraint) ConstraintKind() ConstraintKind   { return PointOnCircleKind }
func (c *ParallelConstraint) ConstraintKind() ConstraintKind        { return ParallelKind }
func (c *PerpendicularConstraint) ConstraintKind() ConstraintKind   { return PerpendicularKind }
func (c *CollinearConstraint) ConstraintKind() ConstraintKind       { return CollinearKind }
func (c *EqualLengthConstraint) ConstraintKind() ConstraintKind     { return EqualLengthKind }
func (c *ConcentricConstraint) ConstraintKind() ConstraintKind      { return ConcentricKind }
func (c *EqualRadiusConstraint) ConstraintKind() ConstraintKind     { return EqualRadiusKind }
func (c *CircularTangentConstraint) ConstraintKind() ConstraintKind { return CircularTangentKind }
func (c *TangentConstraint) ConstraintKind() ConstraintKind         { return TangentKind }
func (c *SymmetryConstraint) ConstraintKind() ConstraintKind        { return SymmetryKind }
func (c *FixConstraint) ConstraintKind() ConstraintKind             { return FixKind }
func (c *SmoothConstraint) ConstraintKind() ConstraintKind          { return SmoothKind }
func (c *GroundConstraint) ConstraintKind() ConstraintKind          { return GroundKind }
func (c *OffsetConstraint) ConstraintKind() ConstraintKind          { return OffsetKind }
func (c *PatternConstraint) ConstraintKind() ConstraintKind         { return PatternLinkKind }
func (c *TextBoxAnchorConstraint) ConstraintKind() ConstraintKind   { return TextBoxAnchorKind }
func (c *CustomConstraint) ConstraintKind() ConstraintKind          { return CustomKind }

// RelatedEntities per type, in API enumeration order.
func (c *CoincidentConstraint) RelatedEntities() []Entity    { return []Entity{c.A, c.B} }
func (c *HorizontalConstraint) RelatedEntities() []Entity    { return []Entity{c.A, c.B} }
func (c *VerticalConstraint) RelatedEntities() []Entity      { return []Entity{c.A, c.B} }
func (c *PointOnLineConstraint) RelatedEntities() []Entity   { return []Entity{c.P, c.L} }
func (c *MidpointConstraint) RelatedEntities() []Entity      { return []Entity{c.P, c.L} }
func (c *PointOnCircleConstraint) RelatedEntities() []Entity { return []Entity{c.P, c.C} }
func (c *ParallelConstraint) RelatedEntities() []Entity      { return []Entity{c.L1, c.L2} }
func (c *PerpendicularConstraint) RelatedEntities() []Entity { return []Entity{c.L1, c.L2} }
func (c *CollinearConstraint) RelatedEntities() []Entity     { return []Entity{c.L1, c.L2} }
func (c *EqualLengthConstraint) RelatedEntities() []Entity   { return []Entity{c.L1, c.L2} }
func (c *ConcentricConstraint) RelatedEntities() []Entity    { return []Entity{c.C1, c.C2} }
func (c *EqualRadiusConstraint) RelatedEntities() []Entity   { return []Entity{c.C1, c.C2} }
func (c *CircularTangentConstraint) RelatedEntities() []Entity {
	return []Entity{c.C1, c.C2}
}
func (c *TangentConstraint) RelatedEntities() []Entity  { return []Entity{c.L, c.C} }
func (c *SymmetryConstraint) RelatedEntities() []Entity { return []Entity{c.A, c.B, c.About} }
func (c *FixConstraint) RelatedEntities() []Entity      { return []Entity{c.P} }
func (c *SmoothConstraint) RelatedEntities() []Entity   { return []Entity{c.C1, c.C2} }
func (c *GroundConstraint) RelatedEntities() []Entity {
	pts := c.Points()
	out := make([]Entity, len(pts))
	for i, p := range pts {
		out[i] = p
	}
	return out
}
func (c *OffsetConstraint) RelatedEntities() []Entity  { return []Entity{c.L1, c.L2} }
func (c *PatternConstraint) RelatedEntities() []Entity { return []Entity{c.Seed, c.Member} }
func (c *TextBoxAnchorConstraint) RelatedEntities() []Entity {
	return []Entity{c.Text}
}
func (c *CustomConstraint) RelatedEntities() []Entity { return c.Entities }

// Every 2D constraint carries the capability — a lost method fails the build.
var (
	_ KindedConstraint = (*CoincidentConstraint)(nil)
	_ KindedConstraint = (*HorizontalConstraint)(nil)
	_ KindedConstraint = (*VerticalConstraint)(nil)
	_ KindedConstraint = (*PointOnLineConstraint)(nil)
	_ KindedConstraint = (*MidpointConstraint)(nil)
	_ KindedConstraint = (*PointOnCircleConstraint)(nil)
	_ KindedConstraint = (*ParallelConstraint)(nil)
	_ KindedConstraint = (*PerpendicularConstraint)(nil)
	_ KindedConstraint = (*CollinearConstraint)(nil)
	_ KindedConstraint = (*EqualLengthConstraint)(nil)
	_ KindedConstraint = (*ConcentricConstraint)(nil)
	_ KindedConstraint = (*EqualRadiusConstraint)(nil)
	_ KindedConstraint = (*CircularTangentConstraint)(nil)
	_ KindedConstraint = (*TangentConstraint)(nil)
	_ KindedConstraint = (*SymmetryConstraint)(nil)
	_ KindedConstraint = (*FixConstraint)(nil)
	_ KindedConstraint = (*SmoothConstraint)(nil)
	_ KindedConstraint = (*GroundConstraint)(nil)
	_ KindedConstraint = (*OffsetConstraint)(nil)
	_ KindedConstraint = (*PatternConstraint)(nil)
	_ KindedConstraint = (*TextBoxAnchorConstraint)(nil)
	_ KindedConstraint = (*CustomConstraint)(nil)
)
