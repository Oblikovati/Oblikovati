// SPDX-License-Identifier: GPL-2.0-only

package renderer

// VisualStyle selects how the scene bodies are drawn — Inventor's Visual Style / display
// mode (DisplayModeEnum). Every style resolves, via [PassSetFor], to a [PassSet] that says
// which passes the frame draws: how faces are shaded, how edges are drawn, and whether an
// NPR outline is composited. The resolver is the single place a mode is defined; the draw
// list and the native pipeline both read it (ADR-0023 §1). It is pure data — no GPU — so it
// is unit-tested on the null/CPU path (ADR-0014).
type VisualStyle uint8

const (
	// Shaded draws lit faces only (no wireframe).
	Shaded VisualStyle = iota
	// ShadedWithEdges draws lit faces with the edge wireframe over them (the default).
	ShadedWithEdges
	// Wireframe draws every edge (no shaded faces, no hidden-line removal).
	Wireframe
	// Realistic draws faces with the physically based (GGX metallic-roughness) shader.
	Realistic
	// ShadedWithHiddenEdges draws shaded faces with visible edges solid and occluded edges
	// dashed (Inventor kShadedWithHiddenEdgesRendering / kHiddenEdgeRendering, 8707).
	ShadedWithHiddenEdges
	// WireframeWithHiddenEdges draws visible edges solid and occluded edges dashed, no faces
	// (Inventor kWireframeWithHiddenEdgesRendering, 8712).
	WireframeWithHiddenEdges
	// WireframeVisibleOnly draws only the visible edges, occluded edges removed
	// (Inventor kWireframeNoHiddenEdges, 8711).
	WireframeVisibleOnly
	// Monochrome draws a desaturated, posterized NPR look with outlines (8713).
	Monochrome
	// Watercolor draws soft pigment washes on paper with edge darkening (8714).
	Watercolor
	// Illustration draws flat/cel shading with crisp outlines (8715).
	Illustration
	// TechnicalIllustration draws Gooch cool-warm shading with emphasized edges (8716).
	TechnicalIllustration
)

// Shading is how a style lights its faces.
type Shading uint8

const (
	// ShadeNone draws no faces (wireframe-family styles).
	ShadeNone Shading = iota
	// ShadeFlat is the baseline Blinn-Lambert headlight shading (Shaded/ShadedWithEdges).
	ShadeFlat
	// ShadePBR is the GGX metallic-roughness physically based shading (Realistic).
	ShadePBR
	// ShadeMonochrome is desaturated + posterized luminance (Monochrome NPR).
	ShadeMonochrome
	// ShadeCel is flat banded diffuse (Illustration NPR).
	ShadeCel
	// ShadeGooch is cool-warm tone shading (Technical Illustration NPR).
	ShadeGooch
	// ShadeWatercolor is soft pigment washes on paper (Watercolor NPR).
	ShadeWatercolor
)

// EdgeMode is how a style draws model edges.
type EdgeMode uint8

const (
	// EdgesNone draws no edge lines.
	EdgesNone EdgeMode = iota
	// EdgesAll draws every edge with no hidden-line removal.
	EdgesAll
	// EdgesVisible draws only the visible edges (hidden-line removal, occluded dropped).
	EdgesVisible
	// EdgesVisiblePlusHidden draws visible edges solid and occluded edges dashed.
	EdgesVisiblePlusHidden
)

// PassSet is the resolved set of passes a [VisualStyle] draws — the output of [PassSetFor].
type PassSet struct {
	Faces   Shading  // how faces are shaded (ShadeNone ⇒ no surface pass)
	Edges   EdgeMode // how model edges are drawn
	Outline bool     // composite an NPR silhouette outline (the NPR edge-detection pass)
}

// HidesEdges reports whether the style needs per-frame hidden-line removal (any edge mode
// beyond "draw them all").
func (p PassSet) HidesEdges() bool {
	return p.Edges == EdgesVisible || p.Edges == EdgesVisiblePlusHidden
}

// styleSpec is one row of the resolver table: a style, its pass set, and its display name.
type styleSpec struct {
	style VisualStyle
	name  string
	pass  PassSet
}

// styleTable defines every visual style exactly once, in the order Inventor's Visual Style
// gallery presents them. Adding a mode is a single row here plus the enum value above — there
// is no default fall-through (see TestPassSetForIsTotal), so a new style cannot be silently
// mis-resolved.
var styleTable = []styleSpec{
	{Realistic, "Realistic", PassSet{Faces: ShadePBR, Edges: EdgesNone}},
	{Shaded, "Shaded", PassSet{Faces: ShadeFlat, Edges: EdgesNone}},
	{ShadedWithEdges, "Shaded with Edges", PassSet{Faces: ShadeFlat, Edges: EdgesAll}},
	{ShadedWithHiddenEdges, "Shaded with Hidden Edges", PassSet{Faces: ShadeFlat, Edges: EdgesVisiblePlusHidden}},
	{Wireframe, "Wireframe", PassSet{Faces: ShadeNone, Edges: EdgesAll}},
	{WireframeVisibleOnly, "Wireframe with Visible Edges Only", PassSet{Faces: ShadeNone, Edges: EdgesVisible}},
	{WireframeWithHiddenEdges, "Wireframe with Hidden Edges", PassSet{Faces: ShadeNone, Edges: EdgesVisiblePlusHidden}},
	{Monochrome, "Monochrome", PassSet{Faces: ShadeMonochrome, Edges: EdgesNone, Outline: true}},
	{Watercolor, "Watercolor", PassSet{Faces: ShadeWatercolor, Edges: EdgesNone, Outline: true}},
	{Illustration, "Illustration", PassSet{Faces: ShadeCel, Edges: EdgesNone, Outline: true}},
	{TechnicalIllustration, "Technical Illustration", PassSet{Faces: ShadeGooch, Edges: EdgesNone, Outline: true}},
}

// PassSetFor resolves a style to the passes it draws. It is total over every defined
// VisualStyle; an unknown value falls back to the default ShadedWithEdges pass set so a
// corrupt style never yields a blank frame.
func PassSetFor(style VisualStyle) PassSet {
	for _, sp := range styleTable {
		if sp.style == style {
			return sp.pass
		}
	}
	return PassSet{Faces: ShadeFlat, Edges: EdgesAll}
}

// String returns the stable, user-facing name of the style (the Visual Style gallery label).
func (v VisualStyle) String() string {
	for _, sp := range styleTable {
		if sp.style == v {
			return sp.name
		}
	}
	return "Shaded with Edges"
}

// VisualStyleOption pairs a style with its gallery label, for building the UI selection box.
type VisualStyleOption struct {
	Style VisualStyle
	Name  string
}

// VisualStyleGallery returns every visual style with its label, in Inventor gallery order —
// the source list for the View-tab selection box.
func VisualStyleGallery() []VisualStyleOption {
	opts := make([]VisualStyleOption, len(styleTable))
	for i, sp := range styleTable {
		opts[i] = VisualStyleOption{Style: sp.style, Name: sp.name}
	}
	return opts
}
