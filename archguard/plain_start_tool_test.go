// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The sick-config commit gate (#1594) inspects the active tool's draft feature —
// a part-feature tool activated through the plain Session.StartTool with no
// DraftFeature skips the gate silently, the bypass-by-omission that shipped
// #1521 (#1626, audit I3). Every activation site of a feature-committing tool
// must go through Session.StartFeatureTool (whose PartFeatureTool parameter the
// compiler checks); the plain StartTool is reserved for the constructors named
// below. A NEW constructor reaching StartTool fails here until it is either
// switched to StartFeatureTool or consciously classified as non-feature.

// allowedPlainStartTools are the documented non-feature tools: sketch and
// 3D-sketch geometry/constraint tools, drawing views/annotations, assembly
// tools (assembly features live outside the part gate — commitBlockedReason is
// part-scoped), work features, measure/analysis probes, and settings dialogs.
// ControlPointEditTool commits nothing (its edits apply live per drag).
var allowedPlainStartTools = map[string]struct{}{
	"AddSheetTool":             {},
	"AngularDimensionTool":     {},
	"Arc3DTool":                {},
	"ArcLengthDimensionTool":   {},
	"ArcTool":                  {},
	"AssemblyChamferTool":      {},
	"AssemblyCircPatternTool":  {},
	"AssemblyExtrudeTool":      {},
	"AssemblyFilletTool":       {},
	"AssemblyHoleTool":         {},
	"AssemblyMirrorTool":       {},
	"AssemblyRectPatternTool":  {},
	"AssemblyRevolveTool":      {},
	"AuxiliaryViewTool":        {},
	"BalloonTool":              {},
	"BaseViewTool":             {},
	"Bend3DTool":               {},
	"BreakViewTool":            {},
	"BreakoutViewTool":         {},
	"CenterMarkTool":           {},
	"CenterPointArcTool":       {},
	"CenterRectangleTool":      {},
	"CenterlineTool":           {},
	"Circle3DTool":             {},
	"CircleTool":               {},
	"CoGMarkerTool":            {},
	"ContinuityCheckTool":      {},
	"ControlPointEditTool":     {},
	"ControlPointSpline3DTool": {},
	"ControlVertexSplineTool":  {},
	"CreateSketchTool":         {},
	"CustomTableTool":          {},
	"DatumFeatureTool":         {},
	"DetailViewTool":           {},
	"DimensionSetTool":         {},
	"DimensionTool":            {}, // the sketch general dimension tool (#2022)
	"DraftViewTool":            {},
	"DraftingStandardTool":     {},
	"EllipseTool":              {},
	"EquationCurve3DTool":      {},
	"FeatureControlFrameTool":  {},
	"GripSnapTool":             {},
	"HatchRegionTool":          {},
	"Helix3DTool":              {},
	"HoleNotesTool":            {},
	"HoleTableTool":            {},
	"IncludeGeometry3DTool":    {},
	"Line3DTool":               {},
	"LineTool":                 {},
	"LinearDimensionTool":      {},
	"MeasureTool":              {},
	"ModelReferenceTool":       {},
	"NoteTool":                 {},
	"AngleWorkPlaneTool":       {}, // datum plane, not a part feature (#2044)
	"FreeformCageEditTool":     {}, // drag-driven cage editing, commits per drag not per OK (#2048)
	"OffsetWorkPlaneTool":      {},
	"OrdinateDimensionTool":    {},
	"PartsListTool":            {},
	"PlaceComponentTool":       {},
	"Point3DTool":              {},
	"PointTool":                {},
	"ProjectGeometryTool":      {},
	"ProjectedViewTool":        {},
	"RadialDimensionTool":      {},
	"RectangleTool":            {},
	"RevisionCloudTool":        {},
	"RevisionTableTool":        {},
	"RevisionTagTool":          {},
	"SectionViewTool":          {},
	"SheetMetalStyleTool":      {},
	"SketchCircPatternTool":    {},
	"SketchCircleTool":         {},
	"SketchCopyTool":           {},
	"SketchCreateBlockTool":    {},
	"SketchExtendTool":         {},
	"SketchMirrorTool":         {},
	"SketchMoveTool":           {},
	"SketchRectPatternTool":    {},
	"SketchRectangleTool":      {},
	"SketchRotateTool":         {},
	"SketchScaleTool":          {},
	"SketchSplitTool":          {},
	"SketchStretchTool":        {},
	"SketchTextTool":           {},
	"SketchTrimTool":           {},
	"SliceViewTool":            {},
	"Spline3DTool":             {},
	"SplineTool":               {},
	"SurfaceCurve3DTool":       {},
	"SurfaceDeviationTool":     {},
	"SurfaceTextureTool":       {},
	"ThreePointCircleTool":     {},
	"ThreePointRectangleTool":  {},

	// Surfaced when the scan widened past commands_*.go and past argument-free constructors
	// (#2051): each is a sketch, assembly or display tool, none a part feature.
	"AssemblyConstraintTool":  {}, // assembly constraint — outside the part commit gate
	"AssemblyFeatureEditTool": {}, // re-opens a committed assembly feature; not a part feature
	"AssemblyJointTool":       {}, // assembly joint — outside the part commit gate
	"CenterPointArcSlotTool":  {}, // sketch geometry
	"CloudMoveTool":           {}, // drag-driven point-cloud move; commits per drag, not per OK (#645)
	"CropBoxTool":             {}, // point-cloud crop box; display-scoped, no part feature
	"PolygonTool":             {}, // sketch geometry (constructor takes the side count)
	"SketchChamferTool":       {}, // sketch geometry
	"SketchFilletTool":        {}, // sketch geometry
	"SketchOffsetTool":        {}, // sketch geometry
	"SketchPlaceBlockTool":    {}, // sketch geometry (constructor takes the block)
	"SketchSlotTool":          {}, // sketch geometry
	"ThreePointArcSlotTool":   {}, // sketch geometry
}

// The 41 pre-#1626 part-feature tools that once reached the plain StartTool
// without DraftFeature (each an open commit-gate bypass) have all been
// converted — do NOT reintroduce a bypass list; a new part-feature tool
// implements DraftFeature and activates via StartFeatureTool, full stop.

// plainStartPatterns match the ways a source feeds a constructor to the plain StartTool:
// directly (with or without arguments), or through a func() Tool flyout-table thunk.
var plainStartPatterns = []*regexp.Regexp{
	regexp.MustCompile(`s\.StartTool\(New(\w+Tool)\(`),
	regexp.MustCompile(`func\(\) Tool \{ return New(\w+Tool)\(`),
}

func TestPlainStartToolReservedForNonFeatureTools(t *testing.T) {
	started := plainStartedToolConstructors(t)
	for name := range started {
		if _, allowed := allowedPlainStartTools[name]; !allowed {
			t.Errorf("New%s reaches the plain StartTool — a part-feature tool must implement DraftFeature and activate via StartFeatureTool so the commit gate cannot be skipped (#1626); a non-feature tool must be classified in allowedPlainStartTools", name)
		}
	}
	for name := range allowedPlainStartTools {
		if _, ok := started[name]; !ok {
			t.Errorf("allowedPlainStartTools entry %q is stale — the tool no longer reaches the plain StartTool; delete the entry", name)
		}
	}
}

// plainStartedToolConstructors collects every constructor fed to the plain
// StartTool across the app's command builders.
func plainStartedToolConstructors(t *testing.T) map[string]struct{} {
	t.Helper()
	// EVERY app source, not just the command builders: a tool started from a browser action, an
	// edit-mode re-open or an input handler is as much an activation site as a ribbon command,
	// and the narrower scan left those outside the seam entirely (#2051).
	files, err := filepath.Glob("../app/*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("globbing app sources: %v (found %d)", err, len(files))
	}
	out := map[string]struct{}{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		for _, p := range plainStartPatterns {
			for _, m := range p.FindAllStringSubmatch(string(src), -1) {
				out[m[1]] = struct{}{}
			}
		}
	}
	return out
}
