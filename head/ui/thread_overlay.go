// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// threadColor: a muted gold so a cosmetic thread's helix reads distinctly from the body edges.
var threadColor = [4]float32{0.85, 0.7, 0.3, 1}

// threadOverlay draws each cosmetic thread as a helix on its cylindrical face — the viewport
// display of Inventor's thread feature (the solid is unchanged; this is the thread graphic).
func threadOverlay(s *app.Session) []renderer.DrawItem {
	part := activePart(s)
	if part == nil {
		return nil
	}
	curves := feature.ThreadDisplayCurves(part.Features())
	if len(curves) == 0 {
		return nil
	}
	acc := &segAccum{}
	for _, c := range curves {
		for i := 0; i+1 < len(c); i++ {
			acc.addSegment(c[i], c[i+1])
		}
	}
	if len(acc.pos) == 0 {
		return nil
	}
	return []renderer.DrawItem{lineItem(acc, threadColor)}
}
