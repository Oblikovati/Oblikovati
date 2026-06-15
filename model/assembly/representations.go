// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/occurrence"
)

// Representations is an assembly's representation hub (M12-F04): the three override-layer
// families plus the model states that select one of each. It holds the occurrences,
// constraints, and joints so capture can snapshot the live state and activate can apply
// overrides (positional activate re-solves via the shared engine). The host
// (compdef.AssemblyComponentDefinition) owns one per assembly.
type Representations struct {
	occs   *occurrence.Occurrences
	cs     *ConstraintSet
	js     *JointSet
	design []*designViewRep
	pos    []*positionalRep
	lod    []*lodRep
	models []*modelState
	nextID uint64
}

// NewRepresentations builds an empty representation hub over an assembly's occurrences,
// constraints, and joints.
func NewRepresentations(occs *occurrence.Occurrences, cs *ConstraintSet, js *JointSet) *Representations {
	return &Representations{occs: occs, cs: cs, js: js}
}

// id allocates the next representation/model-state session id.
func (r *Representations) id() uint64 { r.nextID++; return r.nextID }

// CaptureDesignView snapshots the current visibility (and the given camera) into a new
// design-view representation named name.
func (r *Representations) CaptureDesignView(name string, camera *CapturedCamera) DesignViewRep {
	d := captureDesignView(r.occs, camera)
	d.repBase = repBase{id: r.id(), name: repName(name, "DesignView", len(r.design)+1)}
	r.design = append(r.design, d)
	return d
}

// CapturePositional snapshots the current constraint values into a new positional representation.
func (r *Representations) CapturePositional(name string) PositionalRep {
	p := capturePositional(r.cs)
	p.repBase = repBase{id: r.id(), name: repName(name, "Position", len(r.pos)+1)}
	r.pos = append(r.pos, p)
	return p
}

// CaptureLOD snapshots the current suppression state into a new level-of-detail representation.
func (r *Representations) CaptureLOD(name string) LODRep {
	l := captureLOD(r.occs)
	l.repBase = repBase{id: r.id(), name: repName(name, "LevelOfDetail", len(r.lod)+1)}
	r.lod = append(r.lod, l)
	return l
}

// ActivateDesignView applies a design-view representation's visibility overrides and marks it
// active (the head applies its appearance/section/camera).
func (r *Representations) ActivateDesignView(id uint64) (DesignViewRep, error) {
	d := r.designByID(id)
	if d == nil {
		return nil, fmt.Errorf("assembly: no design-view representation with id %d", id)
	}
	d.apply(r.occs)
	for _, o := range r.design {
		o.active = o.id == id
	}
	return d, nil
}

// ActivatePositional applies a positional representation's value overrides and re-solves the
// assembly (constraints + joints + the representation's joint driven-pins) to that position.
func (r *Representations) ActivatePositional(id uint64) (PositionalRep, error) {
	p := r.posByID(id)
	if p == nil {
		return nil, fmt.Errorf("assembly: no positional representation with id %d", id)
	}
	p.apply(r.cs)
	solveOver(r.occs, r.positionalRelationships(p), true)
	for _, o := range r.pos {
		o.active = o.id == id
	}
	return p, nil
}

// positionalRelationships is the combined constraint+joint set plus a driven-pin for each of
// the representation's joint value overrides (reusing the F03 drive mechanism).
func (r *Representations) positionalRelationships(p *positionalRep) []relationship {
	rels := combinedRelationships(r.cs, r.js)
	for jointID, value := range p.jointValues {
		joint, ok := r.js.ByID(jointID).(*assemblyJoint)
		if !ok {
			continue
		}
		if resolved, ok := drivableVariable(joint.kind, types.DriveNatural); ok {
			rels = append(rels, &drivenPin{joint: joint, resolved: resolved, value: value})
		}
	}
	return rels
}

// ActivateLOD applies a level-of-detail representation's suppression set and marks it active.
func (r *Representations) ActivateLOD(id uint64) (LODRep, error) {
	l := r.lodByIDInternal(id)
	if l == nil {
		return nil, fmt.Errorf("assembly: no level-of-detail representation with id %d", id)
	}
	l.apply(r.occs)
	for _, o := range r.lod {
		o.active = o.id == id
	}
	return l, nil
}

// SetVisibility overrides an occurrence's visibility within a design-view representation.
func (r *Representations) SetVisibility(repID uint64, occ *occurrence.Occurrence, visible bool) error {
	d := r.designByID(repID)
	if d == nil {
		return fmt.Errorf("assembly: no design-view representation with id %d", repID)
	}
	if visible {
		delete(d.hidden, occ.Name())
	} else {
		d.hidden[occ.Name()] = true
	}
	return nil
}

// SetAppearance overrides (or clears, when appearanceID is empty) an occurrence's appearance.
func (r *Representations) SetAppearance(repID uint64, occ *occurrence.Occurrence, appearanceID string) error {
	d := r.designByID(repID)
	if d == nil {
		return fmt.Errorf("assembly: no design-view representation with id %d", repID)
	}
	if appearanceID == "" {
		delete(d.appearance, occ.Name())
	} else {
		d.appearance[occ.Name()] = appearanceID
	}
	return nil
}

// AddSection adds a section plane to a design-view representation.
func (r *Representations) AddSection(repID uint64, plane types.SectionPlane) error {
	d := r.designByID(repID)
	if d == nil {
		return fmt.Errorf("assembly: no design-view representation with id %d", repID)
	}
	d.sections = append(d.sections, plane)
	return nil
}

// SetPositionalOverride sets a constraint's (isJoint false) or joint's (isJoint true) value
// override within a positional representation.
func (r *Representations) SetPositionalOverride(repID, relationship uint64, isJoint bool, value float64) error {
	p := r.posByID(repID)
	if p == nil {
		return fmt.Errorf("assembly: no positional representation with id %d", repID)
	}
	if isJoint {
		p.jointValues[relationship] = value
	} else {
		p.constraintValues[relationship] = value
	}
	return nil
}

// SetFlexible sets an occurrence's flexibility within a positional representation.
func (r *Representations) SetFlexible(repID uint64, occ *occurrence.Occurrence, flexible bool) error {
	p := r.posByID(repID)
	if p == nil {
		return fmt.Errorf("assembly: no positional representation with id %d", repID)
	}
	p.flexible[occ.Name()] = flexible
	return nil
}

// SetSuppressed overrides an occurrence's suppression within a level-of-detail representation.
func (r *Representations) SetSuppressed(repID uint64, occ *occurrence.Occurrence, suppressed bool) error {
	l := r.lodByIDInternal(repID)
	if l == nil {
		return fmt.Errorf("assembly: no level-of-detail representation with id %d", repID)
	}
	if suppressed {
		l.suppressed[occ.Name()] = true
	} else {
		delete(l.suppressed, occ.Name())
	}
	return nil
}

// DeleteDesignView / DeletePositional / DeleteLOD remove a representation by id, reporting
// whether it was found.
func (r *Representations) DeleteDesignView(id uint64) bool {
	for i, d := range r.design {
		if d.id == id {
			r.design = append(r.design[:i], r.design[i+1:]...)
			return true
		}
	}
	return false
}

func (r *Representations) DeletePositional(id uint64) bool {
	for i, p := range r.pos {
		if p.id == id {
			r.pos = append(r.pos[:i], r.pos[i+1:]...)
			return true
		}
	}
	return false
}

func (r *Representations) DeleteLOD(id uint64) bool {
	for i, l := range r.lod {
		if l.id == id {
			r.lod = append(r.lod[:i], r.lod[i+1:]...)
			return true
		}
	}
	return false
}

// AllDesignViews / AllPositionals / AllLODs return the representations in creation order (the
// host reads these to build the wire info rows).
func (r *Representations) AllDesignViews() []DesignViewRep {
	out := make([]DesignViewRep, len(r.design))
	for i, d := range r.design {
		out[i] = d
	}
	return out
}

func (r *Representations) AllPositionals() []PositionalRep {
	out := make([]PositionalRep, len(r.pos))
	for i, p := range r.pos {
		out[i] = p
	}
	return out
}

func (r *Representations) AllLODs() []LODRep {
	out := make([]LODRep, len(r.lod))
	for i, l := range r.lod {
		out[i] = l
	}
	return out
}

// DesignViewByID / PositionalByID / LODByID look a representation up by id (host read surface),
// or nil. The private *ByID return the concrete type for internal mutation.
func (r *Representations) DesignViewByID(id uint64) DesignViewRep {
	if d := r.designByID(id); d != nil {
		return d
	}
	return nil
}
func (r *Representations) PositionalByID(id uint64) PositionalRep {
	if p := r.posByID(id); p != nil {
		return p
	}
	return nil
}
func (r *Representations) LODByID(id uint64) LODRep {
	if l := r.lodByIDInternal(id); l != nil {
		return l
	}
	return nil
}

func (r *Representations) designByID(id uint64) *designViewRep {
	for _, d := range r.design {
		if d.id == id {
			return d
		}
	}
	return nil
}

func (r *Representations) posByID(id uint64) *positionalRep {
	for _, p := range r.pos {
		if p.id == id {
			return p
		}
	}
	return nil
}

func (r *Representations) lodByIDInternal(id uint64) *lodRep {
	for _, l := range r.lod {
		if l.id == id {
			return l
		}
	}
	return nil
}

// ActiveSelection returns the names of the currently-active representation in each family
// (empty when a family has no active one) — what a new model state captures.
func (r *Representations) ActiveSelection() (designView, positional, levelOfDetail string) {
	for _, d := range r.design {
		if d.active {
			designView = d.name
		}
	}
	for _, p := range r.pos {
		if p.active {
			positional = p.name
		}
	}
	for _, l := range r.lod {
		if l.active {
			levelOfDetail = l.name
		}
	}
	return
}

// repName returns the given name, or a default "Prefix:N" when name is empty.
func repName(name, prefix string, n int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("%s:%d", prefix, n)
}
