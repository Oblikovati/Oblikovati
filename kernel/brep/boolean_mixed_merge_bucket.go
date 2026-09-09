// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// What a merge scan needs to know about a face BEFORE it starts pairing (#3523).
//
// The scan is O(F²) pair tests and it built a PER-FACE value inside that loop: the loop box.
// geom.ResolutionForBox(faceLoopBox(a)) is the resolution onOneSurface hands to SurfacesCoincide, and
// faceLoopBox walks every edge of every loop of a and takes a span box per edge — for a value that
// depends on a alone. Measured on the fine-pitch coil join (#879's 2 mm pitch on a Ø6 core), one
// mergeCoincidentFaces call over 1541 faces built that box 1,188,105 times — 771 per face — and those
// builds were 88% of the call. Tabulating it per scan takes the count to 4,620 (three scans over
// 1541, 1540 and 1539 faces): a 257× fall. Quote that COUNT and not a wall-time ratio; the counts
// reproduce digit for digit, while the same change timed 10.9× and 22× on two differently loaded
// machines.
//
// The bucket is the other half, and it is the SMALL half: geom.SurfacesCoincide proves identity only
// between two values of ONE analytic kind (its switch asserts b to a's type), and a merge never
// crosses the keep-sense, so a pair spanning two keys can be skipped without asking. Measured on the
// same body, it skipped 1,539 of that call's 1,188,105 pair tests (0.13%) — the coil's faces are 1540
// planes and one cylinder, so nearly all of them share a bucket. It is here because it is free and
// provably sound, not because it is the win.
//
// The bucket is NOT an equivalence class of coincident surfaces, and must never be read as one:
// SurfacesCoincide compares radii, centres and offsets against a tolerance, so it is neither
// transitive (three radii spaced just under res.Weld() give a≈b, b≈c, a≉c) nor even symmetric (its
// resolution comes from a's loop box alone). The bucket key sidesteps both because it is exact
// equality on a bool and an enum. Every pair INSIDE a bucket is still tested, one at a time.

// faceMergeFact is the per-face half of a pair test: the bucket key that says whether the pair is a
// candidate at all, and the loop box whose resolution onOneSurface reads.
type faceMergeFact struct {
	bucket mergeBucket
	box    math.Box
}

// mergeBucket is the necessary condition for a pair to answer anything but declineUnshared: one
// keep-sense, and one geom.SurfaceKind. Faces with different keys take the declineUnshared exit —
// the one exit reportDeclines never reports and the one that records nothing — so not offering them
// to the pair function changes no output. TestCrossBucketPairsAreUnshared gates that.
type mergeBucket struct {
	reversed bool
	kind     geom.SurfaceKind
	// kinded is false for a surface that names no geom.SurfaceKind. Every kernel surface names one
	// (geom's KindedSurface assertions), so this is the door for a caller-supplied surface, and it
	// puts all of them in ONE bucket rather than assuming anything about them.
	kinded bool
}

// faceMergeFacts tabulates one fact per face, in index order, so the scan reads each face's box and
// bucket once instead of once per pair. It is rebuilt by every scan and must not outlive one: a merge
// rewrites faces[i] and drops faces[j], and both facts about faces[i] then describe the old face.
func faceMergeFacts(faces []curvedFace) []faceMergeFact {
	facts := make([]faceMergeFact, len(faces))
	for i, f := range faces {
		facts[i] = faceMergeFact{bucket: faceMergeBucket(f), box: faceLoopBox(f)}
	}
	return facts
}

// faceMergeBucket keys a face on the two exact facts a merge cannot cross.
//
// Example: faceMergeBucket(a) != faceMergeBucket(b) // then a and b are not a merge candidate
func faceMergeBucket(f curvedFace) mergeBucket {
	kind, kinded := geom.SurfaceKindOf(f.surface)
	return mergeBucket{reversed: f.reversed, kind: kind, kinded: kinded}
}
