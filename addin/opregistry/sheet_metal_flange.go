// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

const sheetMetalFlangeSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to flange from (from get_reference_keys)."},
    "height": {"type": "string", "description": "Flange wall length with units, e.g. \"15 mm\"."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (default). The wall folds this far from the parent face."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sheet."},
    "bendPosition": {"type": "string", "enum": ["adjacentFace", "outsideBaseFace", "insideBendFace", "outerEdgeOffset", "innerEdgeOffset"], "default": "adjacentFace", "description": "How far back from the picked edge the bend sits (Inventor BendPositionEnum). adjacentFace starts the bend AT the edge; outsideBaseFace and insideBendFace set it back until the wall's outer or inner face reaches the edge, so the wall does not overhang; the two ...EdgeOffset positions add positionOffset to those."},
    "positionOffset": {"type": "string", "description": "Explicit distance for the outerEdgeOffset / innerEdgeOffset positions, e.g. \"2 mm\"."},
    "applyAutoMiter": {"type": "boolean", "default": false, "description": "Extend this wall and the one it corners with until they meet, then cut miterGap between them. Two walls each stop at their own bend line, so the corner between them is otherwise open."},
    "miterGap": {"type": "string", "description": "Gap left on the miter line, e.g. \"1 mm\"; absent uses the style's GapSize."},
    "options": {"type": "object", "description": "Override the sheet-metal style's bend properties for THIS bend only (Inventor's BendOptions). An omitted field defers to the style.", "properties": {"reliefShape": {"type": "string", "enum": ["round", "straight", "tear"], "description": "Notch shape at this bend's ends; tear cuts nothing."}, "reliefWidth": {"type": "string"}, "reliefDepth": {"type": "string"}, "minimumRemnant": {"type": "string", "description": "Thinnest strip of parent material a relief may leave; a notch that would leave less takes the sliver with it."}, "transition": {"type": "string", "enum": ["none", "intersection", "straightLine", "arc", "trimToBend", "default"], "description": "How this bend meets the face beside it. Only \"none\" is built; the others are refused where they would apply."}, "transitionArcRadius": {"type": "string"}}},
    "width": {"type": "object", "description": "Bound the wall to PART of the edge (Inventor's flange width extents): a bracket tab on a long edge, or a wall that stops short of the corners. Absent = the whole edge.", "properties": {"type": {"type": "string", "enum": ["edge", "centered", "offsets", "offsetWidth"], "default": "edge", "description": "centered takes width; offsets takes offset (from the edge start) and offset2 (from its end); offsetWidth takes offset and width."}, "width": {"type": "string", "description": "Wall length, e.g. \"20 mm\"."}, "offset": {"type": "string", "description": "Distance from the edge's start."}, "offset2": {"type": "string", "description": "Distance from the edge's end (offsets type)."}}},
    "heightDatum": {"type": "string", "enum": ["tangent", "outer", "inner", "outerOrtho", "innerOrtho"], "default": "tangent", "description": "What height is measured FROM (Inventor HeightDatumTypeEnum). tangent measures the wall from where the bend ends; outer/inner measure from the sharp corner the outer/inner faces would make, the way a drawing dimensions it; the ortho pair measures those corners perpendicular to the base face."},
    "edgeSets": {"type": "array", "description": "Flange SEVERAL edges in one feature (Inventor's FlangeDefinition edge-set collection), each set with its own edges and width. Supersedes edge/width when present; the shared height, angle, radius, flip, bendPosition, heightDatum, options and miter apply to every set.", "items": {"type": "object", "properties": {"edges": {"type": "array", "items": {"type": "string"}, "description": "Reference keys of the edges in this set."}, "width": {"type": "object", "description": "This set's width extent (same shape as the top-level width); absent = each wall spans its whole edge."}}, "required": ["edges"]}}
  },
  "required": ["height"]
}`

// sheetMetalFlangeDescriptor is the self-describing "sheetMetalFlange" operation: fold a wall
// onto a sheet edge over a bend at the active rule's gauge.
func sheetMetalFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalFlange,
		Summary: "Fold a wall (flange) onto a straight sheet-metal edge over a cylindrical bend, at the active rule's thickness and bend radius.",
		Schema:  json.RawMessage(sheetMetalFlangeSchema),
		Apply:   applySheetMetalFlange,
	}
}

func applySheetMetalFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalFlange](s, raw, "sheetMetalFlange")
	if err != nil {
		return nil, err
	}
	if in.Edge == "" && len(in.EdgeSets) == 0 {
		return nil, fmt.Errorf("sheetMetalFlange: an edge (or edgeSets) is required")
	}
	def, err := flangeDef(part, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewSheetMetalFlangeFeatures(part.Features()).Add(def)
	return recomputeResult(part, pf)
}

// flangeDef resolves the flange args into a definition: the edge key, the height closure, and
// the optional angle/radius closures (omitted ⇒ nil, so the feature uses its defaults).
func flangeDef(part *compdef.PartComponentDefinition, in featureargs.SheetMetalFlange) (*feature.SheetMetalFlangeDefinition, error) {
	height, err := lengthClosure(part, in.Height, "sheetMetalFlange: height")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalFlangeDefinition{EdgeKey: []byte(in.Edge), Height: height, Flip: in.Flip}
	if err := bindFlangeBend(part, def, in); err != nil {
		return nil, err
	}
	if err := bindFlangePlacement(part, def, in); err != nil {
		return nil, err
	}
	if err := bindFlangeSpans(part, def, in); err != nil {
		return nil, err
	}
	return def, nil
}

// bindFlangeSpans resolves the wall's width, its multi-edge edge sets (#2071), the per-bend option
// override and the auto-miter onto the definition.
func bindFlangeSpans(part *compdef.PartComponentDefinition, def *feature.SheetMetalFlangeDefinition, in featureargs.SheetMetalFlange) error {
	width, err := flangeWidthExtent(part, in.Width)
	if err != nil {
		return err
	}
	def.Width = width
	if def.EdgeSets, err = flangeEdgeSets(part, in.EdgeSets); err != nil {
		return err
	}
	if def.Options, err = bendOptions(part, in.Options); err != nil {
		return err
	}
	def.AutoMiter = in.ApplyAutoMiter
	if in.MiterGap != "" {
		def.MiterGap, err = lengthClosure(part, in.MiterGap, "sheetMetalFlange: miterGap")
	}
	return err
}

// flangeEdgeSets resolves a multi-edge flange's edge-set collection (#2071): each set's edges and its
// own width extent. Empty in ⇒ nil, the single-edge flange this feature has always built.
func flangeEdgeSets(part *compdef.PartComponentDefinition, in []featureargs.FlangeEdgeSet) ([]feature.FlangeEdgeSet, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]feature.FlangeEdgeSet, len(in))
	for i, s := range in {
		if len(s.Edges) == 0 {
			return nil, fmt.Errorf("sheetMetalFlange: edge set %d has no edges", i)
		}
		width, err := flangeWidthExtent(part, s.Width)
		if err != nil {
			return nil, err
		}
		out[i] = feature.FlangeEdgeSet{EdgeKeys: refKeys(s.Edges), Width: width}
	}
	return out, nil
}

// bendOptions resolves this bend's overrides of the style (#1959). An omitted field stays nil so
// the style still decides it — that is what makes the block an override and not a restatement.
func bendOptions(part *compdef.PartComponentDefinition, in *featureargs.BendOptions) (*feature.BendOptions, error) {
	if in == nil {
		return nil, nil
	}
	out := &feature.BendOptions{}
	if err := bindBendReliefShape(out, in.ReliefShape); err != nil {
		return nil, err
	}
	if err := bindBendTransition(out, in.Transition); err != nil {
		return nil, err
	}
	for _, d := range []struct {
		expr, what string
		dst        *func() float64
	}{
		{in.ReliefWidth, "reliefWidth", &out.ReliefWidth},
		{in.ReliefDepth, "reliefDepth", &out.ReliefDepth},
		{in.MinimumRemnant, "minimumRemnant", &out.MinimumRemnant},
		{in.TransitionArcRadius, "transitionArcRadius", &out.TransitionArcRadius},
	} {
		if d.expr == "" {
			continue
		}
		closure, err := lengthClosure(part, d.expr, "sheetMetalFlange: options "+d.what)
		if err != nil {
			return nil, err
		}
		*d.dst = closure
	}
	return out, nil
}

// bindBendReliefShape resolves a per-bend relief shape override.
func bindBendReliefShape(out *feature.BendOptions, spelling string) error {
	if spelling == "" {
		return nil
	}
	shape, ok := types.ParseReliefShape(strings.TrimSpace(spelling))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown options.reliefShape %q (want round|straight|tear)", spelling)
	}
	out.ReliefShape = &shape
	return nil
}

// bindBendTransition resolves a per-bend transition override; "default" defers to the style.
func bindBendTransition(out *feature.BendOptions, spelling string) error {
	if spelling == "" {
		out.Transition = types.DefaultBendTransition
		return nil
	}
	kind, ok := types.ParseBendTransition(strings.TrimSpace(spelling))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown options.transition %q "+
			"(want none|intersection|straightLine|arc|trimToBend|default)", spelling)
	}
	out.Transition = kind
	return nil
}

// flangeWidthExtent resolves a width extent's type and distances (#1958). Each distance is a
// driven expression, so a parameter change moves the tab with the rest of the part.
func flangeWidthExtent(part *compdef.PartComponentDefinition, in *featureargs.FlangeWidthExtent) (feature.FlangeWidth, error) {
	if in == nil {
		return feature.FlangeWidth{}, nil
	}
	kind, ok := feature.ParseWidthExtent(strings.TrimSpace(in.Type))
	if !ok {
		return feature.FlangeWidth{}, fmt.Errorf("sheetMetalFlange: unknown width extent %q (want "+
			"edge, centered, offsets or offsetWidth)", in.Type)
	}
	w := feature.FlangeWidth{Type: kind}
	for _, d := range []struct {
		expr, what string
		dst        *func() float64
	}{{in.Width, "width", &w.Width}, {in.Offset, "offset", &w.Offset}, {in.Offset2, "offset2", &w.Offset2}} {
		if d.expr == "" {
			continue
		}
		closure, err := lengthClosure(part, d.expr, "sheetMetalFlange: width "+d.what)
		if err != nil {
			return feature.FlangeWidth{}, err
		}
		*d.dst = closure
	}
	return w, nil
}

// bindFlangeBend attaches the optional bend overrides; omitted ⇒ nil, so the feature applies its
// own defaults (90° over the rule's bend radius).
func bindFlangeBend(part *compdef.PartComponentDefinition, def *feature.SheetMetalFlangeDefinition,
	in featureargs.SheetMetalFlange) error {
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalFlange: angle")
		if err != nil {
			return err
		}
		def.Angle = angle
	}
	if in.Radius == "" {
		return nil
	}
	radius, err := lengthClosure(part, in.Radius, "sheetMetalFlange: radius")
	if err != nil {
		return err
	}
	def.Radius = radius
	return nil
}

// bindFlangePlacement resolves where the wall lands (#1957): the bend position and its offset, and
// the datum the height is measured from.
func bindFlangePlacement(part *compdef.PartComponentDefinition, def *feature.SheetMetalFlangeDefinition,
	in featureargs.SheetMetalFlange) error {
	position, ok := feature.ParseBendPosition(strings.TrimSpace(in.BendPosition))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown bendPosition %q (want adjacentFace, "+
			"outsideBaseFace, insideBendFace, outerEdgeOffset or innerEdgeOffset)", in.BendPosition)
	}
	datum, ok := feature.ParseHeightDatum(strings.TrimSpace(in.HeightDatum))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown heightDatum %q (want tangent, outer, inner, "+
			"outerOrtho or innerOrtho)", in.HeightDatum)
	}
	def.Position, def.HeightDatum = position, datum
	if in.PositionOffset == "" {
		return nil
	}
	offset, err := lengthClosure(part, in.PositionOffset, "sheetMetalFlange: positionOffset")
	if err != nil {
		return err
	}
	def.PositionOffset = offset
	return nil
}
