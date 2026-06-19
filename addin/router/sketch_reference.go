// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
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
	d, err := resolveQuantity(part, in.Distance, param.Length)
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

// addText places a single sketch TEXT entity at an anchor. The entity keeps its content +
// font and DERIVES its glyph outlines on demand, so it is embossable/extrudable BY REFERENCE
// — nothing is baked into the sketch (the previous host-side font-path path that emitted raw
// polylines is gone; the bug it caused was that an emboss then stored baked geometry).
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
	h, rot, size, err := textMetrics(part, in)
	if err != nil {
		return nil, err
	}
	tb := sk.TextBoxes().AddStyled(anchor, in.Text, h, rot, parseJustify(in.Justify), parseVJustify(in.VJustify), in.Font, size)
	return json.Marshal(wire.AddEntityIDResult{EntityID: uint64(tb.EntityID())})
}

// getText reads back a sketch text entity's style.
func getText(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.GetTextArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketchAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	tb, err := textBoxRef(sk, in.EntityID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.SketchTextResult{EntityID: in.EntityID, Style: styleOf(tb)})
}

// editText applies a partial edit to an existing sketch text entity and returns its style.
// Editing re-derives the text's geometry, so any emboss referencing it recomputes.
func editText(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.EditTextArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	tb, err := textBoxRef(sk, in.EntityID)
	if err != nil {
		return nil, err
	}
	if err := applyTextEdit(part, tb, in); err != nil {
		return nil, err
	}
	return json.Marshal(wire.SketchTextResult{EntityID: in.EntityID, Style: styleOf(tb)})
}

// applyTextEdit mutates the text box's set fields (a partial edit), parsing unit-bearing
// values against the part's units.
func applyTextEdit(part *compdef.PartComponentDefinition, tb *sketch.TextBox, in wire.EditTextArgs) error {
	if in.Text != nil {
		tb.Text = *in.Text
	}
	if in.Font != "" {
		tb.Family = in.Font
	}
	if in.Justify != "" {
		tb.Justify = parseJustify(in.Justify)
	}
	if in.VJustify != "" {
		tb.VJustify = parseVJustify(in.VJustify)
	}
	return applyTextLengths(part, tb, in)
}

// applyTextLengths parses and applies the edit's unit-bearing height/rotation/fontSize.
func applyTextLengths(part *compdef.PartComponentDefinition, tb *sketch.TextBox, in wire.EditTextArgs) error {
	if in.Height != "" {
		h, err := resolveQuantity(part, in.Height, param.Length)
		if err != nil {
			return fmt.Errorf("sketch.editText: height %q: %w", in.Height, err)
		}
		tb.Height = math.Scalar(h.Value)
	}
	if in.FontSize != "" {
		fsz, err := resolveQuantity(part, in.FontSize, param.Length)
		if err != nil {
			return fmt.Errorf("sketch.editText: fontSize %q: %w", in.FontSize, err)
		}
		tb.FontSize = math.Scalar(fsz.Value)
	}
	if in.Rotation != "" {
		rot, err := modelAngle(part, in.Rotation)
		if err != nil {
			return err
		}
		tb.Rotation = math.Scalar(rot)
	}
	return nil
}

// styleOf projects a text box into its wire/types style value.
func styleOf(tb *sketch.TextBox) types.SketchTextStyle {
	return types.SketchTextStyle{
		Content: tb.Text, Family: tb.Family,
		FontSize: float64(tb.FontSize), Height: float64(tb.Height),
		Rotation: float64(tb.Rotation),
		HAlign:   hAlignString(tb.Justify), VAlign: vAlignString(tb.VJustify),
	}
}

// textBoxRef resolves a sketch text entity by id, erroring if missing or not a text box.
func textBoxRef(sk *sketch.Sketch, id uint64) (*sketch.TextBox, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch text: no entity with id %d", id)
	}
	tb, ok := e.(*sketch.TextBox)
	if !ok {
		return nil, fmt.Errorf("sketch entity %d is a %T, not a text box", id, e)
	}
	return tb, nil
}

// textMetrics parses a text box's unit-bearing height, optional rotation, and font size.
func textMetrics(part *compdef.PartComponentDefinition, in wire.AddTextArgs) (height, rot, fontSize math.Scalar, err error) {
	h, err := resolveQuantity(part, in.Height, param.Length)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sketch.addText: height %q: %w", in.Height, err)
	}
	r := 0.0
	if in.Rotation != "" {
		if r, err = modelAngle(part, in.Rotation); err != nil {
			return 0, 0, 0, err
		}
	}
	size := 0.0
	if in.FontSize != "" {
		fsz, ferr := resolveQuantity(part, in.FontSize, param.Length)
		if ferr != nil {
			return 0, 0, 0, fmt.Errorf("sketch.addText: fontSize %q: %w", in.FontSize, ferr)
		}
		size = fsz.Value
	}
	return math.Scalar(h.Value), math.Scalar(r), math.Scalar(size), nil
}

// parseJustify maps a horizontal-justification string to its model enum (default left).
func parseJustify(j string) sketch.TextHJustify {
	switch types.TextHorizontalAlign(j) {
	case types.TextAlignCenter:
		return sketch.TextCenter
	case types.TextAlignRight:
		return sketch.TextRight
	default:
		return sketch.TextLeft
	}
}

// parseVJustify maps a vertical-justification string to its model enum (default baseline).
func parseVJustify(j string) sketch.TextVJustify {
	switch types.TextVerticalAlign(j) {
	case types.TextAlignLower:
		return sketch.TextLower
	case types.TextAlignMiddle:
		return sketch.TextMiddle
	case types.TextAlignUpper:
		return sketch.TextUpper
	default:
		return sketch.TextBaseline
	}
}

// hAlignString / vAlignString map the model enums back to the api/types string values.
func hAlignString(j sketch.TextHJustify) types.TextHorizontalAlign {
	switch j {
	case sketch.TextCenter:
		return types.TextAlignCenter
	case sketch.TextRight:
		return types.TextAlignRight
	default:
		return types.TextAlignLeft
	}
}

func vAlignString(j sketch.TextVJustify) types.TextVerticalAlign {
	switch j {
	case sketch.TextLower:
		return types.TextAlignLower
	case sketch.TextMiddle:
		return types.TextAlignMiddle
	case sketch.TextUpper:
		return types.TextAlignUpper
	default:
		return types.TextAlignBaseline
	}
}

// imageMetrics parses the image's unit-bearing width/height/rotation.
func imageMetrics(part *compdef.PartComponentDefinition, in wire.AddSketchImageArgs) (math.Scalar, math.Scalar, math.Scalar, error) {
	w, err := resolveQuantity(part, in.Width, param.Length)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("sketch.addImage: width %q: %w", in.Width, err)
	}
	h, err := resolveQuantity(part, in.Height, param.Length)
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
