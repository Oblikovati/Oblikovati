// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"reflect"
	"testing"

	"oblikovati.org/math"
)

// The entity codec registry guards (#1624, audit I1), mirroring
// model/feature/serialize_registry_test (#1416): the registered kind sets must
// equal the persisted vocabulary exactly, a half or duplicate registration
// must panic, and one instance of EVERY registered kind must round-trip to an
// identical recipe — so a kind added to one codec half (or to neither) fails
// here at CI, not at the user's save.

// persistedEntityVocabulary2D / 3D are the exact kind sets a recipe may
// contain, one line per kind. Adding an entity type means registering its
// codec AND extending this list AND giving the round-trip tests below an
// instance — all in this package, all enforced together.
var (
	persistedEntityVocabulary2D = []EntityKind{
		LineKind, CircleKind, ArcKind, EllipseKind, EllipticalArcKind,
		SplineKind, ControlPointSplineKind,
		BlockInstanceKind, ImageKind, TextKind, FillRegionKind,
		ProjectedPointKind, ProjectedCurveKind,
		EquationCurveKind, FixedSplineKind, OffsetSplineKind,
	}
	persistedEntityVocabulary3D = []EntityKind{
		LineKind, CircleKind, ArcKind, HelicalKind, EllipseKind, EllipticalArcKind,
		SplineKind, ControlPointSplineKind, FixedSplineKind, EquationCurveKind,
	}
)

func TestEntityCodecRegistryMatchesVocabulary(t *testing.T) {
	assertKindSetsEqual(t, "2D", registeredEntityKinds2D(), persistedEntityVocabulary2D)
	assertKindSetsEqual(t, "3D", registeredEntityKinds3D(), persistedEntityVocabulary3D)
}

func assertKindSetsEqual(t *testing.T, family string, got, want []EntityKind) {
	t.Helper()
	gotSet, wantSet := kindSet(got), kindSet(want)
	for k := range wantSet {
		if !gotSet[k] {
			t.Errorf("%s kind %q is in the persisted vocabulary but has no codec", family, k)
		}
	}
	for k := range gotSet {
		if !wantSet[k] {
			t.Errorf("%s kind %q has a codec but is not in the persisted vocabulary list", family, k)
		}
	}
}

func kindSet(kinds []EntityKind) map[EntityKind]bool {
	out := make(map[EntityKind]bool, len(kinds))
	for _, k := range kinds {
		out[k] = true
	}
	return out
}

func TestRegisterEntityCodecRejectsBadRegistration(t *testing.T) {
	complete2D := entityCodec{
		encode: func(Entity) (EntityData, error) { return EntityData{}, nil },
		decode: func(*sketchRestorer, EntityData) (Entity, error) { return nil, nil },
	}
	complete3D := entityCodec3D{
		encode: func(Entity) (Entity3DData, error) { return Entity3DData{}, nil },
		decode: func(*Sketch3D, Entity3DData, []*Point3D) (Entity, error) { return nil, nil },
	}
	mustPanic(t, "2D empty kind", func() { registerEntityCodec("", complete2D) })
	mustPanic(t, "2D encode-only", func() { registerEntityCodec("testHalf", entityCodec{encode: complete2D.encode}) })
	mustPanic(t, "2D decode-only", func() { registerEntityCodec("testHalf", entityCodec{decode: complete2D.decode}) })
	mustPanic(t, "2D duplicate", func() { registerEntityCodec(LineKind, complete2D) })
	mustPanic(t, "3D empty kind", func() { registerEntityCodec3D("", complete3D) })
	mustPanic(t, "3D encode-only", func() { registerEntityCodec3D("testHalf", entityCodec3D{encode: complete3D.encode}) })
	mustPanic(t, "3D duplicate", func() { registerEntityCodec3D(HelicalKind, complete3D) })
}

// mustPanic asserts fn panics; every rejection fires before the registry map
// is touched, so the probes leave the global registries untouched.
func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s: want a registration panic, got none", name)
		}
	}()
	fn()
}

// TestEveryRegistered2DKindRoundTripsIdentically saves a sketch holding one
// instance of every registered 2D kind, restores it, saves again, and demands
// the two recipes match byte-for-byte — the enumerating closure over the
// registry (#1624 acceptance: a kind cannot register a codec this test does
// not exercise, and no codec half can drift from its pair).
func TestEveryRegistered2DKindRoundTripsIdentically(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	populateEvery2DKind(t, sc, s)

	defs, err := sc.MarshalBlockDefinitions()
	if err != nil {
		t.Fatalf("MarshalBlockDefinitions: %v", err)
	}
	first, err := sc.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	assertRecipeCoversVocabulary(t, entityKindsIn(first), persistedEntityVocabulary2D)

	fresh := NewSketches()
	if err := fresh.ApplyBlockDefinitions(defs); err != nil {
		t.Fatalf("ApplyBlockDefinitions: %v", err)
	}
	if err := fresh.ApplyRecipe(first); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	second, err := fresh.MarshalRecipe()
	if err != nil {
		t.Fatalf("re-MarshalRecipe: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("recipe changed across a round trip:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// populateEvery2DKind creates one entity of each persisted 2D kind through the
// same factories the tools use.
func populateEvery2DKind(t *testing.T, sc *Sketches, s *Sketch) {
	t.Helper()
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	s.Circles().AddByCenterRadius(math.P2(5, 5), 2)
	s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	s.Ellipses().Add(math.P2(2, 2), math.V2(1, 0), 3, 1)
	s.ellArcs.AddWithCenter(s.newPoint(math.P2(-2, 2)), math.V2(0, 1), 2, 1, 0.2, 1.4)
	sp := s.Splines().AddByPoints([]math.Point2{math.P2(0, 0), math.P2(1, 1), math.P2(2, 0)}, false)
	s.Splines().AddByControlPoints([]math.Point2{math.P2(0, -1), math.P2(1, -2), math.P2(2, -1)}, false)
	s.OffsetSplines().Add(sp, 0.5)
	if _, err := s.EquationCurves().Add("cos(t)", "sin(t)", 0, 1); err != nil {
		t.Fatalf("EquationCurves.Add: %v", err)
	}
	s.FixedSplines().Add([]math.Point2{math.P2(3, 3), math.P2(4, 4), math.P2(5, 3)})
	s.images.Add("res-1", math.P2(6, 6), 2, 1, 0.1, 0.8)
	s.texts.AddStyled(math.P2(7, 7), "note", 0.5, 0, TextHJustify(0), TextVJustify(0), "sans", 0)
	s.fills.Add(math.P2(0.5, 0.25), "solid")
	s.RestoreProjectedPoint(nextID(), math.P2(8, 8), "vertex", "V1")
	s.RestoreProjection(s.addReferencePolyline([]math.Point2{math.P2(8, 0), math.P2(9, 1)}), "edge", "E1")
	blockSource := s.Lines().AddByTwoPoints(math.P2(10, 10), math.P2(11, 10))
	if _, _, err := s.Blocks().CreateFromSelection(sc.BlockDefinitions(), "blk", []Entity{blockSource}); err != nil {
		t.Fatalf("CreateFromSelection: %v", err)
	}
}

// TestEveryRegistered3DKindRoundTripsIdentically is the 3D twin.
func TestEveryRegistered3DKindRoundTripsIdentically(t *testing.T) {
	c := NewSketches3D()
	s := c.Add()
	populateEvery3DKind(t, s)

	first, err := c.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("MarshalRecipe3D: %v", err)
	}
	assertRecipeCoversVocabulary(t, entityKinds3DIn(first), persistedEntityVocabulary3D)

	fresh := NewSketches3D()
	if err := fresh.ApplyRecipe3D(first); err != nil {
		t.Fatalf("ApplyRecipe3D: %v", err)
	}
	second, err := fresh.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("re-MarshalRecipe3D: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("3D recipe changed across a round trip:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// populateEvery3DKind creates one entity of each persisted 3D kind.
func populateEvery3DKind(t *testing.T, s *Sketch3D) {
	t.Helper()
	uz := mustUnit3(t, 0, 0, 1)
	ux := mustUnit3(t, 1, 0, 0)
	s.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	s.AddCircle3D(math.P3(0, 0, 2), uz, 1.5)
	s.AddArc3D(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), true)
	s.AddHelix3D(math.P3(2, 2, 0), uz, 1, 0.5, 0, 3, false)
	s.AddEllipse3D(math.P3(4, 0, 0), uz, ux, 3, 1)
	s.AddEllipticalArc3D(math.P3(0, 4, 0), uz, ux, 3, 1, 0.2, 1.4)
	s.AddSpline3D([]math.Point3{math.P3(0, 0, 0), math.P3(1, 1, 1), math.P3(2, 0, 2)}, false, true)
	s.AddSpline3D([]math.Point3{math.P3(5, 0, 0), math.P3(6, 1, 1), math.P3(7, 0, 2)}, false, false)
	s.AddFixedSpline3D([]math.Point3{math.P3(0, 5, 0), math.P3(1, 6, 1), math.P3(2, 5, 2)}, false)
	if _, err := s.AddEquationCurve3D("cos(t)", "sin(t)", "t", 0, 1); err != nil {
		t.Fatalf("AddEquationCurve3D: %v", err)
	}
}

func mustUnit3(t *testing.T, x, y, z float64) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(math.Scalar(x), math.Scalar(y), math.Scalar(z))
	if err != nil {
		t.Fatalf("NewUnitVector3(%v,%v,%v): %v", x, y, z, err)
	}
	return u
}

// assertRecipeCoversVocabulary demands the marshaled recipe contains every
// registered kind at least once — the closure that forces a newly registered
// kind to also appear in these round-trip tests.
func assertRecipeCoversVocabulary(t *testing.T, seen map[EntityKind]bool, vocabulary []EntityKind) {
	t.Helper()
	for _, k := range vocabulary {
		if !seen[k] {
			t.Errorf("registered kind %q never appeared in the recipe — add an instance to the round-trip fixture", k)
		}
	}
}

func entityKindsIn(data []SketchData) map[EntityKind]bool {
	out := map[EntityKind]bool{}
	for _, sd := range data {
		for _, ed := range sd.Entities {
			out[EntityKind(ed.Kind)] = true
		}
	}
	return out
}

func entityKinds3DIn(data []SketchData3D) map[EntityKind]bool {
	out := map[EntityKind]bool{}
	for _, sd := range data {
		for _, ed := range sd.Entities {
			out[EntityKind(ed.Kind)] = true
		}
	}
	return out
}
