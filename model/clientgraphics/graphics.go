// SPDX-License-Identifier: GPL-2.0-only

// Package clientgraphics is the host-side store and renderer-builder for add-in client
// and interaction graphics (Inventor's ClientGraphics + InteractionGraphics). Add-ins
// submit declarative graphics groups over the wire (api/wire); the router decodes them
// into this package's kernel-typed model, and Build turns the live groups into
// renderer.DrawItem geometry (with per-vertex heatmap colors, glyphs and labels) for the
// frame. It is pure Go with no cgo, so the whole path is headless-testable.
//
// Three lanes mirror Inventor's two surfaces: LanePersistent is document-owned
// ClientGraphics; LaneOverlay and LanePreview are the command-scoped InteractionGraphics
// lanes (overlay draws on top of the scene, preview draws depth-tested with it).
package clientgraphics

import (
	"github.com/Oblikovati/api/types"

	"github.com/Oblikovati/oblikovati/math"
)

// Lane is the display lane of a graphics group (a types.GraphicsLane value, kept as a
// host alias so callers don't depend on the contract module everywhere).
type Lane = types.GraphicsLane

const (
	LanePersistent = types.GraphicsLanePersistent
	LaneOverlay    = types.GraphicsLaneOverlay
	LanePreview    = types.GraphicsLanePreview
)

// ColorMapper maps a scalar value to a color by piecewise-linear interpolation across
// ascending Values stops (Inventor's GraphicsColorMapper) — the heatmap legend.
type ColorMapper struct {
	Values []float64
	Colors [][4]float32
}

// Primitive is one decoded drawable primitive: geometry in kernel types plus its
// rendering attributes. Per-vertex color is resolved at Build from Colors, or Scalars
// through Mapper, or the overall Color.
type Primitive struct {
	Kind          types.GraphicsPrimitiveKind
	Coords        []math.Point3
	Indices       []int
	Colors        [][4]float32
	Normals       []math.Vector3
	Scalars       []float64
	Mapper        *ColorMapper
	Color         [4]float32
	ColorBinding  types.GraphicsColorBinding
	NormalBinding types.GraphicsNormalBinding
	LineType      types.GraphicsLineType
	LineWeight    float64
	PointStyle    types.GraphicsPointStyle
	PointSize     float64
	Text          string
	Anchor        math.Point3
	FontSize      float64
	Opacity       float32
	OnTop         bool
}

// Node groups primitives under one optional transform and visibility/opacity.
type Node struct {
	Transform    math.Matrix4
	HasTransform bool
	Visible      *bool
	Opacity      float32
	Primitives   []Primitive
}

// Group is one named client-graphics group in a lane. It satisfies
// contract.ClientGraphics via accessor methods (fields stay unexported so the method set
// is the boundary).
type Group struct {
	clientID string
	lane     Lane
	visible  bool
	nodes    []Node
}

// Name is the group's client id (its store key).
func (g *Group) Name() string { return g.clientID }

// Lane is the group's display lane (a types.GraphicsLane value as a string).
func (g *Group) Lane() string { return string(g.lane) }

// Visible reports whether the group is currently drawn.
func (g *Group) Visible() bool { return g.visible }

// NodeCount is the number of graphics nodes the group owns.
func (g *Group) NodeCount() int { return len(g.nodes) }

// PrimitiveCount totals the primitives across the group's nodes.
func (g *Group) PrimitiveCount() int {
	n := 0
	for i := range g.nodes {
		n += len(g.nodes[i].Primitives)
	}
	return n
}

// Label is a world-anchored text label extracted from a "text" primitive at Build —
// drawn by the UI head's projected-ImGui path, not as draw-list geometry.
type Label struct {
	Anchor   math.Point3
	Text     string
	Color    [4]float32
	FontSize float64
}
