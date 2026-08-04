// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// TestEdgeCatalogRecordsCurveDisagreementsInsteadOfArbitrating drives the SHIPPED edgeCatalog.use
// and requires the build report of the body it produced to name every disagreement between the two
// consumers of a welded edge — never to resolve one silently.
//
// ★ The two curves in the conflict case have IDENTICAL endpoints and lie on the SAME circle: they
// are an arc and its complement, the exact shape of M8's defect. A gate that compared endpoints, or
// centres, or radii, would pass it. Only a comparison of where the two curves GO between those
// endpoints fires, which is what curveOfferDeviation measures.
func TestEdgeCatalogRecordsCurveDisagreementsInsteadOfArbitrating(t *testing.T) {
	minor, major := complementaryArcPair()
	for _, tc := range []struct {
		name       string
		first, snd geom.Curve3
		wantCode   diag.Code
		wantIn     string
	}{
		{"an arc against its complement", major, minor, CodeAssembleCurveConflict, "decided by build order"},
		{"a curve against a nil", major, nil, CodeAssembleCurveNilOffer, "the other nil"},
		{"a nil against a curve", nil, major, CodeAssembleCurveNilOffer, "the other nil"},
		{"the same curve twice", major, major, "", ""},
		{"nil twice", nil, nil, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ec := newSeamCatalog([]m.Point3{major.PointAt(0), major.PointAt(1)})
			ec.use(0, 1, tc.first, 0)
			ec.use(1, 0, tc.snd, 0)
			assertSoleBuildDiagnostic(t, ec.bld.Build().BuildDiagnostics(), tc.wantCode, tc.wantIn)
		})
	}
}

// assertSoleBuildDiagnostic requires exactly one build diagnostic of the wanted code (or none when
// the code is empty), and that its Detail carries the named substring.
func assertSoleBuildDiagnostic(t *testing.T, got []diag.Diagnostic, want diag.Code, wantIn string) {
	t.Helper()
	if want == "" {
		if len(got) != 0 {
			t.Fatalf("consumers that agree recorded %d diagnostics, want none: %v", len(got), got)
		}
		return
	}
	if len(got) != 1 || got[0].Code != want {
		t.Fatalf("recorded %v, want exactly one %s", got, want)
	}
	if !strings.Contains(got[0].Detail, wantIn) {
		t.Errorf("diagnostic detail %q does not name %q", got[0].Detail, wantIn)
	}
}

// complementaryArcPair returns the two arcs of ONE circle that share both endpoints — the minor
// 104.4775° span and the major 255.5225° one, M8's own pair, on the boss rim's own circle.
func complementaryArcPair() (minor, major geom.Arc3d) {
	parent := m8BossRimParent()
	end := m8RimContactAngle()
	major = geom.Arc3d{Center: parent.Center, Normal: parent.Normal, RefDir: parent.RefDir,
		Radius: parent.Radius, StartAngle: 0, SweepAngle: end}
	minor = geom.Arc3d{Center: parent.Center, Normal: parent.Normal, RefDir: parent.RefDir,
		Radius: parent.Radius, StartAngle: end, SweepAngle: 2*stdmath.Pi - end}
	return minor, major
}

// TestComplementaryArcsAreNotTakenForEachOther pins the deviation measure itself on the pair the
// whole defect turns on: two arcs of one circle with the same endpoints, one 255.5225° and one
// 104.4775°, must read ≈2R apart — and an arc against ITSELF must read zero, whichever direction
// the second consumer traverses it in (the catalog sees both).
func TestComplementaryArcsAreNotTakenForEachOther(t *testing.T) {
	minor, major := complementaryArcPair()
	if got := curveOfferDeviation(major, minor); got < 40 {
		t.Errorf("an arc and its complement read %.6g apart, want > 40 (their circle's diameter is 50) — "+
			"a measure this blunt would let the complement ship", got)
	}
	reversed := geom.Arc3d{Center: major.Center, Normal: major.Normal, RefDir: major.RefDir,
		Radius: major.Radius, StartAngle: major.StartAngle + major.SweepAngle, SweepAngle: -major.SweepAngle}
	if got := curveOfferDeviation(major, reversed); got > 1e-9 {
		t.Errorf("the same arc traversed backwards reads %.6g apart from itself, want ≈0 — the measure must "+
			"orient the second offer before comparing, else every shared edge looks like a conflict", got)
	}
}
