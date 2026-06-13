// SPDX-License-Identifier: GPL-2.0-only

// Package proxy views a definition-space entity (a part's face/edge/vertex or a work
// feature) through an assembly occurrence context, so it reports assembly-space
// geometry while still identifying the underlying native entity. It is the bridge
// assembly features, mates, and measurement need (M11-F03, #347).
//
// The mechanism is implemented ONCE, generically: [Proxy] is parameterized over the
// native entity type, so every entity kind gets a proxy for free and — crucially —
// Proxy[E] is a DISTINCT type from the native E. A definition-space entity therefore
// cannot be used where an assembly-space proxy is required (or vice versa); the
// native≠proxy distinction is enforced by the compiler, not by convention. This is
// the reference API's CreateGeometryProxy and its ~275 *Proxy types, without any
// hand-authored per-entity proxy.
package proxy

import (
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// Context is the assembly-space context a definition-space entity is viewed through:
// the transform from the entity's definition space to assembly space (composed along
// the occurrence path) and the occurrence it is reached through, kept for identity —
// the reference API's ContextDefinition role.
type Context struct {
	occurrence *occurrence.Occurrence
	transform  math.Matrix4
}

// NewContext views entities through a single occurrence: the transform is the
// occurrence's own placement in its assembly.
func NewContext(o *occurrence.Occurrence) Context {
	return Context{occurrence: o, transform: o.Transform()}
}

// NewPathContext views entities through a chain of nested occurrences — root first,
// leaf last, as returned by [occurrence.Occurrences.ResolveChain]. The transform
// composes each occurrence's placement (root·…·leaf) so an entity in the leaf's
// definition space lands in the root assembly's space; the leaf occurrence is the
// identifying context. An empty chain yields the identity context.
func NewPathContext(chain []*occurrence.Occurrence) Context {
	t := math.Identity4()
	for _, o := range chain {
		t = t.Mul(o.Transform())
	}
	var leaf *occurrence.Occurrence
	if len(chain) > 0 {
		leaf = chain[len(chain)-1]
	}
	return Context{occurrence: leaf, transform: t}
}

// Transform returns this context's definition→assembly-space transform.
func (c Context) Transform() math.Matrix4 { return c.transform }

// Occurrence returns the occurrence this context views entities through (the leaf of
// the path), or nil for an empty path.
func (c Context) Occurrence() *occurrence.Occurrence { return c.occurrence }

// Proxy views a native definition-space entity E through a [Context], reporting
// assembly-space geometry (via the In*Context helpers) while still identifying the
// underlying native E (via [Proxy.Native]). Proxy[E] is a distinct type from E — see
// the package doc on the native≠proxy distinction.
type Proxy[E any] struct {
	native  E
	context Context
}

// CreateGeometryProxy views native through ctx — the reference API's
// ComponentOccurrence.CreateGeometryProxy, generic over the entity so the proxy's type
// matches the underlying entity.
//
//	p := proxy.CreateGeometryProxy(proxy.NewContext(occ), face) // Proxy[*topo.Face]
func CreateGeometryProxy[E any](ctx Context, native E) Proxy[E] {
	return Proxy[E]{native: native, context: ctx}
}

// Native returns the underlying definition-space entity this proxy views — the
// reverse of [CreateGeometryProxy], recovering the native identity.
func (p Proxy[E]) Native() E { return p.native }

// Context returns the assembly-space context this proxy views its native through.
func (p Proxy[E]) Context() Context { return p.context }
