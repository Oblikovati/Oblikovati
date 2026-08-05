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
}

// The 41 pre-#1626 part-feature tools that once reached the plain StartTool
// without DraftFeature (each an open commit-gate bypass) have all been
// converted — do NOT reintroduce a bypass list; a new part-feature tool
// implements DraftFeature and activates via StartFeatureTool, full stop.

// plainStartPatterns match the two ways a commands file feeds a constructor to
// the plain StartTool: directly, or through a func() Tool flyout-table thunk.
var plainStartPatterns = []*regexp.Regexp{
	regexp.MustCompile(`s\.StartTool\(New(\w+Tool)\(\)\)`),
	regexp.MustCompile(`func\(\) Tool \{ return New(\w+Tool)\(\) \}`),
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
	files, err := filepath.Glob("../app/commands_*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("globbing app command files: %v (found %d)", err, len(files))
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
