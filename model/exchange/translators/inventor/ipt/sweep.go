// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "math"

// Sweep feature decoding from PmDCSegment. A sweep pushes a profile along a path curve. v1
// handles a circular profile swept along a straight-segment polyline OR a circular-arc path
// sketched on the XZ plane — the path sketch's 2D coordinates (u, v) map to model space
// (u, 0, v). Non-circular profiles, spline paths, and other path planes are future work.

// Sweep is a decoded sweep: a circular profile and the ordered path polyline (in the path
// sketch's 2D coordinates).
type Sweep struct {
	Profile Circle
	Path    []Point2D
}

// HasSweep reports whether the part has a sweep feature ("Sweep" node).
func HasSweep(seg []byte) bool {
	return containsUTF16(seg, "Sweep")
}

// DecodeSweep reports the part's sweep, if present. v1: the circle among the decoded sketch
// geometry is the profile, and the lines are the path (chained into an ordered polyline).
func DecodeSweep(seg []byte) (Sweep, bool) {
	if !HasSweep(seg) {
		return Sweep{}, false
	}
	var circles []Circle
	var lines []Line
	var arcs []Arc
	for _, s := range DecodeSketches(seg) {
		circles = append(circles, s.Circles...)
		lines = append(lines, s.Lines...)
		arcs = append(arcs, s.Arcs...)
	}
	if len(circles) == 0 {
		return Sweep{}, false
	}
	var path []Point2D
	switch {
	case len(arcs) > 0:
		path = tessellateArc(arcs[0], 32)
	case len(lines) > 0:
		path = chainPath(lines)
	}
	if len(path) < 2 {
		return Sweep{}, false
	}
	return Sweep{Profile: circles[0], Path: path}, true
}

// tessellateArc samples the minor arc into n+1 points, ordered from the endpoint nearest the
// origin (the profile sits at the path start) around the centre to the far endpoint.
func tessellateArc(a Arc, n int) []Point2D {
	start, end := a.Start, a.End
	if end.X*end.X+end.Y*end.Y < start.X*start.X+start.Y*start.Y {
		start, end = end, start
	}
	a0 := math.Atan2(start.Y-a.Center.Y, start.X-a.Center.X)
	a1 := math.Atan2(end.Y-a.Center.Y, end.X-a.Center.X)
	da := a1 - a0
	for da > math.Pi {
		da -= 2 * math.Pi
	}
	for da <= -math.Pi {
		da += 2 * math.Pi
	}
	pts := make([]Point2D, n+1)
	for i := 0; i <= n; i++ {
		t := a0 + da*float64(i)/float64(n)
		pts[i] = Point2D{a.Center.X + a.Radius*math.Cos(t), a.Center.Y + a.Radius*math.Sin(t)}
	}
	return pts
}

// chainPath orders the path's line segments into a single polyline, starting from the open
// end nearest the origin (the profile sits at the path start). Returns nil if the segments
// do not form one open chain.
func chainPath(lines []Line) []Point2D {
	segs := make([][2]Point2D, len(lines))
	for i, l := range lines {
		segs[i] = [2]Point2D{l.A, l.B}
	}
	degree := func(p Point2D) int {
		n := 0
		for _, s := range segs {
			if samePt(s[0], p) {
				n++
			}
			if samePt(s[1], p) {
				n++
			}
		}
		return n
	}
	start, found, best := Point2D{}, false, 1e18
	for _, s := range segs {
		for _, p := range s {
			if degree(p) == 1 {
				if d := p.X*p.X + p.Y*p.Y; d < best {
					best, start, found = d, p, true
				}
			}
		}
	}
	if !found {
		return nil
	}
	used := make([]bool, len(segs))
	path, cur := []Point2D{start}, start
	for {
		moved := false
		for i, s := range segs {
			if used[i] {
				continue
			}
			switch {
			case samePt(s[0], cur):
				cur = s[1]
			case samePt(s[1], cur):
				cur = s[0]
			default:
				continue
			}
			used[i], moved = true, true
			path = append(path, cur)
			break
		}
		if !moved {
			return path
		}
	}
}

func samePt(a, b Point2D) bool { return absf(a.X-b.X) < 1e-9 && absf(a.Y-b.Y) < 1e-9 }
