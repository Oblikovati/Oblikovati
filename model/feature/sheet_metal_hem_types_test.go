// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The four Inventor hem shapes (#1956). Two of them are driven by a curl instead of a leg, and the
// double folds twice, so these tests check the SHAPE each one produces — a hem that quietly built
// a single fold would still be a valid watertight solid.

// hemBody folds the given hem onto a 4 cm square sheet of 2 mm gauge and returns the result.
func hemBody(t *testing.T, def *SheetMetalHemDefinition) *topo.Body {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	def.EdgeKey = edge.ReferenceKey()
	pf := NewSheetMetalHemFeatures(fs).Add(def)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("hem sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("hemmed sheet is not a valid solid: %+v", r)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("hemmed sheet has %d boundary edges, want 0 (watertight)", len(open))
	}
	return body
}

// hemHealth folds a hem and reports the feature's health without demanding success.
func hemHealth(t *testing.T, def *SheetMetalHemDefinition) *PartFeature {
	t.Helper()
	fs, edge := seedSheetMetalSheet(t, 4, nil)
	def.EdgeKey = edge.ReferenceKey()
	pf := NewSheetMetalHemFeatures(fs).Add(def)
	fs.Recompute()
	return pf
}

// TestDoubleHemStacksThreeLayers: the second fold has to curl the OTHER way. Curling it the same
// way would drive the free leg down through the parent — still a "valid" boolean, but a part with
// two layers instead of three. The stack height is what tells them apart: a single hem at this
// gauge tops out near 2r+t, a double adds another leg and its own tight fold above that.
func TestDoubleHemStacksThreeLayers(t *testing.T) {
	t.Parallel()
	single := topZOf(hemBody(t, &SheetMetalHemDefinition{Type: SingleHem, Length: constClosure(0.8)}))
	double := topZOf(hemBody(t, &SheetMetalHemDefinition{Type: DoubleHem, Length: constClosure(0.8)}))
	// The sheet's top face is at z=0.2 (2 mm gauge). A tight fold adds 2·(t/2)+t = 0.4, so a single
	// hem tops at 0.6; the double's second fold adds another 0.4. Anything at the single's height
	// means only one fold was built.
	if stdmath.Abs(single-0.6) > 1e-6 {
		t.Errorf("single hem top = %.4f, want 0.6", single)
	}
	if stdmath.Abs(double-1.0) > 1e-6 {
		t.Errorf("double hem top = %.4f, want 1.0 (three stacked layers)", double)
	}
}

// TestDoubleHemAddsMaterial: three layers hold more material than two. Volume is the check the
// height cannot make on its own — a second fold that collapsed onto the first would still be tall.
func TestDoubleHemAddsMaterial(t *testing.T) {
	t.Parallel()
	single := smSolidVolume(hemBody(t, &SheetMetalHemDefinition{Type: SingleHem, Length: constClosure(0.8)}))
	double := smSolidVolume(hemBody(t, &SheetMetalHemDefinition{Type: DoubleHem, Length: constClosure(0.8)}))
	// The extra leg is 0.8 long, 0.2 thick, across the 4 cm edge, plus its fold: ≳ 0.64 cm³.
	if extra := double - single; extra < 0.6 {
		t.Errorf("double hem adds %.4f cm³ over the single, want the second leg's ≳0.6", extra)
	}
}

// TestRolledHemCurlsToItsRadius: a rolled hem is all curl and no leg, so its height is set by the
// radius alone — 2r+t for a half-turn, and more as the sweep passes it.
func TestRolledHemCurlsToItsRadius(t *testing.T) {
	t.Parallel()
	half := hemBody(t, &SheetMetalHemDefinition{
		Type: RolledHem, Radius: constClosure(0.3), Angle: constClosure(stdmath.Pi),
	})
	// The fold starts at the sheet's top face (z=0.2) and rises 2r+t.
	if want := 0.2 + 2*0.3 + 0.2; stdmath.Abs(topZOf(half)-want) > 1e-6 {
		t.Errorf("half-turn roll top = %.4f, want %.4f (2r+t above the face)", topZOf(half), want)
	}
	// A 270° sweep passes the top of the circle and comes back down, so the peak stays 2r+t while
	// the curl now reaches back toward the sheet — the free end is what moved.
	threeQuarter := hemBody(t, &SheetMetalHemDefinition{
		Type: RolledHem, Radius: constClosure(0.3), Angle: constClosure(1.5 * stdmath.Pi),
	})
	if got := topZOf(threeQuarter); stdmath.Abs(got-(0.2+2*0.3+0.2)) > 1e-6 {
		t.Errorf("270° roll top = %.4f, want the same 2r+t peak", got)
	}
	if v := smSolidVolume(threeQuarter); v <= smSolidVolume(half) {
		t.Error("a 270° roll should carry more material than a half-turn")
	}
}

// TestTeardropClosesBackOntoTheSheet: the tail is DERIVED — the whole point of the teardrop is
// that the curl comes back and touches the parent. If the tail were merely omitted (or given some
// invented length) the loop would hang open, which is a different part.
func TestTeardropClosesBackOntoTheSheet(t *testing.T) {
	t.Parallel()
	curl := &SheetMetalHemDefinition{Radius: constClosure(0.3), Angle: constClosure(1.5 * stdmath.Pi)}
	rolled := hemBody(t, &SheetMetalHemDefinition{Type: RolledHem, Radius: curl.Radius, Angle: curl.Angle})
	teardrop := hemBody(t, &SheetMetalHemDefinition{Type: TeardropHem, Radius: curl.Radius, Angle: curl.Angle})

	// The parent's top face is at z=0.2. A bare roll of this sweep ends in mid-air, so the only
	// material it leaves at that height is the sheet itself; the teardrop's tail lands back on the
	// face, which is exactly the extra contact the shape is named for.
	if onFace(teardrop, 0.2) <= onFace(rolled, 0.2) {
		t.Errorf("teardrop touches the face at %d vertices vs the roll's %d — the tail did not close",
			onFace(teardrop, 0.2), onFace(rolled, 0.2))
	}
	if smSolidVolume(teardrop) <= smSolidVolume(rolled) {
		t.Error("the teardrop's tail should add material over the bare roll")
	}
}

// onFace counts a body's vertices lying at height z.
func onFace(body *topo.Body, z float64) int {
	n := 0
	for _, v := range body.Vertices() {
		if stdmath.Abs(float64(v.Point().Z)-z) < 1e-6 {
			n++
		}
	}
	return n
}

// TestTeardropSweepMustCloseTheLoop: at or below a half-turn the tail never heads back to the
// sheet, and at a full turn the loop has already closed — the derived tail would be infinite or
// negative, so both are refused rather than built into a wall that runs off the part.
func TestTeardropSweepMustCloseTheLoop(t *testing.T) {
	t.Parallel()
	for name, angle := range map[string]float64{
		"half turn":    stdmath.Pi,
		"quarter turn": stdmath.Pi / 2,
		"full turn":    2 * stdmath.Pi,
	} {
		t.Run(name, func(t *testing.T) {
			pf := hemHealth(t, &SheetMetalHemDefinition{
				Type: TeardropHem, Radius: constClosure(0.3), Angle: constClosure(angle),
			})
			if pf.Health().OK() {
				t.Errorf("a teardrop sweeping %s should be refused", name)
			}
		})
	}
}

// TestCurledHemsNeedTheirCurl: a rolled or teardrop hem with no radius/angle has nothing to build.
// Falling back to the folded hem's gauge-derived radius would silently produce a single hem.
func TestCurledHemsNeedTheirCurl(t *testing.T) {
	t.Parallel()
	for _, typ := range []HemType{RolledHem, TeardropHem} {
		pf := hemHealth(t, &SheetMetalHemDefinition{Type: typ, Length: constClosure(0.8)})
		if pf.Health().OK() {
			t.Errorf("hem type %d with no radius/angle should be refused", typ)
		}
	}
}

// TestHemBendSpecsCoverEveryFold: the flat pattern develops each fold, so a double hem must report
// both — one reported fold would under-develop the blank and cut the part short.
func TestHemBendSpecsCoverEveryFold(t *testing.T) {
	t.Parallel()
	double := &SheetMetalHemFeature{def: &SheetMetalHemDefinition{Type: DoubleHem, Length: constClosure(0.8)}}
	specs := double.BendSpecs(0.2)
	if len(specs) != 2 {
		t.Fatalf("double hem BendSpecs = %+v, want two folds", specs)
	}
	for i, s := range specs {
		if stdmath.Abs(s.Angle-stdmath.Pi) > 1e-12 || s.Radius != 0.1 {
			t.Errorf("fold %d = %+v, want a π fold at the tight 0.1 radius", i, s)
		}
	}
	rolled := &SheetMetalHemFeature{def: &SheetMetalHemDefinition{
		Type: RolledHem, Radius: constClosure(0.3), Angle: constClosure(1.5 * stdmath.Pi),
	}}
	if s := rolled.BendSpecs(0.2); len(s) != 1 || s[0].Radius != 0.3 {
		t.Errorf("rolled hem BendSpecs = %+v, want one fold at its own 0.3 radius", s)
	}
}

// TestHemTypeRoundTripsByName: the type is persisted by NAME because the ordinals could not be
// extended — ordinal 1 used to be the open hem and is now the double. A document written before
// the names must still read as the single hem both old ordinals meant.
func TestHemTypeRoundTripsByName(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewSheetMetalHemFeatures(fs).Add(&SheetMetalHemDefinition{
		EdgeKey: []byte("edge"), Type: TeardropHem,
		Radius: constClosure(0.3), Angle: constClosure(4.7),
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].SheetMetalHem; d.Type != "teardrop" || d.Radius != 0.3 || d.Angle != 4.7 {
		t.Fatalf("serialized hem = %+v, want the teardrop with its curl", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SheetMetalHemFeature).Definition()
	if def.Type != TeardropHem || def.Radius() != 0.3 {
		t.Errorf("restored hem = type %d radius %g, want the teardrop at 0.3", def.Type, evalFloat(def.Radius))
	}
}

// TestLegacyHemOrdinalRestoresAsSingle: both old ordinals — closed (0) and open (1) — were single
// hems, and the gap is still what tells them apart. Ordinal 1 must NOT decode as the double hem
// that now holds that number.
func TestLegacyHemOrdinalRestoresAsSingle(t *testing.T) {
	t.Parallel()
	for name, d := range map[string]*SheetMetalHemData{
		"legacy closed": {Edge: "ZWRnZQ==", Length: 0.6, LegacyType: 0},
		"legacy open":   {Edge: "ZWRnZQ==", Length: 0.6, LegacyType: 1, Gap: 0.4},
	} {
		t.Run(name, func(t *testing.T) {
			fs := NewPartFeatures(nil)
			if err := fs.ApplyRecipe([]FeatureData{{Kind: "sheet-metal-hem", SheetMetalHem: d}}, oneSketch{}, nil); err != nil {
				t.Fatalf("ApplyRecipe: %v", err)
			}
			def := fs.Item(0).Definition().(*SheetMetalHemFeature).Definition()
			if def.Type != SingleHem {
				t.Errorf("%s restored as type %d, want the single hem", name, def.Type)
			}
			if d.Gap != 0 && evalFloat(def.Gap) != d.Gap {
				t.Errorf("%s lost its gap: %g", name, evalFloat(def.Gap))
			}
		})
	}
}
