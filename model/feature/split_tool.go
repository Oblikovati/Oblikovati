// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// What a split cuts WITH — Inventor's SplitToolTypeEnum (#1891). The split itself (which side to
// keep, faces-only) is in split_solid.go; this file is only about resolving the tool to the
// cutting geometry, because that is where the four tool types differ.

// SplitToolKind names the geometry a split cuts with.
type SplitToolKind uint8

const (
	// SplitByWorkPlane cuts along a datum plane — the original tool, and the default.
	SplitByWorkPlane SplitToolKind = iota
	// SplitByWorkSurface cuts along a construction surface, addressed by its position in the
	// part's WorkSurfaces (which is the order the sheet bodies appear in the result).
	SplitByWorkSurface
	// SplitBySurfaceBody cuts along a running surface body, addressed by its body index.
	SplitBySurfaceBody
	// SplitByPath cuts a FACE along a projected 2D sketch path. Accepted here so the recipe and
	// the wire can name it; the geometry belongs to the deferred split-face feature.
	SplitByPath
)

// splitToolNames is the stable wire/recipe vocabulary for the tool kinds — one source shared by
// the op registry and the .obk codec so they cannot drift.
var splitToolNames = map[SplitToolKind]string{
	SplitByWorkPlane:   "workPlane",
	SplitByWorkSurface: "workSurface",
	SplitBySurfaceBody: "surfaceBody",
	SplitByPath:        "path",
}

// SplitToolName renders a tool kind as its stable name ("" for the work-plane default, so the
// common case serializes nothing).
func SplitToolName(k SplitToolKind) string {
	if k == SplitByWorkPlane {
		return ""
	}
	return splitToolNames[k]
}

// ParseSplitTool maps a name to its tool kind (empty ⇒ work plane); ok is false for an unknown
// name, which must be refused rather than defaulting to the plane and cutting somewhere else.
func ParseSplitTool(name string) (SplitToolKind, bool) {
	if name == "" {
		return SplitByWorkPlane, true
	}
	for k, n := range splitToolNames {
		if n == name {
			return k, true
		}
	}
	return SplitByWorkPlane, false
}

// cuttingPlane resolves the definition's tool to the plane the split cuts along.
//
// A surface tool cuts along ITS PLANE, extended — the same infinite-plane rule the work-plane
// split already follows, so a sheet that only partly spans the body still cuts all the way
// through. A CURVED sheet has no such plane and is refused: trimming to it needs a general
// surface partitioner, not a plane (tracked as a follow-up on #1891).
func (d *SplitSolidDefinition) cuttingPlane(bodies []*topo.Body) (geom.Plane, error) {
	switch d.Tool {
	case SplitByWorkPlane:
		if d.Plane == nil {
			return geom.Plane{}, errors.New("split: no cutting plane")
		}
		return geomPlaneOf(d.Plane)
	case SplitByWorkSurface:
		return sheetToolPlane(surfaceBodiesOf(bodies), d.ToolIndex, "work surface")
	case SplitBySurfaceBody:
		return sheetToolPlane(bodies, d.ToolIndex, "surface body")
	default:
		return geom.Plane{}, fmt.Errorf("split: the %q tool projects a 2D sketch path onto a face, "+
			"which is the split-FACE geometry; split by a work plane, work surface or surface body",
			splitToolNames[d.Tool])
	}
}

// sheetToolPlane picks the tool sheet out of candidates by index and returns the plane it lies in.
func sheetToolPlane(candidates []*topo.Body, index int, what string) (geom.Plane, error) {
	if index < 0 || index >= len(candidates) {
		return geom.Plane{}, fmt.Errorf("split: %s %d out of range (the part has %d)", what, index, len(candidates))
	}
	sheet := candidates[index]
	if sheet.IsSolid() {
		return geom.Plane{}, fmt.Errorf("split: %s %d is a SOLID, not a sheet; a solid tool is a "+
			"combine, not a split", what, index)
	}
	pl, ok := ops.BodyPlane(sheet)
	if !ok {
		return geom.Plane{}, fmt.Errorf("split: %s %d is not planar, and a curved cutting surface "+
			"needs a general surface partitioner rather than a plane; use a planar surface or a "+
			"work plane", what, index)
	}
	return pl, nil
}
