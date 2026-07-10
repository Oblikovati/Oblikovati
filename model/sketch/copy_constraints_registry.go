// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "fmt"

// Constraint copy carry-over is a registry keyed by [ConstraintKind], mirroring
// the serialization registry in serialize_constraint_registry.go (#1637, audit
// S2). The previous dispatch was a pair of hand-maintained type switches whose
// `default: return false` silently dropped Smooth, Ground, Offset, PatternLink,
// Custom and TextBoxAnchor — the same drift class the codec registry was built
// to kill (#1574, #1625). Now every persisted kind must register either a carry
// function or a documented skip reason; the registry closure test asserts the
// registered set matches the persisted vocabulary exactly, so a new kind is a
// compile-visible decision, never an accidental drop.

// constraintCarrier is one 2D kind's copy decision: either carry (remap the
// constraint's operands through the clone map and re-add it on the target,
// returning false when any operand lies outside the copied set) or a non-empty
// skipReason documenting why the kind cannot travel with a copy.
type constraintCarrier struct {
	carry      func(g *GeometricConstraints, c Constraint, m *cloneMap) bool
	skipReason string
}

// carrier adapts a kind's typed carry function to the registry shape, doing the
// concrete-type assertion the old switch cases did.
func carrier[T Constraint](carry func(g *GeometricConstraints, v T, m *cloneMap) bool) constraintCarrier {
	return constraintCarrier{carry: func(g *GeometricConstraints, c Constraint, m *cloneMap) bool {
		return carry(g, c.(T), m)
	}}
}

// constraintCarriers2D maps every persisted 2D kind to its copy decision.
// Duplicate keys are a compile error; completeness against the persisted
// vocabulary is enforced by TestConstraintCarrierRegistryMatchesVocabulary.
var constraintCarriers2D = map[ConstraintKind]constraintCarrier{
	CoincidentKind:           carrier(carryCoincident),
	HorizontalKind:           carrier(carryHorizontal),
	VerticalKind:             carrier(carryVertical),
	SingleLineHorizontalKind: carrier(carryLineHorizontal),
	SingleLineVerticalKind:   carrier(carryLineVertical),
	PointOnLineKind:          carrier(carryPointOnLine),
	MidpointKind:             carrier(carryMidpoint),
	PointOnCircleKind:        carrier(carryPointOnCircle),
	ParallelKind:             carrier(carryParallel),
	PerpendicularKind:        carrier(carryPerpendicular),
	CollinearKind:            carrier(carryCollinear),
	EqualLengthKind:          carrier(carryEqualLength),
	ConcentricKind:           carrier(carryConcentric),
	EqualRadiusKind:          carrier(carryEqualRadius),
	CircularTangentKind:      carrier(carryCircularTangent),
	TangentKind:              carrier(carryTangent),
	SymmetryKind:             carrier(carrySymmetryKind),
	LineSymmetryKind:         carrier(carryLineSymmetry),
	CircularSymmetryKind:     carrier(carryCircularSymmetry),
	ArcMidpointKind:          carrier(carryArcMidpoint),
	EllipseParallelKind:      carrier(carryEllipseAxis),
	EllipsePerpendicularKind: carrier(carryEllipseAxis),
	EllipseCollinearKind:     carrier(carryEllipseAxis),
	EllipseHorizontalKind:    carrier(carryEllipseAxis),
	EllipseVerticalKind:      carrier(carryEllipseAxis),
	FixKind:                  carrier(carryFixKind),
	SmoothKind:               carrier(carrySmooth),
	GroundKind:               carrier(carryGround),
	OffsetKind:               carrier(carryOffset),
	PatternLinkKind:          carrier(carryPatternLink),
	CustomKind:               carrier(carryCustom),
	// A text-box anchor is auto-created with its TextBox (anchorTextBox) and
	// sketch copy does not clone text boxes (cloneEntity has no TextBox case),
	// so the anchor can never be remapped — the one documented skip (#1637).
	TextBoxAnchorKind: {skipReason: "textBox anchor constraint not copied: sketch copy does not clone text boxes, and the anchor is re-created with its box"},
}

// registeredCarrierKinds2D returns the kinds with a copy decision, for the
// registry closure test.
func registeredCarrierKinds2D() []ConstraintKind {
	out := make([]ConstraintKind, 0, len(constraintCarriers2D))
	for k := range constraintCarriers2D {
		out = append(out, k)
	}
	return out
}

// carryFrom re-creates one source geometric constraint on this (target)
// collection via the clone map. carried reports success; skip is a non-empty
// reason when the kind is a documented uncarryable (or, guarded by the closure
// test, has no registered decision) — never both. A registered kind whose
// operands lie outside the copied set drops silently, matching the external-
// operand rule for every carryable kind (#1083).
func (g *GeometricConstraints) carryFrom(c Constraint, m *cloneMap) (carried bool, skip string) {
	kc, ok := c.(KindedConstraint)
	if !ok {
		return false, fmt.Sprintf("constraint %T not copied: it carries no ConstraintKind (expected a registered 2D kind)", c)
	}
	cc, ok := constraintCarriers2D[kc.ConstraintKind()]
	if !ok {
		return false, fmt.Sprintf("constraint kind %q not copied: no copy decision registered", kc.ConstraintKind())
	}
	if cc.carry == nil {
		return false, cc.skipReason
	}
	return cc.carry(g, c, m), ""
}

// The per-kind carry functions. Each remaps its operand shape through the clone
// map and calls the matching Add* factory, returning false when any operand is
// outside the copied set.

func carryCoincident(g *GeometricConstraints, v *CoincidentConstraint, m *cloneMap) bool {
	return carryPoints(m, v.A, v.B, g.AddCoincident)
}

func carryHorizontal(g *GeometricConstraints, v *HorizontalConstraint, m *cloneMap) bool {
	return carryPoints(m, v.A, v.B, g.AddHorizontal)
}

func carryVertical(g *GeometricConstraints, v *VerticalConstraint, m *cloneMap) bool {
	return carryPoints(m, v.A, v.B, g.AddVertical)
}

// carryLineHorizontal / carryLineVertical remap the single line of a single-line
// horizontal/vertical constraint (#1871).
func carryLineHorizontal(g *GeometricConstraints, v *SingleLineHorizontalConstraint, m *cloneMap) bool {
	l, ok := m.line(v.L)
	if !ok {
		return false
	}
	g.AddLineHorizontal(l)
	return true
}

func carryLineVertical(g *GeometricConstraints, v *SingleLineVerticalConstraint, m *cloneMap) bool {
	l, ok := m.line(v.L)
	if !ok {
		return false
	}
	g.AddLineVertical(l)
	return true
}

func carryPointOnLine(g *GeometricConstraints, v *PointOnLineConstraint, m *cloneMap) bool {
	return carryPointLine(m, v.P, v.L, g.AddPointOnLine)
}

func carryMidpoint(g *GeometricConstraints, v *MidpointConstraint, m *cloneMap) bool {
	return carryPointLine(m, v.P, v.L, g.AddMidpoint)
}

func carryPointOnCircle(g *GeometricConstraints, v *PointOnCircleConstraint, m *cloneMap) bool {
	return carryPointCurve(m, v.P, v.C, g.AddPointOnCircle)
}

func carryParallel(g *GeometricConstraints, v *ParallelConstraint, m *cloneMap) bool {
	return carryLines(m, v.L1, v.L2, g.AddParallel)
}

func carryPerpendicular(g *GeometricConstraints, v *PerpendicularConstraint, m *cloneMap) bool {
	return carryLines(m, v.L1, v.L2, g.AddPerpendicular)
}

func carryCollinear(g *GeometricConstraints, v *CollinearConstraint, m *cloneMap) bool {
	return carryLines(m, v.L1, v.L2, g.AddCollinear)
}

func carryEqualLength(g *GeometricConstraints, v *EqualLengthConstraint, m *cloneMap) bool {
	return carryLines(m, v.L1, v.L2, g.AddEqualLength)
}

func carryConcentric(g *GeometricConstraints, v *ConcentricConstraint, m *cloneMap) bool {
	return carryCurves(m, v.C1, v.C2, g.AddConcentric)
}

func carryEqualRadius(g *GeometricConstraints, v *EqualRadiusConstraint, m *cloneMap) bool {
	return carryCurves(m, v.C1, v.C2, g.AddEqualRadius)
}

func carryCircularTangent(g *GeometricConstraints, v *CircularTangentConstraint, m *cloneMap) bool {
	return carryCurves(m, v.C1, v.C2, g.AddCircularTangent)
}

func carryTangent(g *GeometricConstraints, v *TangentConstraint, m *cloneMap) bool {
	return carryLineCurve(m, v.L, v.C, g.AddTangent)
}

func carrySymmetryKind(g *GeometricConstraints, v *SymmetryConstraint, m *cloneMap) bool {
	return carrySymmetry(m, v, g)
}

// carryLineSymmetry remaps the two mirrored lines and the axis; the clone re-derives the
// endpoint pairing from the copied geometry (#1870).
func carryLineSymmetry(g *GeometricConstraints, v *LineSymmetryConstraint, m *cloneMap) bool {
	l1, ok1 := m.line(v.L1)
	l2, ok2 := m.line(v.L2)
	about, ok3 := m.line(v.About)
	if !ok1 || !ok2 || !ok3 {
		return false
	}
	g.AddLineSymmetry(l1, l2, about)
	return true
}

// carryCircularSymmetry remaps the two mirrored circular curves and the axis (#1870).
func carryCircularSymmetry(g *GeometricConstraints, v *CircularSymmetryConstraint, m *cloneMap) bool {
	c1, ok1 := m.curve(v.C1)
	c2, ok2 := m.curve(v.C2)
	about, ok3 := m.line(v.About)
	if !ok1 || !ok2 || !ok3 {
		return false
	}
	g.AddCircularSymmetry(c1, c2, about)
	return true
}

// carryEllipseAxis remaps both operands of an ellipse-axis relation and re-adds it preserving
// the relation and kind (a world-axis operand of a horizontal/vertical form always remaps) (#1879).
func carryEllipseAxis(g *GeometricConstraints, v *AxisRelationConstraint, m *cloneMap) bool {
	a, ok1 := v.a.remap(m)
	b, ok2 := v.b.remap(m)
	if !ok1 || !ok2 {
		return false
	}
	g.addAxisRelation(a, b, v.relation, v.kind)
	return true
}

// carryArcMidpoint remaps the point and the arc (#1872).
func carryArcMidpoint(g *GeometricConstraints, v *ArcMidpointConstraint, m *cloneMap) bool {
	p, ok1 := m.point(v.P)
	a, ok2 := m.arc(v.A)
	if !ok1 || !ok2 {
		return false
	}
	g.AddMidpointToArc(p, a)
	return true
}

func carryFixKind(g *GeometricConstraints, v *FixConstraint, m *cloneMap) bool {
	return carryFix(m, v.P, g)
}

// carrySmooth remaps the two joined curves and their join endpoints; the clone
// keeps the G2 join, so Class-A continuity survives a copy (#1637).
func carrySmooth(g *GeometricConstraints, v *SmoothConstraint, m *cloneMap) bool {
	c1, ok1 := m.smoothCurve(v.C1)
	c2, ok2 := m.smoothCurve(v.C2)
	p1, ok3 := m.point(v.P1)
	p2, ok4 := m.point(v.P2)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return false
	}
	g.AddSmooth(c1, c2, p1, p2)
	return true
}

// carryGround re-grounds every clone at its own (translated) position —
// AddGroundPoints re-freezes coordinates, so the copy is locked where it landed.
func carryGround(g *GeometricConstraints, v *GroundConstraint, m *cloneMap) bool {
	pts := make([]*Point, 0, len(v.Points()))
	for _, p := range v.Points() {
		np, ok := m.point(p)
		if !ok {
			return false
		}
		pts = append(pts, np)
	}
	g.AddGroundPoints(pts...)
	return true
}

func carryOffset(g *GeometricConstraints, v *OffsetConstraint, m *cloneMap) bool {
	l1, ok1 := m.line(v.L1)
	l2, ok2 := m.line(v.L2)
	if !ok1 || !ok2 {
		return false
	}
	g.AddOffset(l1, l2, v.Dist)
	return true
}

// carryPatternLink clones the link as a frozen offset: a live link's offset
// closure reads the source pattern's parameters, which do not travel with a
// copy — the same degradation the persistence codec applies on save (#1637).
func carryPatternLink(g *GeometricConstraints, v *PatternConstraint, m *cloneMap) bool {
	seed, ok1 := m.point(v.Seed)
	member, ok2 := m.point(v.Member)
	if !ok1 || !ok2 {
		return false
	}
	g.AddPatternLink(seed, member)
	return true
}

// carryCustom remaps the tag's entity set; the owning add-in's ClientID and
// name travel unchanged (the attribute-set payload stays with the source).
func carryCustom(g *GeometricConstraints, v *CustomConstraint, m *cloneMap) bool {
	ents := make([]Entity, 0, len(v.Entities))
	for _, e := range v.Entities {
		ne, ok := m.entities[e]
		if !ok {
			return false
		}
		ents = append(ents, ne)
	}
	_, err := g.AddCustom(v.ClientID, v.Name, ents)
	return err == nil
}
