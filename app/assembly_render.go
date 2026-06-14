// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// Assembly viewport geometry (#769): an assembly renders its placed components by transforming
// each occurrence's component body into assembly space. The same world-space bodies feed the
// renderer, the ray-picker (face/edge/occurrence hit-test), and view-fitting — so wiring them
// once through VisibleBodies lights up rendering AND picking for assemblies, which were
// previously invisible (VisibleBodies was part-only).
//
// Transforming a body re-derives its geometry (ops.TransformBody), so the result is cached and
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
		world, err := ops.TransformBody(pb.Body, pb.Transform, occurrenceBodyLineage(i))
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
// ray-picker to resolve a body hit to its component (occurrence-level selection). It reads the
// cache worldAssemblyBodies built, so it is only valid for bodies from the current frame's set.
func (s *Session) OccurrenceOfBody(b *topo.Body) (*occurrence.Occurrence, bool) {
	o, ok := s.asmBodies.owner[b]
	return o, ok
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
