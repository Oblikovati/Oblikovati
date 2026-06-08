// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"errors"
	"fmt"

	"oblikovati/kernel/exchange/step/geommap"
	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// addFace maps one ADVANCED_FACE onto the builder. An unsupported surface is a
// warned skip (the face is omitted) rather than a fatal error, per the plan's
// fallback policy — the resulting body's open edges are then reported by Validate.
func (a *assembler) addFace(faceID int) error {
	ent, err := a.g.Lookup(faceID)
	if err != nil {
		return err
	}
	if ent.Keyword != "ADVANCED_FACE" {
		return fmt.Errorf("topomap: #%d is %s, want ADVANCED_FACE", faceID, ent.Keyword)
	}
	surface, skipped, err := a.faceSurface(ent)
	if err != nil || skipped {
		return err
	}
	sameSense, err := faceSameSense(ent)
	if err != nil {
		return err
	}
	return a.buildFaceLoops(ent, surface, sameSense)
}

// faceSurface resolves an ADVANCED_FACE's surface (parameter 2). An unsupported
// surface type is warned and reported as skipped (no error), so the import
// continues and the missing face shows up as boundary edges in Validate.
func (a *assembler) faceSurface(ent *part21.RawEntity) (geom.Surface, bool, error) {
	if len(ent.Params) < 4 {
		return nil, false, fmt.Errorf("topomap: ADVANCED_FACE #%d wants 4 params, got %d", ent.ID, len(ent.Params))
	}
	surfID, err := ent.Params[2].AsRef()
	if err != nil {
		return nil, false, err
	}
	surf, err := geommap.Surface(a.g, surfID, a.scale)
	if err != nil {
		var unsup geommap.ErrUnsupportedSurface
		if errors.As(err, &unsup) {
			a.warn("skipped face #%d: %v", ent.ID, err)
			return nil, true, nil
		}
		return nil, false, err
	}
	return surf, false, nil
}

// faceSameSense reads ADVANCED_FACE.same_sense (the last boolean parameter): true
// means the face's outward sense agrees with its surface normal.
func faceSameSense(ent *part21.RawEntity) (bool, error) {
	return ent.Params[3].AsBool()
}

// buildFaceLoops builds every bound loop and adds the face. The face is added
// reversed when same_sense is false (its material side opposes the surface normal),
// matching topo.AddReversedFace's contract.
func (a *assembler) buildFaceLoops(ent *part21.RawEntity, surface geom.Surface, sameSense bool) error {
	boundRefs, err := refListValues(ent.Params[1])
	if err != nil {
		return err
	}
	bounds := make([]boundLoop, 0, len(boundRefs))
	for _, boundID := range boundRefs {
		b, err := a.buildBound(boundID)
		if err != nil {
			return err
		}
		bounds = append(bounds, b)
	}
	ensureOuterLoop(bounds)
	loops := make([]topo.LoopSpec, 0, len(bounds))
	for _, b := range bounds {
		if b.degenerate {
			continue // VERTEX_LOOP pole/apex — imposes no boundary
		}
		loops = append(loops, loopSpec(b.outer, b.uses))
	}
	a.addBuiltFace(surface, sameSense, loops)
	return nil
}

// ensureOuterLoop guarantees the face has an outer loop flagged. STEP makes
// FACE_OUTER_BOUND optional and some exporters (OpenCASCADE) write FACE_BOUND for EVERY
// bound, which would leave a face with no outer loop — its planar triangulation then yields
// nothing (a missing cap/plate). When no bound was declared outer, infer it: the loop whose
// edges span the largest bounding box encloses the rest, so it is the outer one.
func ensureOuterLoop(bounds []boundLoop) {
	for _, b := range bounds {
		if b.outer {
			return // STEP already declared the outer loop
		}
	}
	best, bestSize := -1, -1.0
	for i := range bounds {
		if bounds[i].degenerate {
			continue // a vertex loop never encloses anything
		}
		if s := loopBoxDiagonal(bounds[i].uses); s > bestSize {
			best, bestSize = i, s
		}
	}
	if best >= 0 {
		bounds[best].outer = true
	}
}

// loopBoxDiagonal is the diagonal length of the bounding box of a loop's edges — a
// rotation-tolerant size used to pick the enclosing (outer) loop.
func loopBoxDiagonal(uses []topo.Use) float64 {
	box := math.EmptyBox()
	for _, u := range uses {
		box = box.Union(u.Edge.RangeBox())
	}
	return float64(box.Diagonal().Length())
}

// addBuiltFace adds the face forward or reversed, with a stable imported lineage.
func (a *assembler) addBuiltFace(surface geom.Surface, sameSense bool, loops []topo.LoopSpec) {
	lineage := topo.NewLineage(topo.Tok(a.feat, "face", a.nextF))
	a.nextF++
	if sameSense {
		a.builder.AddFace(surface, lineage, loops...)
		return
	}
	a.builder.AddReversedFace(surface, lineage, loops...)
}
