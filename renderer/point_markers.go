// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "oblikovati.org/math"

// Point-cloud display (M17-F06, #645). The renderer has no native point primitive, so a cloud's
// points render as a batch of small 3-axis crosses on the existing Lines pipeline — visible from
// any view angle and reusing the line shader. The head supplies the (already budgeted, model-
// space) points and a marker size; this headless builder turns them into one draw item, so the
// geometry-as-data stays unit-testable. A native point primitive is a later performance pass.

// PointCloudColor is the default marker color: a bright cyan that reads clearly against both
// light and dark viewport backgrounds.
var PointCloudColor = [4]float32{0.25, 0.85, 1.0, 1}

// PointMarkers builds a Lines draw item rendering each point as a 3-axis cross of the given world
// size, in color, tagged with objectID. Returns nil when there are no points or size <= 0.
func PointMarkers(points []math.Point3, size float64, color [4]float32, objectID uint64) *DrawItem {
	if len(points) == 0 || size <= 0 {
		return nil
	}
	h := math.Scalar(size / 2)
	item := DrawItem{Primitive: Lines, Color: color, ObjectID: objectID}
	item.Positions = make([]math.Point3, 0, len(points)*6)
	item.Indices = make([]int, 0, len(points)*6)
	for _, p := range points {
		base := len(item.Positions)
		item.Positions = append(item.Positions,
			math.P3(p.X-h, p.Y, p.Z), math.P3(p.X+h, p.Y, p.Z),
			math.P3(p.X, p.Y-h, p.Z), math.P3(p.X, p.Y+h, p.Z),
			math.P3(p.X, p.Y, p.Z-h), math.P3(p.X, p.Y, p.Z+h),
		)
		item.Indices = append(item.Indices, base, base+1, base+2, base+3, base+4, base+5)
	}
	return &item
}
