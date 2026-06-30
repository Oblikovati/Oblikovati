// SPDX-License-Identifier: GPL-2.0-only

// Package gridlayout is the pure, headless-testable math behind the CSS-grid-like panel
// container (ADR-0019, ADR-0020): resolving column-track pixel widths against an available
// width, and flowing children into rows. It is deliberately free of cgo/ImGui so the
// algorithm — where the bugs live — is unit-tested without a windowed build; the cgo
// renderer in package ui measures auto-column content (via CalcTextWidth), then calls this.
package gridlayout

import "oblikovati.org/api/types"

// defaultAutoWidth is the fallback width for an auto column whose content could not be
// measured (the renderer passes 0). Wide enough for a short label.
const defaultAutoWidth = 80.0

// ResolveColumnWidths returns the pixel width of each column track. Fixed tracks take their
// Value; auto tracks take their measured content width (autoWidths[i], or defaultAutoWidth
// when 0); fraction tracks split the leftover space — after fixed/auto and the inter-column
// gaps are removed — in proportion to their weight. Every width is clamped to the track's
// [MinPx, MaxPx] (0 = unbounded) and floored at 0. autoWidths may be nil.
func ResolveColumnWidths(cols []types.GridTrack, avail, colGap float64, autoWidths []float64) []float64 {
	n := len(cols)
	widths := make([]float64, n)
	if n == 0 {
		return widths
	}
	leftover := avail - colGap*float64(n-1)
	frWeight := resolveSizedTracks(cols, autoWidths, widths, &leftover)
	if frWeight > 0 {
		resolveFractionTracks(cols, widths, leftover, frWeight)
	}
	for i := range widths {
		if widths[i] < 0 {
			widths[i] = 0
		}
	}
	return widths
}

// resolveSizedTracks fills the fixed and auto column widths (subtracting each from leftover)
// and returns the total fraction weight left for resolveFractionTracks to distribute.
func resolveSizedTracks(cols []types.GridTrack, autoWidths, widths []float64, leftover *float64) float64 {
	var frWeight float64
	for i, t := range cols {
		switch t.Kind {
		case types.GridTrackFixed:
			widths[i] = clampTrack(t, t.Value)
			*leftover -= widths[i]
		case types.GridTrackAuto:
			widths[i] = clampTrack(t, autoOrDefault(autoWidths, i))
			*leftover -= widths[i]
		case types.GridTrackFraction:
			frWeight += fractionWeight(t)
		}
	}
	return frWeight
}

// resolveFractionTracks splits leftover across the fraction columns by weight.
func resolveFractionTracks(cols []types.GridTrack, widths []float64, leftover, frWeight float64) {
	for i, t := range cols {
		if t.Kind == types.GridTrackFraction {
			widths[i] = clampTrack(t, leftover*fractionWeight(t)/frWeight)
		}
	}
}

// autoOrDefault is the measured width of auto column i, or defaultAutoWidth when unmeasured.
func autoOrDefault(autoWidths []float64, i int) float64 {
	if autoWidths != nil && autoWidths[i] > 0 {
		return autoWidths[i]
	}
	return defaultAutoWidth
}

// fractionWeight reads a fraction track's weight, treating a non-positive weight as 1 so a
// bare TrackFr()/auto-flex column still claims an equal share.
func fractionWeight(t types.GridTrack) float64 {
	if t.Value <= 0 {
		return 1
	}
	return t.Value
}

// clampTrack bounds w to the track's [MinPx, MaxPx] (0 = unset).
func clampTrack(t types.GridTrack, w float64) float64 {
	if t.MinPx > 0 && w < t.MinPx {
		w = t.MinPx
	}
	if t.MaxPx > 0 && w > t.MaxPx {
		w = t.MaxPx
	}
	return w
}

// Placement is one child positioned in a grid row: which child (index into the original
// children slice), its starting column, and how many columns it spans.
type Placement struct {
	Child   int
	Col     int
	ColSpan int
}

// FlowRows assigns each child to a row and column. cells[i] is child i's optional placement
// (nil = auto-flow). Auto-flow fills left-to-right, wrapping when the next child's span would
// overrun nCols. A cell with ColSpan>1 keeps its span (wrapping to a fresh row if it doesn't
// fit). A cell with an explicit Col>0 is placed at that column on its own row — the common
// "put this at column k" case — without the bookkeeping of arbitrary 2-D packing (deferred
// with row span, ADR-0020).
func FlowRows(cells []*types.GridCell, nCols int) [][]Placement {
	if nCols < 1 {
		nCols = 1
	}
	f := &rowFlow{cols: nCols}
	for i := range cells {
		span, explicitCol := readCell(cells[i], nCols)
		f.place(i, span, explicitCol)
	}
	f.flush()
	return f.rows
}

// rowFlow accumulates placements into rows as children are visited left-to-right.
type rowFlow struct {
	rows [][]Placement
	cur  []Placement
	col  int
	cols int
}

// flush ends the current row if it holds anything.
func (f *rowFlow) flush() {
	if len(f.cur) > 0 {
		f.rows = append(f.rows, f.cur)
		f.cur = nil
		f.col = 0
	}
}

// place positions child i: an explicit column takes its own row; otherwise the child auto-flows
// into the current row, wrapping when its span no longer fits.
func (f *rowFlow) place(child, span, explicitCol int) {
	if explicitCol >= 0 {
		f.flush()
		f.rows = append(f.rows, []Placement{{Child: child, Col: clampCol(explicitCol, span, f.cols), ColSpan: span}})
		return
	}
	if f.col+span > f.cols {
		f.flush()
	}
	f.cur = append(f.cur, Placement{Child: child, Col: f.col, ColSpan: span})
	f.col += span
	if f.col >= f.cols {
		f.flush()
	}
}

// readCell extracts (span, explicitCol) from a cell. span is clamped to [1, nCols]. An
// explicit column is only honored when Col>0 (Col 0 is indistinguishable from auto-flow's
// natural start and so flows); explicitCol is -1 when the child should auto-flow.
func readCell(c *types.GridCell, nCols int) (span, explicitCol int) {
	span, explicitCol = 1, -1
	if c == nil {
		return
	}
	if c.ColSpan > 1 {
		span = c.ColSpan
	}
	if span > nCols {
		span = nCols
	}
	if c.Col > 0 {
		explicitCol = c.Col
	}
	return
}

// clampCol keeps an explicit column within the grid so col+span never overruns nCols.
func clampCol(col, span, nCols int) int {
	if col+span > nCols {
		col = nCols - span
	}
	if col < 0 {
		col = 0
	}
	return col
}
