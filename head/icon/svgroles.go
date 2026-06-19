// SPDX-License-Identifier: GPL-2.0-only

package icon

import (
	"fmt"
	"strings"

	"oblikovati.org/theme/blenderxml"
)

// SVG role filtering: an asset is split into per-role documents by each element's
// effective paint (its own stroke/fill, or the nearest ancestor's — the root <svg>
// carries the shared stroke defaults). The filtered document keeps the role's paints
// and turns every other paint off, so rasterizing it yields that role's coverage mask.

// shapeElems are the SVG elements that actually draw; only their paints are classified
// (groups and the root just carry inherited attributes).
var shapeElems = map[string]bool{
	"rect": true, "circle": true, "ellipse": true, "line": true,
	"polyline": true, "polygon": true, "path": true,
}

// paintState is the inherited stroke/fill pair at a point in the tree. SVG's defaults
// are fill=black, stroke=none.
type paintState struct{ stroke, fill string }

var rootPaintDefaults = paintState{stroke: "", fill: hexBlack}

// filterForRole returns a copy of the document in which only elements painted with
// role's sentinel still draw — the input to the role's rasterization pass.
func filterForRole(doc *blenderxml.Node, role Role) *blenderxml.Node {
	out := doc.Clone()
	rewritePaints(out, rootPaintDefaults, sentinelPaint[role])
	return out
}

// rewritePaints walks the tree tracking inherited paints; every shape gets explicit
// stroke/fill attributes: black where its effective paint is the wanted sentinel,
// "none" otherwise. Black vs none is all that matters — the pass only produces
// coverage, never color.
func rewritePaints(n *blenderxml.Node, inherited paintState, want string) {
	state := inheritPaints(n, inherited)
	if shapeElems[n.XMLName.Local] {
		n.SetAttr("stroke", keepPaint(state.stroke, want))
		n.SetAttr("fill", keepPaint(state.fill, want))
	}
	for _, c := range n.Children {
		rewritePaints(c, state, want)
	}
}

// inheritPaints overlays n's own stroke/fill attributes onto the inherited state.
func inheritPaints(n *blenderxml.Node, inherited paintState) paintState {
	out := inherited
	if v, ok := n.Attr("stroke"); ok {
		out.stroke = normalizePaint(v)
	}
	if v, ok := n.Attr("fill"); ok {
		out.fill = normalizePaint(v)
	}
	return out
}

// keepPaint maps an effective paint to the filtered document's explicit attribute:
// drawn (black) when it matches the wanted sentinel, off otherwise.
func keepPaint(paint, want string) string {
	if paint == want {
		return hexBlack
	}
	return "none"
}

// normalizePaint canonicalizes an SVG paint to lower-case "#rrggbb" ("" for none), so
// "#F00", "#ff0000" and "red" compare equal as the secondary sentinel.
func normalizePaint(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "", "none", "transparent":
		return ""
	case "black":
		return hexBlack
	case "red":
		return "#ff0000"
	case "blue":
		return "#0000ff"
	case "lime":
		return "#00ff00"
	}
	if len(v) == 4 && v[0] == '#' { // #rgb -> #rrggbb
		return strings.ToLower(fmt.Sprintf("#%c%c%c%c%c%c", v[1], v[1], v[2], v[2], v[3], v[3]))
	}
	return v
}

// UsedPaints returns every distinct effective paint the document's shapes draw with
// (normalized, "none" excluded). The asset conformance test uses it to reject an icon
// authored with a color outside the role sentinels.
func UsedPaints(svg []byte) ([]string, error) {
	doc, err := blenderxml.Parse(svg)
	if err != nil {
		return nil, fmt.Errorf("icon: parse SVG: %w", err)
	}
	seen := map[string]bool{}
	collectPaints(doc, rootPaintDefaults, seen)
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out, nil
}

// collectPaints accumulates the effective shape paints under n into seen.
func collectPaints(n *blenderxml.Node, inherited paintState, seen map[string]bool) {
	state := inheritPaints(n, inherited)
	if shapeElems[n.XMLName.Local] {
		for _, p := range []string{state.stroke, state.fill} {
			if p != "" {
				seen[p] = true
			}
		}
	}
	for _, c := range n.Children {
		collectPaints(c, state, seen)
	}
}
