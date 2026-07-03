// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"bytes"
	"errors"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// Assembly edge dress-up tools (M11, #766): chamfer and fillet a component edge on every placed
// instance of that component. The user picks an edge in the viewport (a world-space participant
// body, surfaced by #769's edge picking); the tool resolves it to the COMPONENT-LOCAL edge suffix
// and stores that, so the feature re-applies to every participant through the occurrence-relative
// resolver (#735). The geometry lives in model/feature; these are interaction shells driven
// headlessly here, with the distance/radius read from the generic tool-param dialog.

// componentEdgeSuffix turns a picked world-body edge's reference key into the component-local
// lineage suffix the assembly dress-up feature matches against each participant. A world edge's
// key is [kindByte] + "occurrence:occ#i/<componentLineage>": strip the kind byte (LineageSuffixOf),
// then drop the leading occurrence-prefix token (everything up to the first '/'), leaving
// "<componentLineage>" — the same suffix the wire path stores from a component-local key (#735).
func componentEdgeSuffix(referenceKey []byte) []byte {
	lineage := topo.LineageSuffixOf(referenceKey)
	if i := bytes.IndexByte(lineage, '/'); i >= 0 {
		return lineage[i+1:]
	}
	return lineage
}

// assemblyEdgeSelectTool collects the participant edges a dress-up acts on, filtering picks to
// edges and converting each to its component-local suffix on commit.
type assemblyEdgeSelectTool struct {
	edges []EdgeHandle
}

// Start restricts picking to edges, so a click selects a participant edge rather than the whole
// occurrence (the SelectEdge filter bypasses occurrence selection — see #769).
func (t *assemblyEdgeSelectTool) Start(*Session) {}

// AcceptedKinds declares the assembly dress-up picks component edges (the SelectEdge filter
// bypasses occurrence selection — see #769).
func (t *assemblyEdgeSelectTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectEdge} }

// Picks reports the picked component edges for the unified highlight.
func (t *assemblyEdgeSelectTool) Picks() []Selectable { return selectables(t.edges) }

// Pick appends a picked edge (ignoring a repeat, so a double-click does not duplicate it).
func (t *assemblyEdgeSelectTool) Pick(_ *Session, sel Selectable) {
	if e, ok := sel.(EdgeHandle); ok && !t.hasEdge(e) {
		t.edges = append(t.edges, e)
	}
}

func (t *assemblyEdgeSelectTool) hasEdge(e EdgeHandle) bool {
	for _, h := range t.edges {
		if h == e {
			return true
		}
	}
	return false
}

// Cancel abandons the picks and clears the edge filter.
func (t *assemblyEdgeSelectTool) Cancel(s *Session) {
	t.edges = nil
}

// resolve returns the component-local suffix of each picked edge, erroring
// when no assembly is active or nothing was picked.
func (t *assemblyEdgeSelectTool) resolve(s *Session, op string) ([][]byte, error) {
	if _, err := activeAssembly(s); err != nil {
		return nil, err
	}
	if len(t.edges) == 0 {
		return nil, errors.New(op + ": pick a component edge first")
	}
	suffixes := make([][]byte, len(t.edges))
	for i, e := range t.edges {
		suffixes[i] = componentEdgeSuffix(e.Edge.ReferenceKey())
	}
	return suffixes, nil
}

// finish commits the new dress-up feature through the shared assembly-feature
// seam (naming lives on the aggregate, recompute + undo on the verb — #1612).
func (t *assemblyEdgeSelectTool) finish(s *Session, f feature.Feature, label string) error {
	_, err := s.CommitAssemblyFeature(f, label)
	return err
}

// --- Chamfer --------------------------------------------------------------

// AssemblyChamferTool bevels the picked component edges by a setback distance on every participant.
type AssemblyChamferTool struct {
	assemblyEdgeSelectTool
	distance float64
}

// NewAssemblyChamferTool returns a chamfer tool with a default 1-unit setback.
func NewAssemblyChamferTool() *AssemblyChamferTool { return &AssemblyChamferTool{distance: 1} }
func (t *AssemblyChamferTool) Name() string        { return "Chamfer" }
func (t *AssemblyChamferTool) Prompt(*Session) string {
	return "Pick component edges, set the setback distance, then OK."
}
func (t *AssemblyChamferTool) CanCommit() bool { return len(t.edges) > 0 && t.distance > 0 }

func (t *AssemblyChamferTool) Commit(s *Session) error {
	suffixes, err := t.resolve(s, "chamfer")
	if err != nil {
		return err
	}
	d := t.distance
	return t.finish(s, feature.NewAssemblyChamferFeature(suffixes, func() float64 { return d }), "Chamfer")
}

func (t *AssemblyChamferTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Distance", func() float64 { return t.distance }, func(v float64) { t.distance = v }},
	}}
}

// --- Fillet ---------------------------------------------------------------

// AssemblyFilletTool rounds the picked component edges to a radius on every participant.
type AssemblyFilletTool struct {
	assemblyEdgeSelectTool
	radius float64
}

// NewAssemblyFilletTool returns a fillet tool with a default 1-unit radius.
func NewAssemblyFilletTool() *AssemblyFilletTool { return &AssemblyFilletTool{radius: 1} }
func (t *AssemblyFilletTool) Name() string       { return "Fillet" }
func (t *AssemblyFilletTool) Prompt(*Session) string {
	return "Pick component edges, set the radius, then OK."
}
func (t *AssemblyFilletTool) CanCommit() bool { return len(t.edges) > 0 && t.radius > 0 }

func (t *AssemblyFilletTool) Commit(s *Session) error {
	suffixes, err := t.resolve(s, "fillet")
	if err != nil {
		return err
	}
	r := t.radius
	return t.finish(s, feature.NewAssemblyFilletFeature(suffixes, func() float64 { return r }), "Fillet")
}

func (t *AssemblyFilletTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Radius", func() float64 { return t.radius }, func(v float64) { t.radius = v }},
	}}
}
