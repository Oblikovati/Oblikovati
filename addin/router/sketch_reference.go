// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"os"

	"oblikovati/api/wire"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/param"
	"oblikovati/model/sketch"
	"oblikovati/model/text"
)

// offsetSketchEntity offsets a single line/circle/arc by a unit-bearing distance.
func offsetSketchEntity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.OffsetSketchArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	d, err := part.Units().Parse(in.Distance, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch.offset: distance %q: %w", in.Distance, err)
	}
	if in.ProfileIndex != nil {
		return offsetProfileRegion(sk, *in.ProfileIndex, d.Value, in.ArcSegments)
	}
	if len(in.Entities) > 0 {
		return offsetChain(sk, in.Entities, math.Scalar(d.Value))
	}
	return offsetSingle(sk, in.Entity, math.Scalar(d.Value))
}

// offsetProfileRegion offsets a profile's whole closed boundary (OpenSCAD offset(r)) — convex
// corners rounded — and returns the created lines of the new loop.
func offsetProfileRegion(sk *sketch.Sketch, profIdx int, d float64, arcSegs int) (json.RawMessage, error) {
	profs := sk.Profiles()
	if profIdx < 0 || profIdx >= profs.Count() {
		return nil, fmt.Errorf("sketch.offset: profile %d not found (sketch has %d)", profIdx, profs.Count())
	}
	poly := profs.Item(profIdx).OuterLoop().Polygon()
	if len(poly) < 3 {
		return nil, fmt.Errorf("sketch.offset: profile %d is not a closed region", profIdx)
	}
	ents := sk.OffsetClosedLoop(poly, d, arcSegs)
	if len(ents) == 0 {
		return nil, fmt.Errorf("sketch.offset: region offset collapsed (distance %.4g too negative?)", d)
	}
	created := make([]uint64, len(ents))
	for i, e := range ents {
		created[i] = uint64(e.EntityID())
	}
	return json.Marshal(wire.OffsetSketchResult{EntityID: created[0], Kind: "line", Created: created})
}

// offsetSingle offsets one referenced line/circle/arc.
func offsetSingle(sk *sketch.Sketch, id uint64, d math.Scalar) (json.RawMessage, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch.offset: no entity with id %d", id)
	}
	off, err := sk.OffsetEntity(e, d)
	if err != nil {
		return nil, err
	}
	kind, _, _ := entityShape(off)
	return json.Marshal(wire.OffsetSketchResult{EntityID: uint64(off.EntityID()), Kind: string(kind), Created: []uint64{uint64(off.EntityID())}})
}

// offsetChain offsets a connected chain of referenced lines, mitring the joins.
func offsetChain(sk *sketch.Sketch, refs []uint64, d math.Scalar) (json.RawMessage, error) {
	lines := make([]*sketch.Line, len(refs))
	for i, id := range refs {
		l, err := lineRef(sk, id)
		if err != nil {
			return nil, err
		}
		lines[i] = l
	}
	off := sk.OffsetChain(lines, d)
	created := make([]uint64, len(off))
	for i, l := range off {
		created[i] = uint64(l.EntityID())
	}
	return json.Marshal(wire.OffsetSketchResult{EntityID: created[0], Kind: "line", Created: created})
}

// autoDimensionSketch fully constrains the sketch and reports the added count + DOF.
func autoDimensionSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	added := sk.AutoDimension()
	return json.Marshal(wire.AutoDimensionResult{Added: added, DOF: sk.DegreesOfFreedom()})
}

// addSketchImage places a raster image on the sketch plane.
func addSketchImage(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSketchImageArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	anchor, err := point2Of(in.Anchor, "anchor")
	if err != nil {
		return nil, err
	}
	w, h, rot, err := imageMetrics(part, in)
	if err != nil {
		return nil, err
	}
	img := sk.Images().Add(in.Ref, anchor, w, h, rot, in.Opacity)
	return json.Marshal(wire.AddSketchImageResult{EntityID: uint64(img.EntityID())})
}

// addFillRegion fills the closed region containing the seed point.
func addFillRegion(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddFillRegionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketchAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	seed, err := point2Of(in.Seed, "seed")
	if err != nil {
		return nil, err
	}
	f := sk.FillRegions().Add(seed, in.Style)
	return json.Marshal(wire.AddEntityIDResult{EntityID: uint64(f.EntityID())})
}

// addText places sketch text at an anchor.
func addText(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddTextArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	anchor, err := point2Of(in.Anchor, "anchor")
	if err != nil {
		return nil, err
	}
	h, rot, err := textMetrics(part, in)
	if err != nil {
		return nil, err
	}
	if in.Font != "" {
		return addTextGeometry(sk, anchor, in.Text, float64(h), in.Font)
	}
	t := sk.TextBoxes().Add(anchor, in.Text, h, rot, parseJustify(in.Justify))
	return json.Marshal(wire.AddEntityIDResult{EntityID: uint64(t.EntityID())})
}

// addTextGeometry renders the string's true-type glyph outlines (from the host-side font
// file) into closed sketch polylines anchored at anchor, so the text becomes real,
// embossable/extrudable geometry (each letter's counter is a profile hole).
func addTextGeometry(sk *sketch.Sketch, anchor math.Point2, s string, height float64, fontPath string) (json.RawMessage, error) {
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("sketch.addText: read font %q: %w", fontPath, err)
	}
	ft, err := text.Parse(data)
	if err != nil {
		return nil, err
	}
	contours, err := ft.Outlines(s, height)
	if err != nil {
		return nil, err
	}
	ents := sk.AddTextOutlines(anchor, contours)
	if len(ents) == 0 {
		return nil, fmt.Errorf("sketch.addText: text %q produced no glyph geometry", s)
	}
	ids := make([]uint64, len(ents))
	for i, e := range ents {
		ids[i] = uint64(e.EntityID())
	}
	return json.Marshal(wire.AddSketchEntityResult{EntityID: ids[0], Kind: "text", EntityIDs: ids})
}

// textMetrics parses a text box's unit-bearing height and optional rotation.
func textMetrics(part *compdef.PartComponentDefinition, in wire.AddTextArgs) (math.Scalar, math.Scalar, error) {
	h, err := part.Units().Parse(in.Height, param.Length)
	if err != nil {
		return 0, 0, fmt.Errorf("sketch.addText: height %q: %w", in.Height, err)
	}
	rot := 0.0
	if in.Rotation != "" {
		if rot, err = modelAngle(part, in.Rotation); err != nil {
			return 0, 0, err
		}
	}
	return math.Scalar(h.Value), math.Scalar(rot), nil
}

// parseJustify maps a justification string to its model enum (default left).
func parseJustify(j string) sketch.TextHJustify {
	switch j {
	case "center":
		return sketch.TextCenter
	case "right":
		return sketch.TextRight
	default:
		return sketch.TextLeft
	}
}

// imageMetrics parses the image's unit-bearing width/height/rotation.
func imageMetrics(part *compdef.PartComponentDefinition, in wire.AddSketchImageArgs) (math.Scalar, math.Scalar, math.Scalar, error) {
	w, err := part.Units().Parse(in.Width, param.Length)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sketch.addImage: width %q: %w", in.Width, err)
	}
	h, err := part.Units().Parse(in.Height, param.Length)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sketch.addImage: height %q: %w", in.Height, err)
	}
	rot := 0.0
	if in.Rotation != "" {
		if rot, err = modelAngle(part, in.Rotation); err != nil {
			return 0, 0, 0, err
		}
	}
	return math.Scalar(w.Value), math.Scalar(h.Value), math.Scalar(rot), nil
}
