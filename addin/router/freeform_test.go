// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The #699 acceptance over the wire: a placed freeform body's cage edits — level,
// vertex moves, edge creases — through the freeform.* methods.

// freeformBoxViaAPI places a 4×4×4 freeform box at the default level 1 and returns its
// stable id (the op treats level 0 as absent, so the fixture starts at 1).
func freeformBoxViaAPI(t *testing.T) (*Router, *app.Session, uint64) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "features.add", `{"kind":"freeformBox","args":{"sizeX":"4 mm","sizeY":"4 mm","sizeZ":"4 mm"}}`, &struct{}{})
	tree := modelTreeOf(t, r, s)
	if len(tree.Features) != 1 {
		t.Fatalf("fixture: model.tree reports %d features, want 1", len(tree.Features))
	}
	return r, s, tree.Features[0].ID
}

// partBody returns the active part's only body.
func partBodyFaces(t *testing.T, s *app.Session) int {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies, want 1", def.SurfaceBodies().Count())
	}
	return len(def.SurfaceBodies().Item(0).Faces())
}

func TestFreeformSetLevelOverWire(t *testing.T) {
	t.Parallel()
	r, s, id := freeformBoxViaAPI(t)
	// One Catmull–Clark round splits each cage quad into 4: 6 → 24 at the default level 1.
	if got := partBodyFaces(t, s); got != 24 {
		t.Fatalf("level-1 box has %d faces, want 24", got)
	}
	var detail wire.FeatureDetailResult
	call(t, r, s, "freeform.setLevel", fmt.Sprintf(`{"id":%d,"level":0}`, id), &detail)
	if got := partBodyFaces(t, s); got != 6 {
		t.Errorf("level-0 box has %d faces, want 6 (the raw cage)", got)
	}
	if detail.Feature.ID != id {
		t.Errorf("reply feature id = %d, want %d", detail.Feature.ID, id)
	}
}

func TestFreeformMoveVerticesOverWire(t *testing.T) {
	t.Parallel()
	r, s, id := freeformBoxViaAPI(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := bodyVolume(def)

	// Pull the whole top quad up: the cage is a box, vertices 4..7 are one face's corners
	// in subd.Box order; moving all four up by 2 stretches the box (volume grows).
	var detail wire.FeatureDetailResult
	call(t, r, s, "freeform.moveVertices",
		fmt.Sprintf(`{"id":%d,"vertices":[4,5,6,7],"translation":[0,0,2]}`, id), &detail)
	if after := bodyVolume(def); after <= before {
		t.Errorf("volume after vertex move = %g, want > %g", after, before)
	}
}

func TestFreeformCreaseEdgesOverWire(t *testing.T) {
	t.Parallel()
	r, s, id := freeformBoxViaAPI(t)
	var detail wire.FeatureDetailResult
	call(t, r, s, "freeform.setLevel", fmt.Sprintf(`{"id":%d,"level":2}`, id), &detail)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	smooth := bodyVolume(def)

	// Creasing every cage edge fully sharp keeps the subdivided box at its cage size —
	// strictly more volume than the smooth (inward-rounded) limit body.
	ff := featureByIDOf(t, s, id)
	edges := ff.FreeformBody().Edges()
	creases := ""
	for i := 0; i < edges.Count(); i++ {
		a, b := edges.Item(i).Ends()
		if creases != "" {
			creases += ","
		}
		creases += fmt.Sprintf("[%d,%d]", a, b)
	}
	call(t, r, s, "freeform.creaseEdges",
		fmt.Sprintf(`{"id":%d,"edges":[%s],"sharpness":1}`, id, creases), &detail)
	if creased := bodyVolume(def); creased <= smooth {
		t.Errorf("creased volume = %g, want > smooth %g (sharp box outgrows rounded one)", creased, smooth)
	}
}

func TestFreeformEditRejectsBadInputs(t *testing.T) {
	t.Parallel()
	r, s, id := freeformBoxViaAPI(t)
	if _, err := r.Handle(s, "freeform.moveVertices",
		[]byte(fmt.Sprintf(`{"id":%d,"vertices":[99],"translation":[1,0,0]}`, id))); err == nil {
		t.Error("an out-of-range cage vertex must error")
	}
	if _, err := r.Handle(s, "freeform.creaseEdges",
		[]byte(fmt.Sprintf(`{"id":%d,"edges":[[0,1]],"sharpness":2}`, id))); err == nil {
		t.Error("a sharpness above 1 must error")
	}
	if _, err := r.Handle(s, "freeform.setLevel", []byte(`{"id":999999,"level":1}`)); err == nil {
		t.Error("an unknown feature id must error")
	}
}

// featureByIDOf resolves the placed freeform feature for cage introspection.
func featureByIDOf(t *testing.T, s *app.Session, id uint64) *feature.FreeformFeature {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	pf, ok := def.Features().ByID(feature.ID(id))
	if !ok {
		t.Fatalf("feature %d not found", id)
	}
	return pf.Definition().(*feature.FreeformFeature)
}

// bodyVolume measures the part's first body (for before/after cage-edit comparisons).
func bodyVolume(def *compdef.PartComponentDefinition) float64 {
	return float64(ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume)
}
