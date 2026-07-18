// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "math"

// Mirror feature decoding from PmDCSegment. A mirror reflects a feature across a plane. The
// plane is a Plane node (CE52DF42) carrying a centre point plus two in-plane axes; the
// stored "normal" slot is actually a second in-plane axis, so v1 infers the reflection
// normal from the mirror work-plane's centre — an axis-aligned offset plane's centre lies on
// its normal axis (a plane at x=3 has centre (3,0,0) → normal +X). Mirrors across an origin
// plane (centre at the origin) or a tilted plane are future work.

// Mirror is a decoded mirror plane: a point on it (cm) and its unit normal.
type Mirror struct {
	Origin [3]float64
	Normal [3]float64
}

// planeNodeType is the PmDCSegment Plane node (InventorLoader Read_CE52DF42), whose payload
// contains the centre + two in-plane axis vectors (3 float64 each, not 8-byte aligned).
const planeNodeType = 0xCE52DF42

// DecodeMirror reports the part's mirror plane, if the part has a mirror feature. v1 finds
// the offset work plane (the one Plane node whose centre is off the origin) and takes its
// normal as the axis of that offset. Returns ok=false when there is no mirror or no offset
// plane (e.g. a mirror across an origin plane — not yet supported).
func DecodeMirror(d *Document) (Mirror, bool) {
	seg, ok := d.Segment("PmDCSegment")
	if !ok || !containsUTF16(seg, "Mirror") || containsUTF16(seg, "Pattern") {
		return Mirror{}, false
	}
	var found Mirror
	var have bool
	d.walkSegment("PmDCSegment", func(typ uint32, pay []byte) bool {
		if typ != planeNodeType {
			return true
		}
		if c, okc := planeCentre(pay); okc && absf(c[0])+absf(c[1])+absf(c[2]) > 1e-6 {
			found = Mirror{Origin: c, Normal: dominantAxis(c)}
			have = true
			return false // the first offset plane is the mirror plane
		}
		return true
	})
	return found, have
}

// planeCentre reads a Plane node's centre point: the payload holds centre + two in-plane
// axis vectors consecutively (3 float64 each), the two axes being unit vectors. Scans every
// byte offset (the doubles are not 8-byte aligned) for that signature.
func planeCentre(pay []byte) ([3]float64, bool) {
	for o := 0; o+72 <= len(pay); o++ {
		if isUnitVec(pay, o+24) && isUnitVec(pay, o+48) {
			return [3]float64{f64(pay, o), f64(pay, o+8), f64(pay, o+16)}, true
		}
	}
	return [3]float64{}, false
}

// isUnitVec reports whether the 3 float64 at o form a unit vector.
func isUnitVec(b []byte, o int) bool {
	if o+24 > len(b) {
		return false
	}
	x, y, z := f64(b, o), f64(b, o+8), f64(b, o+16)
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		return false
	}
	return absf(math.Sqrt(x*x+y*y+z*z)-1) < 1e-9
}

// dominantAxis returns the +unit axis of c's largest-magnitude coordinate — the reflection
// normal of an axis-aligned offset plane whose centre is c.
func dominantAxis(c [3]float64) [3]float64 {
	ax, ay, az := absf(c[0]), absf(c[1]), absf(c[2])
	switch {
	case ax >= ay && ax >= az:
		return [3]float64{1, 0, 0}
	case ay >= az:
		return [3]float64{0, 1, 0}
	default:
		return [3]float64{0, 0, 1}
	}
}
