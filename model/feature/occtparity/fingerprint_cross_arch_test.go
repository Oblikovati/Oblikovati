// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The cross-architecture half of the byte-identity pins (ADR-0064, Oblikovati#3528).
//
// The pins were amd64-only: TestByteIdentityFingerprints skipped outright on the macOS leg, because
// gc contracts x*y+z into one unrounded fused multiply-add on arm64 and never on amd64, so the same
// bodies tessellate to different last bits and hash differently. Skipping made the strongest gate
// in the repository run on two of three CI legs and say nothing about the third.
//
// ADR-0064 rounds every product in `math`, `kernel/geom` and `kernel/predicates` explicitly, which
// removes that difference for the arithmetic those packages own. It does not remove it everywhere:
// `kernel/brep`, `kernel/ops/*`, `kernel/topo` and `model/feature` still contract, and so does the
// standard library's own source. So the pins now RUN everywhere, and the bodies whose mesh still
// differs on a contracting platform are NAMED here rather than hidden behind a skip.
//
// The set is a ratchet in both directions. A body that drifts without being listed is a real
// regression — this policy is meant to be closing that gap, not opening it. A listed body that
// stops drifting means the gap closed and the entry must go, or the list stops meaning anything.

// crossArchHashDrift names the pinned bodies whose mesh still differs on a platform that contracts
// x*y+z into an FMA — arm64, which is the macOS CI leg. It is a list of NAMES, not a second set of
// hashes: what it records is that a body is not yet bit-identical, never what it hashes to there.
//
// MEASURED for ADR-0064 by dumping every pin's fingerprint under linux/arm64 emulation
// (`make arm64 PKG=./model/feature/occtparity`) at this wave's base 6590a9ba and again on this HEAD,
// and comparing both against the amd64 values the pins already hold:
//
//	of the 88 pinned bodies         base    HEAD
//	bit-identical to the amd64 pin    23  ->   42
//	hash differs, same triangles      49  ->   43
//	triangle COUNT differs            16  ->    3
//
// Twenty-one bodies became bit-identical (simple/B1, B9, C5, C9, D3, D5, D9, E4, F4, I7, J1, J2, J6,
// J8, K9, L4, L9, N5, N9, U3, V6) plus tolblend_simple/A1 in the cluster-A set, and the structural
// half of the drift — a refinement decision landing differently, which is the kind that changes a
// triangle COUNT — all but disappeared, 16 rows down to 3.
//
// TWO went the other way: bfuseblend/A6 and simple/D8 hashed identically before and do not now.
// That is not a regression in the policy, it is what a PARTIAL conversion means. These bodies pass
// through kernel/brep, kernel/ops/* and model/feature, which still contract; feeding them inputs
// that no longer round the way they used to moves where their remaining fused arithmetic lands. The
// two rows are the honest cost of stopping at the arithmetic floor, and they close when the
// packages above it are converted.
//
// So this list is a measurement of how much of the kernel still contracts, and it shrinks to empty
// as `make arm64-fma` reaches zero package by package (#3528 follow-ups).
var crossArchHashDrift = map[string]bool{
	"bfuseblend/A4":      true,
	"bfuseblend/A6":      true,
	"bfuseblend/B1":      true,
	"bfuseblend/B4":      true,
	"bfuseblend/B5":      true,
	"simple/A6":          true,
	"simple/B3":          true,
	"simple/C2":          true,
	"simple/C6":          true,
	"simple/C8":          true,
	"simple/D4":          true,
	"simple/D8":          true,
	"simple/E1":          true,
	"simple/E2":          true,
	"simple/E3":          true,
	"simple/E7":          true,
	"simple/G5":          true,
	"simple/G7":          true,
	"simple/G9":          true,
	"simple/I3":          true,
	"simple/I5":          true,
	"simple/I9":          true,
	"simple/J3":          true,
	"simple/K2":          true,
	"simple/K3":          true,
	"simple/L6":          true,
	"simple/L7":          true,
	"simple/M2":          true,
	"simple/M4":          true,
	"simple/M8":          true,
	"simple/N1":          true,
	"simple/N4":          true,
	"simple/N7":          true,
	"simple/O1":          true,
	"simple/S1":          true,
	"simple/S4":          true,
	"simple/S7":          true,
	"simple/S9":          true,
	"simple/T1":          true,
	"simple/T3":          true,
	"simple/T4":          true,
	"simple/T7":          true,
	"simple/U4":          true,
	"simple/W6":          true,
	"simple/W8":          true,
	"simple/Z1":          true,
	"tolblend_simple/D4": true,
}

// contractsFloatingPointArithmetic reports whether this platform's compiler fuses x*y+z. amd64
// never has; arm64 (the macOS CI leg) always does.
func contractsFloatingPointArithmetic() bool { return runtime.GOARCH != "amd64" }

// crossArchDriftVolTol is the relative volume tolerance a drifting body is held to on a contracting
// platform. The pinned 1e-9 is a CROSS-COMMIT tolerance on one machine; across architectures the
// worst measured difference over the drifting bodies is 6.2e-6 (simple/E3, whose triangle count
// differs), so 1e-4 leaves the gate real with about a decade and a half of margin. A body that is
// NOT listed keeps volTolFor's own tolerance everywhere — it is bit-identical, so it has no excuse,
// and measured, the worst such body differs by 9.6e-15.
const crossArchDriftVolTol = 1e-4

// assertPinnedFingerprint compares a rebuilt fingerprint with its pin, allowing exactly the drift
// crossArchHashDrift records and only on a platform that contracts.
func assertPinnedFingerprint(t *testing.T, tc fingerprintPin, fp meshFingerprint) {
	t.Helper()
	key := pinKey(tc.grid, tc.name)
	matched := fp.Hash == tc.hash && fp.Triangles == tc.tris
	if !contractsFloatingPointArithmetic() {
		if !matched {
			t.Fatalf("%s fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d (shared geometry changed)",
				tc.name, fp.Hash, fp.Triangles, tc.hash, tc.tris)
		}
		return
	}
	switch listed := crossArchHashDrift[key]; {
	case matched && listed:
		t.Fatalf("%s now matches its amd64 pin on %s/%s — the contraction no longer reaches this "+
			"body. DELETE its crossArchHashDrift entry. (The list was measured under linux/arm64 "+
			"emulation, `make arm64 PKG=./model/feature/occtparity`; a darwin/arm64 runner "+
			"disagreeing on a row is a bookkeeping fix here, not a defect.) ADR-0064, #3528",
			key, runtime.GOOS, runtime.GOARCH)
	case !matched && !listed:
		t.Fatalf("%s is NOT bit-identical on %s/%s: hash=%#x tris=%d, want hash=%#x tris=%d. Either "+
			"a change reintroduced a contractible product (run `make arm64-fma PKG=...` on the "+
			"packages it touched) or this body was never bit-identical and the pin set is wrong "+
			"(ADR-0064, #3528)",
			key, runtime.GOOS, runtime.GOARCH, fp.Hash, fp.Triangles, tc.hash, tc.tris)
	}
}

// pinnedVolumeTolerance is volTolFor, widened for a body that is allowed to drift on a contracting
// platform: a different triangulation integrates to a different volume. It only ever WIDENS —
// volTolFor's own two exceptions (M4 at 5e-4, D3 at 3e-3) are already looser than the cross-arch
// allowance and must not be tightened by it.
func pinnedVolumeTolerance(grid, name string) float64 {
	tol := volTolFor(name)
	if contractsFloatingPointArithmetic() && crossArchHashDrift[pinKey(grid, name)] && tol < crossArchDriftVolTol {
		return crossArchDriftVolTol
	}
	return tol
}

// pinKey names a pinned body the way crossArchHashDrift does: grid-qualified, because A2, A3 and A4
// exist in more than one corpus grid and are different bodies there.
func pinKey(grid, name string) string {
	if grid == "" {
		return "simple/" + name
	}
	return grid + "/" + name
}

// TestCrossArchDriftListIsExactlyThePinnedNames keeps the drift list honest: every name in it must
// be a pin, so a rename or a deletion cannot leave a dead entry that silently excuses a real body.
func TestCrossArchDriftListIsExactlyThePinnedNames(t *testing.T) {
	t.Parallel()
	pinned := map[string]bool{}
	for _, tc := range append(byteIdentityPins(), clusterAPins()...) {
		pinned[pinKey(tc.grid, tc.name)] = true
	}
	var orphans []string
	for name := range crossArchHashDrift {
		if !pinned[name] {
			orphans = append(orphans, name)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("crossArchHashDrift names bodies that are no longer pinned — delete these entries:\n  %s",
			strings.Join(orphans, "\n  "))
	}
}
