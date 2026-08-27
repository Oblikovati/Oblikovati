// SPDX-License-Identifier: GPL-2.0-only

// Package collview is the shared index-guarded read-only view behind the contract
// collection implementations (M40 audit G7, #1655). The bounds guard is the safety
// property of the whole object-model surface: Item(i) must return the zero value
// (nil), never panic, because add-ins probe collections with raw indices. One guard
// here replaces the hand-typed copies each contract collection used to carry.
// The api/contract collection INTERFACES stay non-generic by design (COM-style
// object model); only the GPL-side implementations use this view.
//
// Promoted out of model/internal (G16 #2177) once app/'s own contract
// implementations (colorStylesView, colorSchemesAdapter) needed the same guard —
// Go's internal-package visibility only reaches model/'s own subtree, so a package
// two sibling trees both legitimately depend on cannot stay internal to either.
package collview

// Indexed adapts a concrete slice to a contract Count/Item collection, e.g.
//
//	collview.Over(r.design, func(v *designViewRep) contract.DesignViewRepresentation { return v })
//
// The as conversion is needed because Go does not implicitly convert []Elem to
// []Iface; it also guarantees the out-of-range result is a true nil interface
// (returning a zero Elem pointer through an interface would be non-nil).
type Indexed[Elem, Iface any] struct {
	items []Elem
	as    func(Elem) Iface
}

// Over builds an Indexed view of items, converting each element with as.
func Over[Elem, Iface any](items []Elem, as func(Elem) Iface) Indexed[Elem, Iface] {
	return Indexed[Elem, Iface]{items: items, as: as}
}

// Count returns the number of elements in the view.
func (v Indexed[Elem, Iface]) Count() int { return len(v.items) }

// Item returns the i-th element as its contract interface, or the zero Iface
// (nil) when i is out of range — never panicking.
func (v Indexed[Elem, Iface]) Item(i int) Iface { return ItemAs(v.items, i, v.as) }

// ItemAs is the shared bounds guard for collections that keep their own struct
// (they carry mutation or lookup methods) but expose the same probing-safe Item, e.g.
//
//	func (s *JointSet) Item(i int) contract.AssemblyJoint {
//		return collview.ItemAs(s.items, i, func(j Joint) contract.AssemblyJoint { return j })
//	}
func ItemAs[Elem, Iface any](items []Elem, i int, as func(Elem) Iface) Iface {
	var zero Iface
	if i < 0 || i >= len(items) {
		return zero
	}
	return as(items[i])
}

// At is ItemAs for collections whose Item returns the element type itself
// (e.g. *WorkSurface): out of range yields the zero Elem — a nil pointer.
func At[Elem any](items []Elem, i int) Elem { return ItemAs(items, i, same[Elem]) }

// same is the identity conversion At plugs into the shared guard.
func same[Elem any](e Elem) Elem { return e }
