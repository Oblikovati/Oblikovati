// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// The thread table wire surface (M09-F01 PBI-101, #325).

// TestThreadsTableQueryProgressive: types always; sizes with a type;
// designations with a size; classes with a designation (side-aware).
func TestThreadsTableQueryProgressive(t *testing.T) {
	r, s := emptyPartSession(t)
	var bare wire.ThreadTableQueryResult
	call(t, r, s, "threads.tableQuery", `{}`, &bare)
	if len(bare.ThreadTypes) < 2 || bare.NominalSizes != nil {
		t.Fatalf("bare query = %+v, want types only", bare)
	}

	var sized wire.ThreadTableQueryResult
	call(t, r, s, "threads.tableQuery", `{"threadType":"ISO"}`, &sized)
	if len(sized.NominalSizes) == 0 {
		t.Fatal("ISO query listed no nominal sizes")
	}

	var full wire.ThreadTableQueryResult
	call(t, r, s, "threads.tableQuery", `{"threadType":"ISO","nominalSize":"M8","designation":"M8x1.25","internal":true}`, &full)
	if len(full.Designations) == 0 || len(full.Classes) == 0 {
		t.Fatalf("full query = %+v, want designations and classes", full)
	}
	if full.Classes[0] != "6H" {
		t.Errorf("internal ISO class = %q, want 6H", full.Classes[0])
	}

	if _, err := r.Handle(s, "threads.tableQuery", []byte(`{"threadType":"BSW"}`)); err == nil {
		t.Fatal("an unknown thread type must error")
	}
}

// TestThreadsResolveCarriesParityFields: the resolved DTO carries class /
// tapered / handedness alongside the derived diameters.
func TestThreadsResolveCarriesParityFields(t *testing.T) {
	r, s := emptyPartSession(t)
	var info wire.ThreadInfoResult
	call(t, r, s, "threads.resolve", `{"designation":"M8x1.25","class":"6H","internal":true,"tapered":false}`, &info)
	if !info.Metric || info.ThreadType != "ISO" || info.NominalSize != "M8" {
		t.Errorf("resolved = %+v, want metric ISO M8", info)
	}
	if info.Class != "6H" || !info.Internal || !info.RightHanded {
		t.Errorf("parity fields = class %q internal %v rh %v", info.Class, info.Internal, info.RightHanded)
	}
	if info.TapDrillDiameter != 8-1.25 || info.PitchDiameter != 8-0.6495*1.25 {
		t.Errorf("derived diameters = %g / %g", info.TapDrillDiameter, info.PitchDiameter)
	}

	var lh wire.ThreadInfoResult
	call(t, r, s, "threads.resolve", `{"designation":"1/4-20","leftHanded":true}`, &lh)
	if lh.RightHanded || lh.Metric || lh.ThreadType != "ANSI" {
		t.Errorf("left-handed inch resolve = %+v", lh)
	}

	if _, err := r.Handle(s, "threads.resolve", []byte(`{}`)); err == nil {
		t.Fatal("resolve without a designation must error")
	}
}
