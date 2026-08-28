// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Completeness of the package's optional capability interfaces (audit I10, #1633).
// Structural typing satisfies an interface silently, so nothing distinguishes "this type
// deliberately does not implement the capability" from "someone forgot to implement it" —
// the drift class that left a spline drag pinning nothing. Each optional capability is
// therefore classified here as TOTAL (a compile-time assertion block: a new type that
// forgets the method is a build break) or PARTIAL (an explicit coverage table the test in
// capability_coverage_test.go enumerates: absence is a recorded decision, not an accident):
//
//	idCarrier    — TOTAL over the restore-provisioned entities (asserted below).
//	pointDefined — PARTIAL: geometry with draggable defining points (coverage table).
//	SmoothCurve  — PARTIAL: Line/Arc/Spline only, sealed (coverage table).
//	CircularCurve— PARTIAL: Circle/Arc only, sealed (coverage table).
//	sourceKinded — TOTAL over the reference sources (asserted in model/compdef).

// idCarrier is TOTAL over the restore-provisioned 2D entities: every entity the sketchRestorer
// pins by saved local id must implement setID, or its constraints lose their operands across
// save/load (#153). A new such entity kind that forgets setID is a build break here, not a
// silent id-collision on reload.
//
// A ProjectedPoint is deliberately NOT an idCarrier: it is not pinned through sketchRestorer.pin but
// rebuilt through RestoreProjectedPoint, which mints its anchor Point (itself an idCarrier) with the
// saved id (#1268). A projected CURVE, by contrast, IS now a concrete reference Line/Circle/Arc
// (ADR-0055 phase 3): it embeds entityBase, so it is a normal idCarrier pinned by the generic restore
// step, and only its owning Projection record is rebuilt from the source descriptor. Writing this
// exception down is the point of the audit (I10, #1633) — the earlier "every entity satisfies it"
// claim quietly overstated the set.
var (
	_ idCarrier = (*Point)(nil)
	_ idCarrier = (*Line)(nil)
	_ idCarrier = (*Circle)(nil)
	_ idCarrier = (*Arc)(nil)
	_ idCarrier = (*Ellipse)(nil)
	_ idCarrier = (*EllipticalArc)(nil)
	_ idCarrier = (*Spline)(nil)
	_ idCarrier = (*SplineHandle)(nil)
	_ idCarrier = (*FixedSpline)(nil)
	_ idCarrier = (*OffsetSpline)(nil)
	_ idCarrier = (*EquationCurve)(nil)
	_ idCarrier = (*BlockInstance)(nil)
	_ idCarrier = (*SketchImage)(nil)
	_ idCarrier = (*TextBox)(nil)
	_ idCarrier = (*FillRegion)(nil)
)
