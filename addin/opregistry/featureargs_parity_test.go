// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/api/wire/featureargs"
)

// Every feature kind add_feature can create must have a typed argument struct in
// api/wire/featureargs (so an add-in builds it with a compile-checked type, not a raw
// JSON blob) OR be an explicit, tracked exception in dynamicKinds. This is the wire<->host
// parity guard for feature creation (#1616, audit B5), the same shape as the
// wire/router/client registration parity guard. Because opregistry now decodes into the
// SAME featureargs types the client marshals, a promoted kind's wire shape and host decoder
// cannot drift — the round-trip is guaranteed by type identity (the featureargs package's
// own marshal round-trip test proves each type is JSON-symmetric).

// dynamicKinds are the registered kinds NOT yet promoted to featureargs — the composite,
// multi-kind, args-less, and mechanical-remainder cases tracked by #1709. Shrink-only:
// promoting a kind here fails the guard until its entry is removed. A truly dynamic
// add-in-registered op would also live here (with its own justification).
var dynamicKinds = map[string]string{
	"bendPart":                "#1709: mechanical promotion pending",
	"boundaryPatch":           "#1709: mechanical promotion pending",
	"bridgeSurface":           "#1709: mechanical promotion pending",
	"chamfer":                 "#1709: composite kind — nested helper structs to promote too",
	"combine":                 "#1709: mechanical promotion pending",
	"coreCavity":              "#1709: mechanical promotion pending",
	"deleteFace":              "#1709: mechanical promotion pending",
	"draft":                   "#1709: mechanical promotion pending",
	"extend":                  "#1709: mechanical promotion pending",
	"faceOffset":              "#1709: mechanical promotion pending",
	"fairSurface":             "#1709: mechanical promotion pending",
	"fillSurface":             "#1709: mechanical promotion pending",
	"fillet":                  "#1709: composite kind — nested helper structs to promote too",
	"fitSurface":              "#1709: mechanical promotion pending",
	"freeformBox":             "#1709: one struct serves several kinds",
	"freeformPlane":           "#1709: one struct serves several kinds",
	"freeformQuadBall":        "#1709: one struct serves several kinds",
	"fullRoundFillet":         "#1709: composite kind — nested helper structs to promote too",
	"hull":                    "#1709: args-less kind",
	"lip":                     "#1709: mechanical promotion pending",
	"loft":                    "#1709: composite kind — nested helper structs to promote too",
	"midSurface":              "#1709: mechanical promotion pending",
	"mirror":                  "#1709: mechanical promotion pending",
	"modelTolerance":          "#1709: composite kind — nested helper structs to promote too",
	"moveBody":                "#1709: composite kind — nested helper structs to promote too",
	"moveFace":                "#1709: mechanical promotion pending",
	"networkSurface":          "#1709: mechanical promotion pending",
	"patternCircular":         "#1709: composite kind — nested helper structs to promote too",
	"patternRectangular":      "#1709: composite kind — nested helper structs to promote too",
	"patternSketchDriven":     "#1709: composite kind — nested helper structs to promote too",
	"replaceFace":             "#1709: mechanical promotion pending",
	"rest":                    "#1709: mechanical promotion pending",
	"ruleFillet":              "#1709: composite kind — nested helper structs to promote too",
	"ruledSurface":            "#1709: mechanical promotion pending",
	"sculpt":                  "#1709: mechanical promotion pending",
	"sheetMetalBend":          "#1709: mechanical promotion pending",
	"sheetMetalContourFlange": "#1709: mechanical promotion pending",
	"sheetMetalContourRoll":   "#1709: mechanical promotion pending",
	"sheetMetalCorner":        "#1709: mechanical promotion pending",
	"sheetMetalCornerSeam":    "#1709: mechanical promotion pending",
	"sheetMetalCosmeticBend":  "#1709: mechanical promotion pending",
	"sheetMetalCut":           "#1709: mechanical promotion pending",
	"sheetMetalFace":          "#1709: mechanical promotion pending",
	"sheetMetalFlange":        "#1709: mechanical promotion pending",
	"sheetMetalFold":          "#1709: mechanical promotion pending",
	"sheetMetalHem":           "#1709: mechanical promotion pending",
	"sheetMetalLip":           "#1709: mechanical promotion pending",
	"sheetMetalLoftedFlange":  "#1709: mechanical promotion pending",
	"sheetMetalPunch":         "#1709: mechanical promotion pending",
	"sheetMetalRefold":        "#1709: mechanical promotion pending",
	"sheetMetalRip":           "#1709: mechanical promotion pending",
	"sheetMetalUnfold":        "#1709: mechanical promotion pending",
	"shell":                   "#1709: mechanical promotion pending",
	"simplify":                "#1709: mechanical promotion pending",
	"snapFit":                 "#1709: mechanical promotion pending",
	"split":                   "#1709: mechanical promotion pending",
	"splitSolid":              "#1709: mechanical promotion pending",
	"stitch":                  "#1709: mechanical promotion pending",
	"surfaceOffset":           "#1709: mechanical promotion pending",
	"sweep":                   "#1709: composite kind — nested helper structs to promote too",
	"thicken":                 "#1709: mechanical promotion pending",
	"trim":                    "#1709: mechanical promotion pending",
	"unwrap":                  "#1709: mechanical promotion pending",
}

func TestEveryRegisteredKindHasWireArgsOrIsAllowlisted(t *testing.T) {
	promoted := map[string]bool{}
	for _, k := range featureargs.Kinds() {
		promoted[k] = true
	}
	registered := map[string]bool{}
	for _, d := range Default().All() {
		registered[d.Name] = true
		if promoted[d.Name] {
			continue
		}
		if _, ok := dynamicKinds[d.Name]; !ok {
			t.Errorf("feature kind %q has no api/wire/featureargs arg struct and is not in "+
				"dynamicKinds — promote it (define featureargs.X with Kind()==%q, make its "+
				"descriptor decode that type) or allowlist it with the tracking issue (#1616/#1709).",
				d.Name, d.Name)
		}
	}
	reportStaleDynamicKinds(t, registered, promoted)
}

// reportStaleDynamicKinds keeps dynamicKinds honest: an entry that is now promoted, or that
// names no registered kind, must be deleted.
func reportStaleDynamicKinds(t *testing.T, registered, promoted map[string]bool) {
	for kind, why := range dynamicKinds {
		if promoted[kind] {
			t.Errorf("dynamicKinds[%q] (%s) is stale — %q is promoted to featureargs now; delete the entry.", kind, why, kind)
		}
		if !registered[kind] {
			t.Errorf("dynamicKinds[%q] (%s) names no registered feature kind; delete the entry.", kind, why)
		}
	}
}
