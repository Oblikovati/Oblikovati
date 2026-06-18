// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/analysis"
)

// Measure (M18-F01 PBI-164, #428): the interactive measurement command. The user picks faces, edges
// and vertices; after each pick the tool reports every applicable quantity for the current
// selection — an edge's length, a face's area and perimeter, the distance or minimum distance
// between two entities, the angle between two directed entities, and the angle at the apex of three
// vertices. It is read-only: it reuses the model/analysis measurement functions and only reports.

// measurePick is one picked entity, kept as a resolved analysis entity.
type measurePick struct{ entity analysis.MeasureEntity }

// MeasureTool reports measurements for the picked faces/edges/vertices.
type MeasureTool struct {
	picks   []measurePick
	readout string
}

// NewMeasureTool returns an empty measurement tool.
func NewMeasureTool() *MeasureTool { return &MeasureTool{} }

// Name is the tool's display name.
func (t *MeasureTool) Name() string { return "Measure" }

// Start lets the user pick any face, edge or vertex.
func (t *MeasureTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectFace, SelectEdge, SelectVertex))
}

// Pick adds an entity (starting a fresh selection when the pick can't extend the current one) and
// refreshes the readout.
func (t *MeasureTool) Pick(s *Session, sel Selectable) {
	p, ok := measurePickFrom(sel)
	if !ok {
		return
	}
	if !t.accepts(p) {
		t.picks = t.picks[:0]
	}
	t.picks = append(t.picks, p)
	t.readout = measureReadout(t.picks)
	s.SetNotice("Measure — " + t.readout)
}

// accepts reports whether p extends the current selection; a third pick is only meaningful as the
// final vertex of a three-vertex angle.
func (t *MeasureTool) accepts(p measurePick) bool {
	switch len(t.picks) {
	case 0, 1:
		return true
	case 2:
		return t.allVertices() && p.entity.Vertex != nil
	default:
		return false
	}
}

// allVertices reports whether every current pick is a vertex.
func (t *MeasureTool) allVertices() bool {
	for _, p := range t.picks {
		if p.entity.Vertex == nil {
			return false
		}
	}
	return true
}

// Readout is the current measurement text (empty before the first pick).
func (t *MeasureTool) Readout() string { return t.readout }

// Prompt guides the picking and echoes the current readout.
func (t *MeasureTool) Prompt(*Session) string {
	if t.readout == "" {
		return "Measure: pick a face, edge or vertex."
	}
	return "Measure — " + t.readout + "  (pick more, or Close)"
}

// CanCommit reports whether anything has been measured (OK simply closes the tool).
func (t *MeasureTool) CanCommit() bool { return len(t.picks) > 0 }

// Commit closes the tool, leaving the last readout in the status bar.
func (t *MeasureTool) Commit(s *Session) error {
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel clears the picks and the filter.
func (t *MeasureTool) Cancel(s *Session) {
	t.picks = nil
	t.readout = ""
	s.Selection().SetFilter(NewSelectionFilter())
}

// measurePickFrom resolves a selection handle to a measurement entity.
func measurePickFrom(sel Selectable) (measurePick, bool) {
	switch h := sel.(type) {
	case FaceHandle:
		return measurePick{analysis.MeasureEntity{Face: h.Face}}, true
	case EdgeHandle:
		return measurePick{analysis.MeasureEntity{Edge: h.Edge}}, true
	case VertexHandle:
		return measurePick{analysis.MeasureEntity{Vertex: h.Vertex}}, true
	}
	return measurePick{}, false
}

// measureReadout formats every applicable measurement for the current picks.
func measureReadout(picks []measurePick) string {
	q := ops.DefaultQuality()
	switch len(picks) {
	case 0:
		return "pick a face, edge or vertex."
	case 1:
		return singleReadout(picks[0].entity, q)
	case 2:
		return pairReadout(picks[0].entity, picks[1].entity, q)
	default:
		return tripleReadout(picks, q)
	}
}

// singleReadout reports the quantities of one entity: an edge's length, or a face's area + perimeter.
func singleReadout(e analysis.MeasureEntity, q ops.Quality) string {
	switch {
	case e.Edge != nil:
		return fmt.Sprintf("edge length %.3f mm", analysis.EdgeLengthMm(e.Edge, q))
	case e.Face != nil:
		return fmt.Sprintf("face area %.3f mm², perimeter %.3f mm",
			analysis.FaceAreaMm2(e.Face, q), analysis.FaceLoopLengthMm(e.Face, q))
	}
	return "vertex — pick another entity for a distance or angle."
}

// pairReadout reports the distance (two vertices) or minimum distance (any two entities), adding the
// angle when both entities have a direction (edge or face).
func pairReadout(a, b analysis.MeasureEntity, q ops.Quality) string {
	if a.Vertex != nil && b.Vertex != nil {
		return fmt.Sprintf("distance %.3f mm", analysis.VertexDistanceMm(a.Vertex, b.Vertex))
	}
	out := fmt.Sprintf("min distance %.3f mm", analysis.MinDistanceMm(a, b, q))
	if a.Vertex == nil && b.Vertex == nil {
		if deg, err := analysis.AngleDegrees(a, b, q); err == nil {
			out += fmt.Sprintf(", angle %.2f°", deg)
		}
	}
	return out
}

// tripleReadout reports the angle at the middle vertex of three vertices.
func tripleReadout(picks []measurePick, q ops.Quality) string {
	a, apex, c := picks[0].entity, picks[1].entity, picks[2].entity
	if a.Vertex != nil && apex.Vertex != nil && c.Vertex != nil {
		return fmt.Sprintf("angle %.2f° at the middle vertex",
			analysis.ThreePointAngleDegrees(a.Vertex, apex.Vertex, c.Vertex))
	}
	return pairReadout(a, apex, q)
}

// compile-time guard that the tool satisfies the interactive Tool + Prompted contracts.
var _ interface {
	Tool
	Prompted
} = (*MeasureTool)(nil)
