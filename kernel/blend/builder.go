// SPDX-License-Identifier: GPL-2.0-only

package blend

// SegmentSolver is the general marcher (Phase 4): given a spine and a section, it walks the guide
// solving the section at each step and returns the run of blend segments with a localized status.
// It is injected into the Builder so the Phase-3 skeleton carries no marcher yet — the ODE/Newton
// math lands in Phase 4 behind this seam (mirrors OCCT BRepBlend_Walking + BRepBlend_AppSurface).
type SegmentSolver interface {
	March(sp *Spine, sec SectionFunctional) Result
}

// Builder orchestrates a blend along a spine: known analytic parts short-circuit to the closed-form
// catalog (owned by ops, which holds the topo assembly), everything else marches. This is the
// ADR-0050 dispatch seam. Phase 3 lands it with the general branch reporting NotImplemented unless a
// SegmentSolver is wired, so the shipping analytic path — which ops still drives directly — is
// unchanged; Phase 4 supplies the solver and moves the general case onto it.
type Builder struct {
	solver SegmentSolver // Phase 4 marcher; nil in the Phase-3 skeleton
}

// NewBuilder returns a Builder driven by the given general-case marcher (nil until Phase 4).
func NewBuilder(solver SegmentSolver) *Builder { return &Builder{solver: solver} }

// Plan classifies how a spine's blend will be built without building it — the dispatch decision ops
// consults to route a known part to its analytic catalog and everything else to the marcher.
func (b *Builder) Plan(sp *Spine) KnownPartKind { return ClassifyKnownPart(sp) }

// March builds the general (non-known-part) blend through the injected solver, or reports
// NotImplemented when none is wired (the Phase-3 skeleton). Known parts do not reach here — ops
// dispatches them to the closed-form catalog on Plan == known.
func (b *Builder) March(sp *Spine, sec SectionFunctional) Result {
	if b.solver == nil {
		return Result{Status: StatusNotImplemented}
	}
	return b.solver.March(sp, sec)
}
