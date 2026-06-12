// SPDX-License-Identifier: GPL-2.0-only

package topo

import "oblikovati.org/math"

// Shell queries and connectivity regrouping (M07-F06,
// Oblikovati/Oblikovati#629): the FaceShell API object needs shells that mean
// something — one per edge-connected face group — plus edges, range boxes and
// stable reference keys.

// Edges returns the shell's distinct edges (via its faces' loops).
func (s *Shell) Edges() []*Edge {
	seen := map[*Edge]bool{}
	var out []*Edge
	for _, f := range s.faces {
		for _, e := range f.Edges() {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// Index returns the shell's position among its body's shells (-1 if detached).
func (s *Shell) Index() int {
	for i, sh := range s.body.shells {
		if sh == s {
			return i
		}
	}
	return -1
}

// Lineage derives the shell's lineage from its body's, extended by the shell
// ordinal. Shells are connectivity groups, not modeled entities — they have no
// generative path of their own, so the body's path plus a stable position is
// the identity that survives a recompute (the regrouping walks faces in body
// face order, which recompute reproduces).
func (s *Shell) Lineage() Lineage {
	return NewLineage(append(s.body.lineage.Tokens(), Tok("shell", "shell", s.Index()))...)
}

// ReferenceKey returns the shell's persistent reference key (M03 scheme).
func (s *Shell) ReferenceKey() []byte {
	return referenceKey(KindShell, s.Lineage())
}

// RangeBox returns the union of the shell's face range boxes.
func (s *Shell) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, f := range s.faces {
		box = box.Union(f.RangeBox())
	}
	return box
}

// RegroupShells re-partitions a body's faces into true connectivity shells:
// faces sharing an edge belong to one shell, and a shell is closed when every
// edge it touches is used at least twice within the group. Builder.Build runs
// this so a body's shells always reflect actual face connectivity (a stitched
// pair of disjoint quilts is two shells, a cavity cut is the outer skin plus
// the void skin).
func RegroupShells(b *Body) {
	faces := b.Faces()
	if len(faces) == 0 {
		return
	}
	groups := connectedFaceGroups(faces)
	b.shells = make([]*Shell, len(groups))
	for i, g := range groups {
		sh := &Shell{id: nextID(), body: b, faces: g, closed: groupClosed(g)}
		for _, f := range g {
			f.shell = sh
		}
		b.shells[i] = sh
	}
}

// connectedFaceGroups partitions faces into edge-connected groups, preserving
// body face order within and across groups (stable shell identity).
func connectedFaceGroups(faces []*Face) [][]*Face {
	parent := make([]int, len(faces))
	for i := range parent {
		parent[i] = i
	}
	byEdge := map[*Edge]int{}
	for i, f := range faces {
		for _, e := range f.Edges() {
			if first, ok := byEdge[e]; ok {
				union(parent, first, i)
			} else {
				byEdge[e] = i
			}
		}
	}
	return groupByRoot(faces, parent)
}

// groupByRoot buckets faces by their union-find root, ordered by first member.
func groupByRoot(faces []*Face, parent []int) [][]*Face {
	order := []int{}
	members := map[int][]*Face{}
	for i, f := range faces {
		r := find(parent, i)
		if _, ok := members[r]; !ok {
			order = append(order, r)
		}
		members[r] = append(members[r], f)
	}
	out := make([][]*Face, len(order))
	for i, r := range order {
		out[i] = members[r]
	}
	return out
}

// groupClosed reports whether every edge of the group is used at least twice by
// the group's faces (no boundary edge → the shell bounds a region).
func groupClosed(faces []*Face) bool {
	uses := map[*Edge]int{}
	for _, f := range faces {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				uses[u.Edge()]++
			}
		}
	}
	for _, n := range uses {
		if n < 2 {
			return false
		}
	}
	if len(uses) == 0 {
		// No edges at all: a boundary-less face is a closed surface (a whole
		// sphere or torus carried as one loop-less face) — closed by definition.
		return len(faces) > 0
	}
	return true
}

// union and find are the package's shared union-find primitives.
func union(parent []int, i, j int) {
	ri, rj := find(parent, i), find(parent, j)
	if ri != rj {
		parent[rj] = ri
	}
}

func find(parent []int, i int) int {
	for parent[i] != i {
		parent[i] = parent[parent[i]]
		i = parent[i]
	}
	return i
}
