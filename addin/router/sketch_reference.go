// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	e, ok := sk.EntityByID(sketch.ID(in.Entity))
	if !ok {
		return nil, fmt.Errorf("sketch.offset: no entity with id %d", in.Entity)
	}
	d, err := part.Units().Parse(in.Distance, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch.offset: distance %q: %w", in.Distance, err)
	}
	off, err := sk.OffsetEntity(e, math.Scalar(d.Value))
	if err != nil {
		return nil, err
	}
	kind, _, _ := entityShape(off)
	return json.Marshal(wire.OffsetSketchResult{EntityID: uint64(off.EntityID()), Kind: string(kind)})
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
	t := sk.TextBoxes().Add(anchor, in.Text, h, rot, parseJustify(in.Justify))
	return json.Marshal(wire.AddEntityIDResult{EntityID: uint64(t.EntityID())})
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
