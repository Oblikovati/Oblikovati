// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// BlockDefinition is a reusable named group of sketch entities. Instances
// placed from it (see [BlockInstance]) read it live, so editing the
// definition updates every instance.
type BlockDefinition struct {
	id   ID
	name string
	ents []Entity
}

// ID returns the definition's id; Name returns its name.
func (d *BlockDefinition) ID() ID       { return d.id }
func (d *BlockDefinition) Name() string { return d.name }

// Add appends an entity to the definition; existing instances reflect it.
func (d *BlockDefinition) Add(e Entity) { d.ents = append(d.ents, e) }

// Entities returns the definition's geometry.
func (d *BlockDefinition) Entities() []Entity {
	out := make([]Entity, len(d.ents))
	copy(out, d.ents)
	return out
}

// EntityCount returns the number of entities in the definition.
func (d *BlockDefinition) EntityCount() int { return len(d.ents) }

// BlockInstance places a [BlockDefinition] into a sketch under a 2D
// transform. It reads the definition live, so it tracks definition edits.
type BlockInstance struct {
	entityBase
	def       *BlockDefinition
	transform math.Matrix3
}

// Definition returns the instanced block definition.
func (b *BlockInstance) Definition() *BlockDefinition { return b.def }

// Transform returns the placement transform.
func (b *BlockInstance) Transform() math.Matrix3 { return b.transform }

// SetTransform repositions the instance.
func (b *BlockInstance) SetTransform(t math.Matrix3) { b.transform = t }

// EntityCount returns the live entity count of the underlying definition, so it
// reflects definition edits.
func (b *BlockInstance) EntityCount() int { return b.def.EntityCount() }

// Blocks owns a sketch's block definitions and instances.
type Blocks struct {
	s         *Sketch
	defs      []*BlockDefinition
	instances []*BlockInstance
}

// DefineBlock creates a new, empty block definition with the given name.
func (c *Blocks) DefineBlock(name string) *BlockDefinition {
	d := &BlockDefinition{id: nextID(), name: name}
	c.defs = append(c.defs, d)
	return d
}

// Insert places an instance of def at transform t into the sketch.
func (c *Blocks) Insert(def *BlockDefinition, t math.Matrix3) *BlockInstance {
	inst := &BlockInstance{entityBase: newEntity(), def: def, transform: t}
	c.s.add(inst)
	c.instances = append(c.instances, inst)
	return inst
}

// DefinitionCount and InstanceCount report the collection sizes.
func (c *Blocks) DefinitionCount() int { return len(c.defs) }
func (c *Blocks) InstanceCount() int   { return len(c.instances) }
