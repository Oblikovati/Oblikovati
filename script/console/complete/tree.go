// SPDX-License-Identifier: GPL-2.0-only

// Package complete is the autocomplete engine for the Script Console editor. Given the source
// line and caret column it returns ranked candidates: the live `oblikovati.*` host API (built
// from the dotted wire-method names the host already publishes via methods()), plus Lua
// keywords and standard-library builtins. It is pure Go — the cgo popup just renders the
// candidates — so context detection and ranking are unit-tested headlessly (ADR-0028).
package complete

import "strings"

// node is one segment of the API namespace trie. children holds the next segments by name;
// method is true when a full wire method ends here (a leaf the script can call). A node can be
// both a namespace and — rarely — a method, so the flags are independent.
type node struct {
	children map[string]*node
	method   bool
}

// newNode returns an empty namespace node.
func newNode() *node { return &node{children: map[string]*node{}} }

// child returns the named child, or nil when absent.
func (n *node) child(name string) *node { return n.children[name] }

// buildTree assembles the namespace trie from dotted wire-method names (e.g.
// "documents.activate", "sketch.rectangle"). Each name's segments become a path; the final
// segment is marked as a method. The root represents `oblikovati`.
func buildTree(methods []string) *node {
	root := newNode()
	for _, m := range methods {
		segs := strings.Split(m, ".")
		cur := root
		for i, seg := range segs {
			next := cur.child(seg)
			if next == nil {
				next = newNode()
				cur.children[seg] = next
			}
			if i == len(segs)-1 {
				next.method = true
			}
			cur = next
		}
	}
	return root
}

// walk descends the trie through the exact segments in path, returning the reached node or nil
// if any segment is missing. An empty path returns the start node.
func (n *node) walk(path []string) *node {
	cur := n
	for _, seg := range path {
		cur = cur.child(seg)
		if cur == nil {
			return nil
		}
	}
	return cur
}
