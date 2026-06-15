// SPDX-License-Identifier: GPL-2.0-only

// Package occurrence models component occurrences — the placements of a component (a
// part or a sub-assembly) inside an assembly. One shared component definition can be
// placed many times; each placement is an occurrence carrying its own transform and
// per-instance state. This is the flyweight at the heart of assemblies: memory scales
// with unique components, not with placements, and editing a definition updates every
// occurrence of it. It is the reference API's ComponentOccurrence /
// ComponentOccurrences (M11-F01/F02, #345/#346).
package occurrence

import "oblikovati.org/math"

// Definition is the component definition an occurrence instances — a part or an
// assembly definition (model/compdef). Many occurrences can share one Definition (the
// flyweight): placing a part twice yields two occurrences pointing at the same
// Definition, so editing it updates every instance. The occurrence needs the
// definition's range box to place it; an assembly definition additionally satisfies
// [Composite] so its instances can be navigated into. Kept narrow here so
// model/occurrence does not import model/compdef, which imports this package.
type Definition interface {
	// RangeBox returns the definition's axis-aligned bounding box in its own space.
	RangeBox() math.Box
}

// Composite is a [Definition] that owns its own occurrences — an assembly. An
// occurrence of a Composite has sub-occurrences (the components nested inside it), the
// basis of the assembly nesting model. A part definition is not a Composite. The
// assembly definition satisfies this through its existing Occurrences accessor.
type Composite interface {
	Definition
	// Occurrences returns the components placed inside this composite definition.
	Occurrences() *Occurrences
}

// Occurrence is one placement of a component inside an assembly: the shared
// definition it instances, a transform locating it in the assembly's space, and its
// per-instance state. Created and owned by an [Occurrences] collection; mutating its
// transform or suppression bumps the owner's revision so the assembly's geometry
// version advances (grounding/adaptivity are non-geometric and do not).
type Occurrence struct {
	id         uint64
	name       string
	transform  math.Matrix4
	suppressed bool
	hidden     bool // inverse of visibility (M12-F04 design-view reps); zero ⇒ visible
	grounded   bool
	adaptive   bool
	flexible   bool // sub-assembly solves independently per placement (M12-F06); excl. adaptive
	substitute bool
	definition Definition
	// childOverrides is a flexible occurrence's independent child placement: keyed by child
	// instance name, it overrides the shared sub-assembly definition's default transform so THIS
	// placement positions its components independently (M12-F06). Nil for a rigid occurrence.
	childOverrides map[string]math.Matrix4
	// componentName is the full document name of the component this occurrence instances,
	// recorded when placed from a document so the placement can be restored on reopen
	// (#715). Empty for an in-memory placement made from a bare definition (no file) and
	// for replicated copies, which inherit the seed's definition rather than re-record it.
	componentName string
	owner         *Occurrences
}

// ID returns the occurrence's session id (unique within its owning collection).
func (o *Occurrence) ID() uint64 { return o.id }

// Name returns the occurrence's instance name (e.g. "pin:1").
func (o *Occurrence) Name() string { return o.name }

// Definition returns the shared component definition this occurrence instances.
func (o *Occurrence) Definition() Definition { return o.definition }

// ComponentName returns the full document name of the component this occurrence
// instances, or "" when it was placed from a bare in-memory definition. It is the
// persistent link the assembly recipe records and resolves on reopen (#715).
func (o *Occurrence) ComponentName() string { return o.componentName }

// Transform returns the occurrence's placement in its assembly's space.
func (o *Occurrence) Transform() math.Matrix4 { return o.transform }

// SetTransform repositions the occurrence and advances the owning assembly's version.
// It is the reference API's Transformation set / SetTransformWithoutConstraints:
// constraint adjustment after a move is the solver's job (M12), not this call's.
func (o *Occurrence) SetTransform(m math.Matrix4) {
	previous := o.transform
	o.transform = m
	o.owner.transformed(o, previous)
}

// Suppressed reports whether the occurrence is excluded from the model — it
// contributes no geometry and is skipped by the range box (and, from M12, by
// constraint solving).
func (o *Occurrence) Suppressed() bool { return o.suppressed }

// SetSuppressed sets the occurrence's suppression state, advancing the version only
// when it actually changes (suppression adds/removes geometry).
func (o *Occurrence) SetSuppressed(suppressed bool) {
	if o.suppressed == suppressed {
		return
	}
	o.suppressed = suppressed
	o.owner.suppressed(o)
}

// Grounded reports whether the occurrence is fixed in space (the solver may not move
// it). Grounding is a constraint hint, not geometry, so it does not bump the version.
func (o *Occurrence) Grounded() bool { return o.grounded }

// SetGrounded fixes or releases the occurrence in space.
func (o *Occurrence) SetGrounded(grounded bool) { o.grounded = grounded }

// Visible reports whether the occurrence is drawn. Visibility is a display concern (a
// design-view representation override, M12-F04) — not geometry — so it does not bump the
// version or affect solving. The field is stored inverted (hidden) so the zero-value
// occurrence is visible.
func (o *Occurrence) Visible() bool { return !o.hidden }

// SetVisible shows or hides the occurrence in the render queue.
func (o *Occurrence) SetVisible(visible bool) { o.hidden = !visible }

// Adaptive reports whether the occurrence's underdimensioned geometry may flex to
// satisfy assembly constraints (resolved by the solver from M12).
func (o *Occurrence) Adaptive() bool { return o.adaptive }

// SetAdaptive marks the occurrence adaptive or rigid. Adaptive and flexible are mutually
// exclusive (M12-F06), so enabling adaptive clears flexible.
func (o *Occurrence) SetAdaptive(adaptive bool) {
	if adaptive {
		o.flexible = false
	}
	o.adaptive = adaptive
}

// Flexible reports whether this sub-assembly occurrence solves its components independently per
// placement — so two placements of the same sub-assembly can show their components in different
// positions (M12-F06, the reference API's flexible component). Mutually exclusive with adaptive.
func (o *Occurrence) Flexible() bool { return o.flexible }

// SetFlexible marks a sub-assembly occurrence flexible (or rigid). Only an occurrence that
// instances a sub-assembly (a [Composite] definition) can be flexible — for a leaf part it is a
// no-op. Enabling flexible clears adaptive (the two states are mutually exclusive).
func (o *Occurrence) SetFlexible(flexible bool) {
	if flexible {
		if _, ok := o.definition.(Composite); !ok {
			return
		}
		o.adaptive = false
	}
	o.flexible = flexible
}

// ChildTransform returns the independent transform this flexible occurrence assigns to the
// named child of its sub-assembly definition, and whether one is set (M12-F06). A caller falls
// back to the child's shared default transform when ok is false.
func (o *Occurrence) ChildTransform(childName string) (math.Matrix4, bool) {
	m, ok := o.childOverrides[childName]
	return m, ok
}

// SetChildTransform positions the named child of this flexible occurrence's sub-assembly
// independently of the other placements, bumping the owning assembly's version so the render
// refreshes. It is the per-placement degree of freedom that makes a sub-assembly flexible.
func (o *Occurrence) SetChildTransform(childName string, m math.Matrix4) {
	if o.childOverrides == nil {
		o.childOverrides = map[string]math.Matrix4{}
	}
	o.childOverrides[childName] = m
	o.owner.transformed(o, o.transform) // bump the version (the placement moved a child)
}

// ChildOverrides returns a copy of this occurrence's per-child transform overrides — what the
// recipe persists for a flexible placement (M12-F06).
func (o *Occurrence) ChildOverrides() map[string]math.Matrix4 {
	if len(o.childOverrides) == 0 {
		return nil
	}
	out := make(map[string]math.Matrix4, len(o.childOverrides))
	for k, v := range o.childOverrides {
		out[k] = v
	}
	return out
}

// SetChildOverrides restores the per-child overrides (the recipe-apply path on reopen).
func (o *Occurrence) SetChildOverrides(overrides map[string]math.Matrix4) {
	if len(overrides) == 0 {
		o.childOverrides = nil
		return
	}
	o.childOverrides = make(map[string]math.Matrix4, len(overrides))
	for k, v := range overrides {
		o.childOverrides[k] = v
	}
}

// IsSubstitute reports whether this occurrence is a substitute — a simplified
// representation that stands in for a set of detailed components (the reference API's
// IsSubstituteOccurrence). Set by [Substitute]; the simplified geometry it references
// is generated in M11-F06.
func (o *Occurrence) IsSubstitute() bool { return o.substitute }

// SetSubstitute marks the occurrence as a substitute (or not). It is the restore-time
// setter for the persisted flag (#715); the interactive substitute path goes through
// [Substitute], which also wires the simplified definition the flag stands for.
func (o *Occurrence) SetSubstitute(substitute bool) { o.substitute = substitute }

// SubOccurrences returns the components nested inside this occurrence when it
// instances an assembly (a [Composite] definition), or nil for a leaf part. They are
// the shared definition's occurrences — the flyweight: every placement of one
// sub-assembly navigates into the same children, disambiguated by occurrence path.
func (o *Occurrence) SubOccurrences() *Occurrences {
	if c, ok := o.definition.(Composite); ok {
		return c.Occurrences()
	}
	return nil
}

// RangeBox returns the occurrence's bounding box in its assembly's space: the
// definition's local box placed by the transform. A suppressed occurrence reports the
// empty box, contributing nothing to the assembly.
func (o *Occurrence) RangeBox() math.Box {
	if o.suppressed {
		return math.EmptyBox()
	}
	return o.definition.RangeBox().Transform(o.transform)
}
