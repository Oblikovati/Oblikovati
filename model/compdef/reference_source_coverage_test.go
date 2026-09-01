// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"reflect"
	"testing"
)

// sourceKinded is the optional capability a projection reference source declares so a restored
// projection can rebuild a live source and stay associative after a reload; a source without it
// is persisted frozen (model/sketch, #1268). It is a DELIBERATELY-PARTIAL capability, so the
// audit (I10, #1633) records the classification of every reference source in a coverage table
// rather than leaving each absence a silent question.
type sourceKinded interface{ SourceKind() string }

// refSourceKindCoverage records which reference sources declare a SourceKind. The point/curve
// sources that back a sketch projection do (so the projection rebinds); FaceRefSource does NOT —
// it is a surface source on the 3D surface-projection path, not the point/curve rebind, so its
// absence is a decision, not an oversight. A new reference source must be classified here.
var refSourceKindCoverage = map[reflect.Type]bool{
	reflect.TypeFor[EdgeRefSource]():      true,
	reflect.TypeFor[VertexRefSource]():    true,
	reflect.TypeFor[WorkPointRefSource](): true,
	reflect.TypeFor[WorkAxisRefSource]():  true,
	reflect.TypeFor[WorkPlaneRefSource](): true,
	reflect.TypeFor[FaceRefSource]():      false,
}

// TestReferenceSourceKindCoverage asserts each reference source's real sourceKinded satisfaction
// matches the table — a source the table calls kinded that forgot SourceKind (so it would freeze
// on reload), or a newly kinded source the table denies, fails CI (#1633, #1268).
func TestReferenceSourceKindCoverage(t *testing.T) {
	t.Parallel()
	samples := []any{
		EdgeRefSource{}, VertexRefSource{}, FaceRefSource{},
		WorkPointRefSource{}, WorkAxisRefSource{}, WorkPlaneRefSource{},
	}
	if len(samples) != len(refSourceKindCoverage) {
		t.Fatalf("ref-source coverage has %d entries, %d sample types — a stale or missing classification",
			len(refSourceKindCoverage), len(samples))
	}
	for _, s := range samples {
		want, listed := refSourceKindCoverage[reflect.TypeOf(s)]
		if !listed {
			t.Errorf("%T is unclassified — mark it in refSourceKindCoverage", s)
			continue
		}
		if _, got := s.(sourceKinded); got != want {
			t.Errorf("%T sourceKinded=%v, table says %v", s, got, want)
		}
	}
}
