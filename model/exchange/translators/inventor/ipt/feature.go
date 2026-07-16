// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

// Feature decoding from PmDCSegment. Inventor names features ("Extrusion1",
// "Revolution1", ...) as UTF-16 nodes; a feature's dimension (e.g. an extrude's
// distance) is held by an auto model parameter it references.

// Boolean operation values as authored in PmDCSegment (an operation node:
// dcMarker + u16(0x0005) + u16(op) + 0x00000026). Order is Inventor-file-specific.
const (
	OpNewBody   = 1
	OpCut       = 2
	OpJoin      = 3
	OpIntersect = 4
)

// enumTrailer marks an enum node's payload end: `26 00 00 00` at node+20.
const enumTrailer = 0x00000026

// enumNodeValues returns the value of each enum node of the given kind, in feature order.
// An enum node is a nameless graph node whose payload is a small (kind, value) pair:
//
//	0x80000003(dcNodeTag) | id | 0xFFFFFFFF(nullRef) | schemaMarker | kind:u16@+16 value:u16@+18 | 0x26@+20
//
// Inventor uses this one shape for several small enums — kind 5 is a feature's boolean
// operation, kind 3 a hole's type. It is anchored on the node structure and the 0x26 trailer,
// NOT on the +12 schema marker, which is 0x0C20 in API-authored corpus parts but varies in
// real interactively-modelled parts (0x0A96, 0x0B5C, …) — keying on 0x0C20 found zero
// operation nodes on real parts, so no extrude/revolve could bind its boolean op.
func enumNodeValues(seg []byte, kind uint16) []int {
	var out []int
	for i := 0; i+24 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag ||
			binary.LittleEndian.Uint32(seg[i+8:]) != nullRef ||
			binary.LittleEndian.Uint32(seg[i+20:]) != enumTrailer {
			continue
		}
		if binary.LittleEndian.Uint16(seg[i+16:]) == kind {
			out = append(out, int(binary.LittleEndian.Uint16(seg[i+18:])))
		}
	}
	return out
}

// DecodeOperations returns each feature's boolean operation (OpNewBody/Cut/Join/
// Intersect) in feature order — the kind-5 0x0C20 enum nodes.
func DecodeOperations(seg []byte) []int {
	return enumNodeValues(seg, 5)
}

// Extrude is a decoded extrude feature: its distance in cm and boolean operation.
type Extrude struct {
	Distance  float64
	Operation int
}

// DecodeExtrude reports the part's first extrude feature, if present.
func DecodeExtrude(d *Document) (Extrude, bool) {
	if ex := DecodeExtrudes(d); len(ex) > 0 {
		return ex[0], true
	}
	return Extrude{}, false
}

// Inventor unit-type ids on a model parameter, at valueOffset+20 (the value f64, its duplicate,
// `00 00 ff ff`, then this byte). They distinguish an angle (radians) from a length (cm) — the
// distinction RevolveAngle needs so a shaft's leading profile LENGTH is not read as a sweep angle.
const (
	angleUnitID  = 0x4a
	lengthUnitID = 0x46
)

// modelParam is a decoded auto "d…" model parameter: its value (database units) and unit-type id.
type modelParam struct {
	value float64
	unit  byte
}

// modelParams returns every model parameter (auto "d…" dimension params) in order, each with its
// Inventor unit-type id. Model-param nodes share the user-parameter framing but carry a
// non-identifier name (a null byte). The unit sits 20 bytes past the value.
func modelParams(seg []byte) []modelParam {
	var out []modelParam
	forEachNamefulNode(seg, func(rawName []byte, valueStart int) {
		if isIdentifier(string(rawName)) || len(rawName) == 0 || rawName[0] != 'd' {
			return
		}
		if v, at, ok := firstNonZeroDoubleAt(seg, valueStart, 48); ok {
			unit := byte(0)
			if at+20 < len(seg) {
				unit = seg[at+20]
			}
			out = append(out, modelParam{value: v, unit: unit})
		}
	})
	return out
}

// modelParamValues returns every model parameter value in order — one per sketched feature's
// driving dimension.
func modelParamValues(seg []byte) []float64 {
	mp := modelParams(seg)
	out := make([]float64, len(mp))
	for i, p := range mp {
		out[i] = p.value
	}
	return out
}

// DecodeExtrudes returns every extrude feature in order, each bound to ITS OWN distance and
// operation — both resolved through the feature node's own property references, not by position.
// The predecessor took the i-th model parameter as the distance and the i-th enum node as the
// operation; on any part with more than one of either, those are unrelated: BigChunkyPlate extruded
// a 3 cm plate by 40 cm (26x its true volume). Empty when the part has no extrude.
func DecodeExtrudes(d *Document) []Extrude {
	nodes := dcNodes(d)
	var out []Extrude
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		dist, ok := extrudeDepth(nodes, n.payload)
		if !ok {
			continue
		}
		op, ok := extrudeOperation(nodes, n.payload)
		if !ok {
			op = OpNewBody // a feature naming no operation starts a body
		}
		out = append(out, Extrude{Distance: dist, Operation: op})
	}
	return out
}

// HasRevolve reports whether the part has a revolve feature ("Revolution" node).
func HasRevolve(seg []byte) bool {
	return containsUTF16(seg, "Revolution")
}

// RevolveAxisLine returns the index in s.Lines of the revolve centreline: the isolated line
// whose two endpoints are shared with no other line, while the profile it turns about is a
// closed loop (every vertex degree 2). A real revolve sketch draws the axis as a separate
// centreline (e.g. a shaft's (0,0)-(0,h) line at x=0), so revolving about it — rather than the
// fixed X origin axis — is what turns the profile into the right solid instead of a blob.
// ok=false when no single line is unambiguously the axis; callers then fall back to a work axis.
func RevolveAxisLine(s Sketch) (int, bool) {
	deg := map[[2]int64]int{}
	for _, l := range s.Lines {
		deg[endpointKey(l.A)]++
		deg[endpointKey(l.B)]++
	}
	axis, found := -1, 0
	for i, l := range s.Lines {
		if deg[endpointKey(l.A)] == 1 && deg[endpointKey(l.B)] == 1 {
			axis, found = i, found+1
		}
	}
	return axis, found == 1
}

// RevolveBinding names the sketch to revolve and the centreline to turn it about (which may live
// in a different sketch), resolved by RevolveProfile. AxisSketch may equal ProfileSketch.
type RevolveBinding struct {
	ProfileSketch int // index into the sketch slice
	AxisSketch    int // index of the sketch holding the centreline
	AxisLine      int // index into AxisSketch's Lines
}

// RevolveProfile selects which sketch to revolve and the centreline to turn it about, and —
// crucially — validates that the profile is a genuine CLOSED, one-sided loop. A decoded profile
// that is merely Resolved can still be an incomplete open chain: a segment gets dropped or split
// into a neighbouring 800-byte cluster, yet the reference/point counts happen to balance so
// resolveByRefs reports success. Revolving that open chain builds a plausible-but-wrong solid
// (the ReelToReel shafts came out ~2x too large), which is worse than falling back to the mesh —
// so closure, not mere resolution, is the real gate (tessellation-correctness is the #1 priority).
//
// Two axis encodings occur: (A) the centreline sits inside the profile sketch as a line isolated
// from the loop (a shaft's x=0 line); (B) the profile is a clean closed loop and the centreline is
// a separate single-line sketch (the API corpus draws it this way). In both, the profile ring must
// lie wholly on one side of the axis. ok=false ⇒ no sketch pair forms a valid revolve and the
// caller must fall back to the mesh body rather than emit a wrong revolve.
func RevolveProfile(sketches []Sketch) (b RevolveBinding, ok bool) {
	for i, s := range sketches {
		if !s.Resolved || len(s.Lines) < 3 {
			continue
		}
		// Case A: a VERTICAL centreline inside the profile sketch (≥4 lines: loop + axis). The
		// vertical requirement is the standard upright-shaft encoding; it rejects an ambiguous
		// horizontal isolated line (seen picked wrongly on a disk part, revolving the profile the
		// wrong way) that would otherwise pass — safer to fall back than to guess the axis.
		if ai, found := RevolveAxisLine(s); found && len(s.Lines) >= 4 && isVerticalLine(s.Lines[ai]) {
			loop := append(append([]Line(nil), s.Lines[:ai]...), s.Lines[ai+1:]...)
			if isClosedRing(loop) && oneSideOfAxis(loop, s.Lines[ai]) {
				return RevolveBinding{i, i, ai}, true
			}
		}
		if !isClosedRing(s.Lines) {
			continue
		}
		xi, hasSep := separateAxisSketch(sketches, i, s.Lines)
		ci, hasEdge := axisEdgeOnX(s.Lines)
		onEdge := hasEdge && oneSideOfAxis(s.Lines, s.Lines[ci])
		// Ambiguous axis: the profile offers TWO different centrelines — its own edge on x≈0 AND a
		// separate sketch line that is not collinear with it (a non-vertical, off-axis centreline).
		// The intended axis (and, for such parts, the sweep angle) can't be inferred from geometry —
		// 1677K262 is a partial revolve about a horizontal centreline whose angle lives in no readable
		// parameter; either axis choice, swept full, is a wrong solid. Decline to the mesh instead.
		if hasSep && onEdge && !onXAxis(sketches[xi].Lines[0]) {
			continue
		}
		// Case B: the centreline is a separate single-line sketch the profile is strictly OFFSET from
		// (does not touch). The offset requirement rejects a stray lone line that merely bounds the
		// profile (a diameter/annotation line at its far edge), which otherwise revolves it into a
		// giant torus. Checked before case C so a dedicated centreline (any orientation) wins over an
		// incidental x≈0 profile edge.
		if hasSep {
			return RevolveBinding{i, xi, 0}, true
		}
		// Case C: a solid shaft drawn touching its centreline — the closed profile has a single edge
		// ON the sketch's vertical axis (x≈0) and lies wholly to one side of it; revolve about that
		// edge. This is the common shaft encoding (LeverShaft, PressureRollerMainShaft) where the
		// axis is the profile's own boundary, so there is no isolated centreline or separate sketch.
		if onEdge {
			return RevolveBinding{i, i, ci}, true
		}
	}
	return RevolveBinding{}, false
}

// ReuniteRevolveAxis merges a revolve's separate VERTICAL centreline sketch back into its profile
// sketch, so both live in one sketch — as Inventor authored them, where the centreline is
// construction geometry inside the profile's own sketch. Incidence decoding splits them because the
// centreline shares no endpoint with the profile loop (a disconnected component), landing it in its
// own sketch; but a dimension that positions a profile edge relative to the centreline (a revolve's
// radius dimension) can only bind when both live in ONE sketch. Reuniting also moves the part from
// RevolveProfile case B to case A (an in-profile vertical centreline) with the identical axis line,
// so the rebuilt solid is unchanged. Only a vertical centreline is merged (case A requires it); a
// non-vertical separate centreline is left as its own sketch for case B.
func ReuniteRevolveAxis(sketches []Sketch) []Sketch {
	b, ok := RevolveProfile(sketches)
	if !ok || b.AxisSketch == b.ProfileSketch {
		return sketches
	}
	axis := sketches[b.AxisSketch].Lines[b.AxisLine]
	if !isVerticalLine(axis) {
		return sketches
	}
	out := make([]Sketch, 0, len(sketches))
	for i, s := range sketches {
		if i == b.AxisSketch {
			continue // its line moves into the profile sketch; drop the now-empty axis sketch
		}
		if i == b.ProfileSketch {
			s.Lines = append(append([]Line(nil), s.Lines...), axis)
		}
		out = append(out, s)
	}
	return out
}

// onXAxis reports whether a line lies on the sketch's vertical axis (both endpoints x≈0) — used to
// tell a separate centreline that coincides with a profile's x≈0 edge (same axis, unambiguous)
// from one running elsewhere (a conflicting axis).
func onXAxis(l Line) bool {
	return math.Abs(l.A.X) < 1e-4 && math.Abs(l.B.X) < 1e-4
}

// axisEdgeOnX returns the index of the single profile line lying on the sketch's vertical axis
// (both endpoints x≈0) — the edge a shaft profile drawn touching its centreline turns about.
// found=false unless exactly one edge is on the axis (two would make the axis ambiguous).
func axisEdgeOnX(lines []Line) (int, bool) {
	idx, found := -1, 0
	for i, l := range lines {
		if math.Abs(l.A.X) < 1e-4 && math.Abs(l.B.X) < 1e-4 {
			idx, found = i, found+1
		}
	}
	return idx, found == 1
}

// separateAxisSketch finds a sketch (other than the profile at pi) that is a lone centreline —
// exactly one line and no other geometry — that the profile lies wholly on one side of AND is
// strictly offset from (no vertex within a hair of the axis line). Returns its index. The offset
// test is what distinguishes a genuine external centreline from a boundary/annotation line that
// coincides with the profile's edge.
func separateAxisSketch(sketches []Sketch, pi int, profile []Line) (int, bool) {
	for xi, t := range sketches {
		if xi == pi || len(t.Lines) != 1 || len(t.Circles) != 0 || len(t.Arcs) != 0 {
			continue
		}
		if oneSideOfAxis(profile, t.Lines[0]) && offsetFromAxis(profile, t.Lines[0]) {
			return xi, true
		}
	}
	return 0, false
}

// isVerticalLine reports whether a line runs along the sketch's Y axis (endpoints share X) — the
// orientation of an upright shaft's centreline.
func isVerticalLine(l Line) bool { return math.Abs(l.A.X-l.B.X) < 1e-4 }

// offsetFromAxis reports whether every profile vertex is clear of the infinite line through the
// axis segment by more than 10 microns — i.e. the profile does not touch the axis.
func offsetFromAxis(profile []Line, axis Line) bool {
	dx, dy := axis.B.X-axis.A.X, axis.B.Y-axis.A.Y
	n := math.Hypot(dx, dy)
	if n < 1e-9 {
		return false
	}
	for _, l := range profile {
		for _, p := range [2]Point2D{l.A, l.B} {
			d := math.Abs(dx*(p.Y-axis.A.Y)-dy*(p.X-axis.A.X)) / n
			if d < 1e-3 {
				return false
			}
		}
	}
	return true
}

// isClosedRing reports whether the (unordered) lines form exactly one closed ring — every line
// consumed by walking shared endpoints from an arbitrary start back to it. An open chain, a
// figure-eight, or two disjoint loops all return false. This is what distinguishes a complete
// revolve profile from an incomplete one whose reference counts merely balanced.
func isClosedRing(lines []Line) bool {
	if len(lines) < 3 {
		return false
	}
	used := map[int]bool{0: true}
	start, tail := lines[0].A, lines[0].B
	for len(used) < len(lines) {
		progressed := false
		for i, l := range lines {
			if used[i] {
				continue
			}
			switch {
			case samePoint2D(l.A, tail):
				tail = l.B
			case samePoint2D(l.B, tail):
				tail = l.A
			default:
				continue
			}
			used[i], progressed = true, true
			break
		}
		if !progressed {
			return false
		}
	}
	return samePoint2D(tail, start)
}

// oneSideOfAxis reports whether every profile vertex lies on one side of (or on) the infinite
// line through the axis segment. A valid solid of revolution never crosses its axis; a profile
// straddling the axis (a symptom of a mis-picked centreline) would sweep a self-intersecting body.
func oneSideOfAxis(profile []Line, axis Line) bool {
	dx, dy := axis.B.X-axis.A.X, axis.B.Y-axis.A.Y
	sign := 0
	for _, l := range profile {
		for _, p := range [2]Point2D{l.A, l.B} {
			cross := dx*(p.Y-axis.A.Y) - dy*(p.X-axis.A.X)
			if cross > 1e-7 {
				if sign < 0 {
					return false
				}
				sign = 1
			} else if cross < -1e-7 {
				if sign > 0 {
					return false
				}
				sign = -1
			}
		}
	}
	return true
}

// samePoint2D reports whether two sketch points coincide to within a micron.
func samePoint2D(a, b Point2D) bool {
	return math.Abs(a.X-b.X) < 1e-4 && math.Abs(a.Y-b.Y) < 1e-4
}

// endpointKey quantises a sketch point to a micron so that lines sharing a vertex hash equal.
func endpointKey(p Point2D) [2]int64 {
	return [2]int64{int64(p.X * 1e4), int64(p.Y * 1e4)}
}

// revolveExtentKind is the feature extent-type enum kind (shared by extrude and revolve);
// revolveFullExtent is its value for a FULL (360°) revolve. A partial revolve stores 1 (an angle
// extent), an extrude 1 (a distance), a hole 5 (through-all) — only a full revolve, or a sweep,
// stores 3. It is the same 0x26-trailer enum shape as the boolean operation (kind 5) and hole type
// (kind 3), decoded here per-part (single-revolve). Values verified on a single-variable corpus
// (revolve at 80/150/220°/full) and on real parts (ReelMotorBearingShaft/TorquimeterShaft = 3).
const (
	revolveExtentKind = 12
	revolveFullExtent = 3
)

// HasFullRevolveExtent reports whether the part carries a full-revolve extent enum — the feature's
// OWN sweep flag (kind=12 value=3), the authoritative full/partial signal. It is trusted only when
// the part has a revolve and no sweep (a sweep also stores extent 3). Some real parts don't expose
// the enum at all (PressureRollerMainShaft) — then this is false and RevolveAngle's param heuristic
// decides — so it is used ADDITIVELY: it can confirm FULL, never force partial.
func HasFullRevolveExtent(seg []byte) bool {
	if !HasRevolve(seg) || containsUTF16(seg, "Sweep") {
		return false
	}
	for _, v := range enumNodeValues(seg, revolveExtentKind) {
		if v == revolveFullExtent {
			return true
		}
	}
	return false
}

// RevolveAngle returns the revolve's sweep angle in radians when it is partial. The feature's own
// extent flag is authoritative when present: a FULL revolve stores extent 3 (HasFullRevolveExtent),
// so no sweep angle is read for it even if the profile carries an angle DIMENSION (a chamfer's 30°)
// that IS an angle parameter but NOT the sweep — the case that made a param scan alone turn a full
// revolve into a sliver. When the extent enum is absent (only some real parts expose it) the angle
// comes from soleSweepAngle: reported only when exactly one angle parameter exists, so the API
// corpus's lone driving param binds (24_revolve_270 → 3π/2) while a real part that reuses the angle
// unit for many dimensions stays FULL (ok=false) — a wrong partial is worse than the mesh.
func RevolveAngle(seg []byte) (float64, bool) {
	if !HasRevolve(seg) || HasFullRevolveExtent(seg) {
		return 0, false
	}
	return soleSweepAngle(seg)
}

// soleSweepAngle finds the revolve's partial sweep angle by the on-disk shape of an angle
// parameter — a float64 in (0, 2π) immediately followed by an identical nominal copy, with the
// angle unit id (angleUnitID) at +20 from the value. It returns the value only when EXACTLY ONE
// such parameter exists (a single-revolve part's driving dimension). More than one is ambiguous —
// a real part tags lengths with the same unit id — so it yields ok=false and the caller sweeps
// full. This byte-shape scan is more robust than the "d"-named model-param reader, which mis-picked
// a stray near-zero double ahead of the real angle on some parts (a 150° revolve read as 0).
func soleSweepAngle(seg []byte) (float64, bool) {
	var angle float64
	n := 0
	for o := 0; o+21 <= len(seg); o++ {
		v := f64(seg, o)
		if v > 1e-3 && v < 2*math.Pi && v == f64(seg, o+8) && seg[o+20] == angleUnitID {
			angle, n = v, n+1
		}
	}
	if n != 1 {
		return 0, false
	}
	return angle, true
}

// containsUTF16 reports whether the UTF-16LE encoding of s occurs in seg.
func containsUTF16(seg []byte, s string) bool {
	u := utf16.Encode([]rune(s))
	pat := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(pat[2*i:], c)
	}
	return indexOf(seg, pat) >= 0
}

func indexOf(hay, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if string(hay[i:i+len(needle)]) == string(needle) {
			return i
		}
	}
	return -1
}

const mathPi = 3.141592653589793
