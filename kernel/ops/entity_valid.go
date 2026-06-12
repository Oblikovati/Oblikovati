// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// Entity validity (M07-F07, Oblikovati/Oblikovati#630): the IsEntityValid
// API face over Validate (PBI-084) plus, at the deeper check level, the
// self-intersection scan. Problem entities are reported with their kind,
// session id and reference key so a client can bind back to them.

// EntityCheckLevel selects how deep ValidateEntity digs.
type EntityCheckLevel int

const (
	// CheckTopology runs the manifold/orientation/closure checks (cheap).
	CheckTopology EntityCheckLevel = 1
	// CheckGeometry adds the face self-intersection scan (tessellates).
	CheckGeometry EntityCheckLevel = 2
)

// ProblemEntity is one offending entity of a failed validity check.
type ProblemEntity struct {
	Kind         topo.EntityKind
	ID           uint64
	ReferenceKey []byte
	Issue        string
}

// ValidateBodyEntities checks the whole body at the given level, returning
// the verdict and each offending entity.
//
// Example: ok, problems := ops.ValidateBodyEntities(b, ops.CheckGeometry, ops.DefaultQuality())
func ValidateBodyEntities(b *topo.Body, level EntityCheckLevel, q Quality) (bool, []ProblemEntity) {
	problems := topologyProblems(b)
	if level >= CheckGeometry {
		for _, hit := range SelfIntersections(b, q) {
			problems = append(problems,
				faceProblem(hit.FaceA, fmt.Sprintf("self-intersects face %d near %v", hit.FaceB.ID(), hit.Witness)),
				faceProblem(hit.FaceB, fmt.Sprintf("self-intersects face %d near %v", hit.FaceA.ID(), hit.Witness)))
		}
	}
	return len(problems) == 0, problems
}

// topologyProblems maps Validate's per-edge findings onto problem entities.
func topologyProblems(b *topo.Body) []ProblemEntity {
	var out []ProblemEntity
	for _, e := range b.Edges() {
		switch uses := e.Uses(); {
		case len(uses) < 2 && b.IsSolid():
			out = append(out, edgeProblem(e, "boundary (open) edge on a solid"))
		case len(uses) > 2:
			out = append(out, edgeProblem(e, fmt.Sprintf("non-manifold: used by %d faces", len(uses))))
		case len(uses) == 2 && uses[0].Reversed() == uses[1].Reversed():
			out = append(out, edgeProblem(e, "inconsistent orientation between its two uses"))
		}
	}
	return out
}

// EdgeEntityValid checks one edge: 1–2 uses, opposite orientation when manifold.
func EdgeEntityValid(e *topo.Edge) bool {
	uses := e.Uses()
	if len(uses) == 0 || len(uses) > 2 {
		return false
	}
	return len(uses) == 1 || uses[0].Reversed() != uses[1].Reversed()
}

// FaceEntityValid checks one face: an outer loop with edges, every use wired
// back to this face.
func FaceEntityValid(f *topo.Face) bool {
	hasOuter := false
	for _, l := range f.Loops() {
		if len(l.EdgeUses()) == 0 {
			return false
		}
		if l.IsOuter() {
			hasOuter = true
		}
		for _, u := range l.EdgeUses() {
			if u.Loop().Face() != f {
				return false
			}
		}
	}
	return hasOuter
}

func edgeProblem(e *topo.Edge, issue string) ProblemEntity {
	return ProblemEntity{Kind: topo.KindEdge, ID: e.ID(), ReferenceKey: e.ReferenceKey(), Issue: issue}
}

func faceProblem(f *topo.Face, issue string) ProblemEntity {
	return ProblemEntity{Kind: topo.KindFace, ID: f.ID(), ReferenceKey: f.ReferenceKey(), Issue: issue}
}
