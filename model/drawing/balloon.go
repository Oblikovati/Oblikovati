// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	"math"
	"strconv"

	"oblikovati.org/api/types"
)

// Balloons (M14-F04 PBI-143, #390): a circle holding a parts-list item number, with an optional
// leader to the component it tags. The circle and leader are drawing curves; the item number is a
// text label. Balloons reference parts-list items by their item number.

// balloonRadiusMM is the balloon circle's radius.
const balloonRadiusMM = 5.0

// AddBalloon adds a balloon centred at (x, y) holding the parts-list item number, with an optional
// leader to (leaderX, leaderY) — the component it tags. A leader of (0, 0) means none.
func (as *DrawingAnnotations) AddBalloon(name string, x, y float64, item int, leaderX, leaderY float64) (*DrawingAnnotation, error) {
	if item <= 0 {
		return nil, fmt.Errorf("drawing: a balloon needs a positive item number, got %d", item)
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.BalloonAnnotation,
		x: x, y: y, w: leaderX, h: leaderY, tag: strconv.Itoa(item),
	}
	a.curves, a.labels = balloonGeometry(x, y, item, leaderX, leaderY)
	as.items = append(as.items, a)
	return a, nil
}

// balloonGeometry builds the balloon's circle + optional leader (drawing curves) and the item
// number label centred in the circle.
func balloonGeometry(x, y float64, item int, leaderX, leaderY float64) ([]DrawingCurve, []AnnotationLabel) {
	curves := circlePolyline(x, y, balloonRadiusMM)
	if leaderX != 0 || leaderY != 0 {
		// Start the leader on the circle's edge nearest the target, not its centre.
		dx, dy := leaderX-x, leaderY-y
		if d := math.Hypot(dx, dy); d > 1e-9 {
			sx, sy := x+dx/d*balloonRadiusMM, y+dy/d*balloonRadiusMM
			curves = append(curves, dimSegment(sx, sy, leaderX, leaderY))
		}
	}
	return curves, []AnnotationLabel{{Text: strconv.Itoa(item), X: x, Y: y}}
}
