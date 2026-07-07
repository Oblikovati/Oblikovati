// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// ownerBody returns the visible body that owns edge e (matched by identity), or nil when e is
// not part of the current scene — the reverse lookup the tangent-chain selection needs, since a
// picked topo.Edge carries no back-reference to its body.
func (s *Session) ownerBody(e *topo.Edge) *topo.Body {
	key := e.ReferenceKey()
	for _, b := range s.VisibleBodies() {
		if found, ok := b.FindEdgeByKey(key); ok && found == e {
			return b
		}
	}
	return nil
}

// tangentChainHandles expands a seed edge into its maximal run of tangent-continuous edge
// handles — the "select tangent chain / loop" affordance behind Fillet and Chamfer
// (Oblikovati/Oblikovati#1798). It returns just the seed when the owning body or the chain
// cannot be resolved, so a Shift-click never silently drops the user's pick.
func (s *Session) tangentChainHandles(seed EdgeHandle) []EdgeHandle {
	b := s.ownerBody(seed.Edge)
	if b == nil {
		return []EdgeHandle{seed}
	}
	keys, _, err := ops.TangentEdgeChain(b, seed.Edge.ReferenceKey(), ops.DefaultTangentChainAngle)
	if err != nil {
		return []EdgeHandle{seed}
	}
	out := make([]EdgeHandle, 0, len(keys))
	for _, k := range keys {
		if e, ok := b.FindEdgeByKey(k); ok {
			out = append(out, EdgeHandle{Edge: e})
		}
	}
	return out
}
