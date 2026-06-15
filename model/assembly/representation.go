// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// The exported read interfaces let the host (router/head) encode representations without naming
// the unexported concrete types. Design-view adds the override counts and the captured camera
// on top of its contract surface; the other families' contract surfaces already suffice.
type (
	// DesignViewRep is a design-view representation's host-read surface.
	DesignViewRep interface {
		contract.DesignViewRepresentation
		Camera() *CapturedCamera
		HiddenCount() int
		AppearanceCount() int
	}
	// PositionalRep is a positional representation's host-read surface.
	PositionalRep interface {
		contract.PositionalRepresentation
	}
	// LODRep is a level-of-detail representation's host-read surface.
	LODRep interface {
		contract.LevelOfDetailRepresentation
	}
	// ModelStateRep is a model state's host-read surface.
	ModelStateRep interface{ contract.ModelState }
)

// Representations (M12-F04, Oblikovati/Oblikovati#361/#367) are named override layers over an
// immutable base assembly: design-view (visibility/appearance/section/camera), positional
// (constraint/joint value overrides) and level-of-detail (occurrence suppression). Each layer
// keys its overrides by the occurrence's stable name (its path leaf within this assembly) so a
// capture survives recompute and reload. Capture snapshots the current state; activate applies
// the overrides — none of this mutates the base definition. No new solver: positional activate
// reuses the F01–F03 combined solve.

// CapturedCamera is a design-view representation's captured viewpoint, carried by the engine
// for the head to apply on activate (the model does not interpret it).
type CapturedCamera struct {
	Eye, Target  math.Point3
	Up           math.Vector3
	FOV          float64
	Orthographic bool
}

// repBase is the identity shared by every representation: id, name, and active flag.
type repBase struct {
	id     uint64
	name   string
	active bool
}

// ID returns the representation's session id.
func (r *repBase) ID() uint64 { return r.id }

// Name returns the representation's display name.
func (r *repBase) Name() string { return r.name }

// Active reports whether this is the active representation of its family.
func (r *repBase) Active() bool { return r.active }

// designViewRep overrides visibility, appearance, section planes, and the camera.
type designViewRep struct {
	repBase
	hidden     map[string]bool // occurrence name → hidden
	appearance map[string]string
	sections   []types.SectionPlane
	camera     *CapturedCamera
}

// Kind identifies the design-view family.
func (d *designViewRep) Kind() types.RepresentationKind { return types.RepresentationDesignView }

// SectionPlanes returns the representation's section/clipping planes.
func (d *designViewRep) SectionPlanes() []types.SectionPlane { return d.sections }

// Camera returns the captured camera, or nil.
func (d *designViewRep) Camera() *CapturedCamera { return d.camera }

// Appearance returns the occurrence appearance overrides (name → appearance id).
func (d *designViewRep) Appearance() map[string]string { return d.appearance }

// HiddenCount / AppearanceCount report the override counts for the info row.
func (d *designViewRep) HiddenCount() int     { return len(d.hidden) }
func (d *designViewRep) AppearanceCount() int { return len(d.appearance) }

// captureDesignView snapshots the current occurrence visibility (appearance/section/camera are
// set explicitly afterwards) into a new design-view representation.
func captureDesignView(occs *occurrence.Occurrences, camera *CapturedCamera) *designViewRep {
	d := &designViewRep{hidden: map[string]bool{}, appearance: map[string]string{}, camera: camera}
	for _, o := range occs.All() {
		if !o.Visible() {
			d.hidden[o.Name()] = true
		}
	}
	return d
}

// apply sets every occurrence's visibility to this representation's override (default visible).
func (d *designViewRep) apply(occs *occurrence.Occurrences) {
	for _, o := range occs.All() {
		o.SetVisible(!d.hidden[o.Name()])
	}
}

// positionalRep overrides constraint/joint values (and per-occurrence flexibility); activating
// it re-solves the assembly to the alternate position.
type positionalRep struct {
	repBase
	constraintValues map[uint64]float64 // constraint id → override value
	jointValues      map[uint64]float64 // joint id → driven-variable target
	flexible         map[string]bool    // occurrence name → flexible
}

// Kind identifies the positional family.
func (p *positionalRep) Kind() types.RepresentationKind { return types.RepresentationPositional }

// OverrideCount returns the number of value overrides the representation carries.
func (p *positionalRep) OverrideCount() int { return len(p.constraintValues) + len(p.jointValues) }

// capturePositional snapshots the current constraint values into a new positional
// representation (joint driven values and flexibility are set explicitly).
func capturePositional(cs *ConstraintSet) *positionalRep {
	p := &positionalRep{constraintValues: map[uint64]float64{}, jointValues: map[uint64]float64{}, flexible: map[string]bool{}}
	for _, c := range cs.All() {
		p.constraintValues[c.ID()] = c.Value()
	}
	return p
}

// apply writes the representation's constraint value overrides onto the live constraints (the
// caller re-solves, building joint driven-pins from jointValues).
func (p *positionalRep) apply(cs *ConstraintSet) {
	for id, v := range p.constraintValues {
		if c := cs.ByID(id); c != nil {
			c.SetValue(v)
		}
	}
}

// lodRep suppresses occurrences for performance.
type lodRep struct {
	repBase
	suppressed map[string]bool // occurrence name → suppressed
}

// Kind identifies the level-of-detail family.
func (l *lodRep) Kind() types.RepresentationKind { return types.RepresentationLevelOfDetail }

// SuppressedCount returns the number of occurrences the representation suppresses.
func (l *lodRep) SuppressedCount() int { return len(l.suppressed) }

// captureLOD snapshots the current suppression state into a new level-of-detail representation.
func captureLOD(occs *occurrence.Occurrences) *lodRep {
	l := &lodRep{suppressed: map[string]bool{}}
	for _, o := range occs.All() {
		if o.Suppressed() {
			l.suppressed[o.Name()] = true
		}
	}
	return l
}

// apply sets every occurrence's suppression to this representation's override.
func (l *lodRep) apply(occs *occurrence.Occurrences) {
	for _, o := range occs.All() {
		o.SetSuppressed(l.suppressed[o.Name()])
	}
}
