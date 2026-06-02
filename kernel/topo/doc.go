// SPDX-License-Identifier: GPL-2.0-only

// Package topo is the B-rep topological model: the Body→Shell→Face→Loop→EdgeUse→
// Edge→Vertex graph with full adjacency, each face/edge bound to its transient
// geometry (kernel/geom). It is pure Go and cgo-free (ADR-0002, architecture
// core/03).
//
// The non-negotiable design rule (parametric-cad §7): every topological entity
// records HOW it was generated — its [Lineage] ("end cap of feature F", "side wall
// of extrude E, profile edge p") — because persistent identity (reference keys,
// M03) derives from generative history, not from pointers. After a recompute
// destroys and recreates the B-rep, an entity is re-found by matching lineage.
//
// Layering note: topo sits below model/, so it does NOT import model/identity. It
// exposes lineage-derived [Entity.ReferenceKey] bytes; the identity.KeyManager that
// binds those keys lives in the model layer and is wired where features select
// topology (M08). topo proves rebind-after-recompute self-contained ([FindByKey]).
package topo
