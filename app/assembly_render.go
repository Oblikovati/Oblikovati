// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/scene"
)

// InstanceGroup is one unique component mesh and the world transforms it is drawn at — the
// render-side instancing unit (ADR-0038). A part placed K times in an assembly is ONE group with K
// transforms: the renderer tessellates/uploads Source once and draws it K times with a per-instance
// model matrix, instead of K independent transformed copies. Source is component-LOCAL geometry.
type InstanceGroup struct {
	Source     *topo.Body
	Transforms []math.Matrix4
}

// VisibleInstances returns the render scene as instance groups: a part is its visible bodies, each a
// single identity-transform group; an assembly's placements are grouped by their shared source body,
// so repeated components collapse to one mesh + many transforms. This is a render-only view —
// picking, selection and mass properties still use the world-space VisibleBodies (ADR-0038).
func (s *Session) VisibleInstances() []InstanceGroup {
	if asm, err := activeAssembly(s); err == nil {
		return s.assemblyInstances(asm)
	}
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	var out []InstanceGroup
	for _, body := range part.SurfaceBodies().All() {
		if s.BodyVisible(body) {
			out = append(out, InstanceGroup{Source: body, Transforms: []math.Matrix4{math.Identity4()}})
		}
	}
	return out
}

// CulledInstances returns the render instance groups limited to the components whose world AABB is
// inside the camera frustum — per-instance frustum culling backed by the assembly BVH, so off-screen
// placements never reach the GPU upload (M34-F1, essential at the 1M-placement target). Grouping
// matches VisibleInstances (by shared source body, first-seen order), so the head's per-source mesh
// cache still hits. A part — or no active assembly — has no index and is small, so it returns the
// full VisibleInstances unchanged. Use VisibleInstances (not this) for framing/shadow bounds, which
// must cover the whole model regardless of view.
func (s *Session) CulledInstances(cam scene.Camera) []InstanceGroup {
	asm, err := activeAssembly(s)
	if err != nil {
		return s.VisibleInstances()
	}
	idx := s.assemblyPickIndexFor(asm)
	visible := idx.frustumPlacements(cam.Frustum())
	visible = idx.detailVisible(visible, cam)
	return idx.groupPlacements(visible)
}

// assemblyInstances groups the assembly's placed bodies by their shared source body pointer (copies
// of one component reference the same definition's bodies), preserving first-seen order so the scene
// is deterministic. Suppression/visibility are already applied by PlacedBodies.
func (s *Session) assemblyInstances(asm *compdef.AssemblyComponentDefinition) []InstanceGroup {
	placed := asm.PlacedBodies()
	index := make(map[*topo.Body]int, len(placed))
	out := make([]InstanceGroup, 0, len(placed))
	for _, pb := range placed {
		i, ok := index[pb.Body]
		if !ok {
			i = len(out)
			index[pb.Body] = i
			out = append(out, InstanceGroup{Source: pb.Body})
		}
		out[i].Transforms = append(out[i].Transforms, pb.Transform)
	}
	return out
}

// Assembly viewport geometry (#769): an assembly renders its placed components by transforming
// each occurrence's component body into assembly space. The same world-space bodies feed the
// renderer, the ray-picker (face/edge/occurrence hit-test), and view-fitting — so wiring them
// once through VisibleBodies lights up rendering AND picking for assemblies, which were
// previously invisible (VisibleBodies was part-only).
//
// Transforming a body re-derives its geometry (transform.TransformBody), so the result is cached and
// only rebuilt when the occurrence structure changes (placement, suppression, add/remove — all
// bump occurrences.Revision). The cache returns STABLE *topo.Body pointers between rebuilds, so
// the head's per-body tessellation cache keeps hitting.

// assemblyBodyCache memoizes the active assembly's world-space bodies and the occurrence each
// came from (for occurrence picking), keyed on the assembly and its occurrence revision.
type assemblyBodyCache struct {
	asm      *compdef.AssemblyComponentDefinition
	revision uint64
	bodies   []*topo.Body
	owner    map[*topo.Body]*occurrence.Occurrence
}

// worldAssemblyBodies returns asm's unsuppressed placed bodies transformed into assembly space,
// rebuilding the cache when the occurrence structure changed. A component placed at several
// occurrences yields independent bodies (distinct lineage prefixes), so picking tells the
// instances apart.
func (s *Session) worldAssemblyBodies(asm *compdef.AssemblyComponentDefinition) []*topo.Body {
	rev := asm.Occurrences().Revision()
	if s.asmBodies.asm == asm && s.asmBodies.revision == rev && s.asmBodies.bodies != nil {
		return s.asmBodies.bodies
	}
	placed := asm.PlacedBodies()
	bodies := make([]*topo.Body, 0, len(placed))
	owner := make(map[*topo.Body]*occurrence.Occurrence, len(placed))
	for i, pb := range placed {
		world, err := transform.TransformBody(pb.Body, pb.Transform, occurrenceBodyLineage(i))
		if err != nil {
			s.notice = "assembly render: " + err.Error()
			continue
		}
		bodies = append(bodies, world)
		owner[world] = pb.Source
	}
	s.asmBodies = assemblyBodyCache{asm: asm, revision: rev, bodies: bodies, owner: owner}
	return bodies
}

// OccurrenceOfBody returns the occurrence a world-space assembly body was placed from, for the
// ray-picker to resolve a body hit to its component (occurrence-level selection). It checks the
// F5 pick index first (the path picking now takes) and falls back to the legacy
// worldAssemblyBodies cache, so either source of world body resolves.
func (s *Session) OccurrenceOfBody(b *topo.Body) (*occurrence.Occurrence, bool) {
	if s.pickIndex != nil {
		if o, ok := s.pickIndex.occurrenceOf(b); ok {
			return o, true
		}
	}
	o, ok := s.asmBodies.owner[b]
	return o, ok
}

// assemblyPickIndexFor returns the BVH pick index for asm, rebuilding it when the occurrence
// structure changed (the same revision key worldAssemblyBodies uses). It is the picker's
// spatial query so a selection does not materialize every world body (M34-F5).
func (s *Session) assemblyPickIndexFor(asm *compdef.AssemblyComponentDefinition) *assemblyPickIndex {
	rev := asm.Occurrences().Revision()
	if s.pickIndex != nil && s.pickIndex.asm == asm && s.pickIndex.revision == rev {
		return s.pickIndex
	}
	s.pickIndex = newAssemblyPickIndex(asm)
	return s.pickIndex
}

// RayPickBodies returns the world-space bodies a pick ray could hit, using the assembly BVH so
// only ray-crossed placements are transformed (M34-F5). For a part — or when there is no active
// assembly — it returns nil so the RayPicker falls back to its full body list; part scenes are
// small and need no index.
func (s *Session) RayPickBodies(origin math.Point3, dir math.Vector3) []*topo.Body {
	asm, err := activeAssembly(s)
	if err != nil {
		return nil
	}
	return s.assemblyPickIndexFor(asm).rayBodies(origin, dir)
}

// occurrenceBodyLineage gives each placed body a distinct lineage prefix by occurrence index, so
// the same component placed at several occurrences yields independent reference keys (the
// derived-assembly flatten uses the same scheme).
func occurrenceBodyLineage(index int) func(topo.Lineage) topo.Lineage {
	prefix := topo.Tok("occurrence", "occ", index)
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append([]topo.LineageToken{prefix}, l.Tokens()...)...)
	}
}
