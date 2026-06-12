// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	"sort"
	"strings"

	"oblikovati.org/math"
)

// Sketch blocks (M06-F07, Oblikovati/Oblikovati#622): a BlockDefinition is a
// named, self-contained group of sketch entities owned by the part's sketch
// collection (the component definition's SketchBlocks registry); sketches
// place it as BlockInstance entities, each with its own 2D placement
// transform. Instances read the definition live, so a definition edit updates
// every instance; a definition may nest instances of other definitions
// (cycles are rejected).

// BlockDefinition is one reusable entity group. It owns its geometry — the
// entities and their defining points live here, not in any sketch — in the
// definition's own coordinate frame.
type BlockDefinition struct {
	id        ID
	name      string
	ents      []Entity
	pts       []*Point
	instances []*BlockInstance
}

// ID returns the definition's id; Name returns its unique name.
func (d *BlockDefinition) ID() ID       { return d.id }
func (d *BlockDefinition) Name() string { return d.name }

// Add appends an entity to the definition; existing instances reflect it.
// Adding a nested BlockInstance that would (transitively) contain this
// definition again is rejected — the expansion would never terminate.
func (d *BlockDefinition) Add(e Entity) error {
	if inst, ok := e.(*BlockInstance); ok && inst.def.contains(d) {
		return fmt.Errorf("block %q cannot nest %q: that would create a definition cycle",
			d.name, inst.def.name)
	}
	d.ents = append(d.ents, e)
	return nil
}

// contains reports whether the definition is, or transitively nests, target.
func (d *BlockDefinition) contains(target *BlockDefinition) bool {
	if d == target {
		return true
	}
	for _, e := range d.ents {
		if inst, ok := e.(*BlockInstance); ok && inst.def.contains(target) {
			return true
		}
	}
	return false
}

// Entities returns the definition's geometry.
func (d *BlockDefinition) Entities() []Entity {
	out := make([]Entity, len(d.ents))
	copy(out, d.ents)
	return out
}

// EntityCount returns the number of entities in the definition.
func (d *BlockDefinition) EntityCount() int { return len(d.ents) }

// InstanceCount returns how many placed instances reference this definition.
func (d *BlockDefinition) InstanceCount() int { return len(d.instances) }

// attach / detach maintain the placed-instance back-references that make
// delete-in-use detection and consumer naming possible.
func (d *BlockDefinition) attach(inst *BlockInstance) { d.instances = append(d.instances, inst) }
func (d *BlockDefinition) detach(inst *BlockInstance) {
	for i, x := range d.instances {
		if x == inst {
			d.instances = append(d.instances[:i], d.instances[i+1:]...)
			return
		}
	}
}

// consumerNames lists the (deduplicated, sorted) names of sketches holding
// instances — the offenders named when deletion is refused.
func (d *BlockDefinition) consumerNames() []string {
	seen := map[string]bool{}
	for _, inst := range d.instances {
		name := "(definition nesting)"
		if inst.owner != nil {
			name = inst.owner.Name()
		}
		seen[name] = true
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// BlockInstance places a [BlockDefinition] under a 2D transform — into a
// sketch (owner set) or nested inside another definition (owner nil).
type BlockInstance struct {
	entityBase
	def       *BlockDefinition
	owner     *Sketch
	transform math.Matrix3
}

// Definition returns the instanced block definition.
func (b *BlockInstance) Definition() *BlockDefinition { return b.def }

// DefinitionName implements api/contract.SketchBlock.
func (b *BlockInstance) DefinitionName() string { return b.def.name }

// Transform returns the placement transform; SetTransform repositions the
// instance.
func (b *BlockInstance) Transform() math.Matrix3     { return b.transform }
func (b *BlockInstance) SetTransform(t math.Matrix3) { b.transform = t }

// EntityCount returns the live entity count of the underlying definition, so
// it reflects definition edits.
func (b *BlockInstance) EntityCount() int { return b.def.EntityCount() }

// ExpandedPolylines returns the instance's realized geometry: every
// definition entity sampled and mapped through the placement transform,
// recursing into nested instances (their transforms compose).
func (b *BlockInstance) ExpandedPolylines() [][]math.Point2 {
	return expandBlock(b.def, b.transform)
}

// expandBlock samples a definition's entities under the accumulated
// transform. Cycles are impossible by construction ([BlockDefinition.Add]).
func expandBlock(def *BlockDefinition, t math.Matrix3) [][]math.Point2 {
	var out [][]math.Point2
	for _, e := range def.ents {
		if nested, ok := e.(*BlockInstance); ok {
			out = append(out, expandBlock(nested.def, t.Mul(nested.transform))...)
			continue
		}
		poly := blockEntityPolyline(e)
		if len(poly) == 0 {
			continue
		}
		mapped := make([]math.Point2, len(poly))
		for i, p := range poly {
			mapped[i] = t.TransformPoint(p)
		}
		out = append(out, mapped)
	}
	return out
}

// blockEntityPolyline samples one definition entity in definition space.
// Self-closing curves get their closing vertex appended so the drawn loop
// closes visually.
func blockEntityPolyline(e Entity) []math.Point2 {
	switch t := e.(type) {
	case *Point:
		return []math.Point2{t.Position()}
	case *Circle:
		return closeLoop(sampleCircle(t))
	case *Ellipse:
		return closeLoop(sampleEllipseEntity(t))
	case *Spline:
		if t.Closed {
			return closeLoop(sampleSplineEntity(t))
		}
		return sampleSplineEntity(t)
	default:
		return naturalPolyline(e)
	}
}

// closeLoop appends the first vertex so an open polyline draws as a loop.
func closeLoop(pts []math.Point2) []math.Point2 {
	if len(pts) < 2 {
		return pts
	}
	return append(pts, pts[0])
}

// BlockDefinitions is the part-level definition registry — the component
// definition's SketchBlocks collection. It lives on [Sketches] so every
// sketch of the part places from the same set.
type BlockDefinitions struct {
	defs []*BlockDefinition
}

// Define creates a new, empty block definition. Names are the registry key
// and must be unique and non-empty.
func (r *BlockDefinitions) Define(name string) (*BlockDefinition, error) {
	if name == "" {
		return nil, fmt.Errorf("a block definition needs a non-empty name")
	}
	if _, ok := r.ByName(name); ok {
		return nil, fmt.Errorf("block definition %q already exists", name)
	}
	d := &BlockDefinition{id: nextID(), name: name}
	r.defs = append(r.defs, d)
	return d, nil
}

// ByName resolves a definition by its name.
func (r *BlockDefinitions) ByName(name string) (*BlockDefinition, bool) {
	for _, d := range r.defs {
		if d.name == name {
			return d, true
		}
	}
	return nil, false
}

// Count returns the number of definitions; Item returns the i-th.
func (r *BlockDefinitions) Count() int                  { return len(r.defs) }
func (r *BlockDefinitions) Item(i int) *BlockDefinition { return r.defs[i] }

// Delete removes a definition by name. A definition still in use cannot be
// deleted; the error names the block and the consuming sketches.
func (r *BlockDefinitions) Delete(name string) error {
	for i, d := range r.defs {
		if d.name != name {
			continue
		}
		if n := d.InstanceCount(); n > 0 {
			return fmt.Errorf("block %q has %d placed instance(s) in %s; delete those first",
				name, n, strings.Join(d.consumerNames(), ", "))
		}
		r.defs = append(r.defs[:i], r.defs[i+1:]...)
		return nil
	}
	return fmt.Errorf("block definition %q does not exist", name)
}

// Blocks owns a sketch's placed block instances.
type Blocks struct {
	s         *Sketch
	instances []*BlockInstance
}

// Insert places an instance of def at transform t into the sketch.
func (c *Blocks) Insert(def *BlockDefinition, t math.Matrix3) *BlockInstance {
	inst := &BlockInstance{entityBase: newEntity(), def: def, owner: c.s, transform: t}
	def.attach(inst)
	c.s.add(inst)
	c.instances = append(c.instances, inst)
	return inst
}

// InstanceCount reports the sketch's placed instances; Item returns the i-th.
func (c *Blocks) InstanceCount() int          { return len(c.instances) }
func (c *Blocks) Item(i int) *BlockInstance   { return c.instances[i] }
func (c *Blocks) Instances() []*BlockInstance { return append([]*BlockInstance(nil), c.instances...) }
func (c *Blocks) remove(inst *BlockInstance) { // deleteEntity hook
	inst.def.detach(inst)
	c.instances = removeItem(c.instances, inst)
}
