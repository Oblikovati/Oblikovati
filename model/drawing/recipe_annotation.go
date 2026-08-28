// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"encoding/hex"
	"strconv"

	"oblikovati.org/api/types"
)

// Drawing-recipe — the ANNOTATIONS section (M48 #2226 split of recipe.go). The YAML shape of one
// drawing annotation (GD&T frame, surface texture, balloon, note, revision/custom table, CoG marker,
// revision cloud) and its revision-table rows, plus the snapshot/restore of that section. A
// model-associative annotation re-derives its glyph on the next RecomputeViews; a purely-persisted one
// re-derives now from its recipe fields.

// annotationRecipe is the YAML shape of one drawing annotation. A CoG marker's glyph re-derives
// from its view's centroid on open; a revision cloud's scallops re-derive from its rectangle.
type annotationRecipe struct {
	Name     string  `yaml:"name"`
	Kind     string  `yaml:"kind"`
	ViewName string  `yaml:"viewName,omitempty"`
	X        float64 `yaml:"xmm,omitempty"`
	Y        float64 `yaml:"ymm,omitempty"`
	W        float64 `yaml:"widthMm,omitempty"`
	H        float64 `yaml:"heightMm,omitempty"`
	Tag      string  `yaml:"tag,omitempty"`
	EdgeKey  string  `yaml:"edgeKey,omitempty"`  // centre mark: circular edge; chamfer: edge A; bend: bend edge
	EdgeKeyB string  `yaml:"edgeKeyB,omitempty"` // chamfer note: edge B
	// feature control frame (GD&T):
	Characteristic string   `yaml:"characteristic,omitempty"`
	Tolerance      string   `yaml:"tolerance,omitempty"`
	Datums         []string `yaml:"datums,omitempty"`
	// surface texture: the material-removal variant (the roughness value reuses Tag).
	MaterialRemoval string `yaml:"materialRemoval,omitempty"`
	// revision table: the user-supplied change-history rows.
	Revisions []revisionRowRecipe `yaml:"revisions,omitempty"`
	// custom table: the column headers and data rows.
	Headers []string   `yaml:"headers,omitempty"`
	Rows    [][]string `yaml:"rows,omitempty"`
	// hole notes: the quantity-grouping mode ("" ⇒ perHole).
	HoleQuantity string `yaml:"holeQuantity,omitempty"`
}

// revisionRowRecipe is the YAML shape of one revision-table row.
type revisionRowRecipe struct {
	Revision    string `yaml:"revision"`
	Date        string `yaml:"date,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// annotationRecipesOf snapshots a sheet's annotations for persistence.
func annotationRecipesOf(sh *Sheet) []annotationRecipe {
	if sh.annotations == nil {
		return nil
	}
	out := make([]annotationRecipe, 0, len(sh.annotations.items))
	for _, a := range sh.annotations.items {
		out = append(out, annotationRecipe{
			Name: a.name, Kind: a.kind.String(), ViewName: a.viewName,
			X: a.x, Y: a.y, W: a.w, H: a.h, Tag: a.tag,
			EdgeKey: hex.EncodeToString(a.edgeKey), EdgeKeyB: hex.EncodeToString(a.edgeKeyB),
			Characteristic: a.characteristic.String(), Tolerance: a.tolerance, Datums: a.datums,
			MaterialRemoval: a.materialRemoval.String(), Revisions: revisionRowRecipesOf(a.revisions),
			Headers: a.headers, Rows: a.tableRows, HoleQuantity: holeQuantityString(a),
		})
	}
	return out
}

// holeQuantityString persists a hole note's grouping mode, and only that — "" for every other
// annotation and for the per-hole default, so recipes stay clean.
func holeQuantityString(a *DrawingAnnotation) string {
	if a.kind != types.HoleNoteAnnotation || a.holeQuantity == types.HoleNotePerHole {
		return ""
	}
	return a.holeQuantity.String()
}

// revisionRowRecipesOf snapshots a revision table's rows for persistence.
func revisionRowRecipesOf(rows []RevisionRow) []revisionRowRecipe {
	if len(rows) == 0 {
		return nil
	}
	out := make([]revisionRowRecipe, len(rows))
	for i, r := range rows {
		out[i] = revisionRowRecipe(r)
	}
	return out
}

// revisionRowsOf rebuilds a revision table's rows from its recipe.
func revisionRowsOf(recs []revisionRowRecipe) []RevisionRow {
	if len(recs) == 0 {
		return nil
	}
	out := make([]RevisionRow, len(recs))
	for i, r := range recs {
		out[i] = RevisionRow(r)
	}
	return out
}

// restoreAnnotations rebuilds a sheet's annotations from its recipe; CoG glyphs re-derive on the
// next RecomputeViews, revision-cloud scallops re-derive now from the rectangle.
func restoreAnnotations(sh *Sheet, recs []annotationRecipe) {
	if len(recs) == 0 {
		return
	}
	as := sh.Annotations()
	for _, ar := range recs {
		kind, _ := types.ParseDrawingAnnotationKind(ar.Kind)
		edgeKey, _ := hex.DecodeString(ar.EdgeKey)
		edgeKeyB, _ := hex.DecodeString(ar.EdgeKeyB)
		characteristic, _ := types.ParseGeometricCharacteristic(ar.Characteristic)
		holeQuantity, _ := types.ParseHoleNoteQuantity(ar.HoleQuantity)
		a := &DrawingAnnotation{name: ar.Name, kind: kind, viewName: ar.ViewName, x: ar.X, y: ar.Y, w: ar.W, h: ar.H, tag: ar.Tag, edgeKey: edgeKey, edgeKeyB: edgeKeyB,
			characteristic: characteristic, tolerance: ar.Tolerance, datums: ar.Datums, revisions: revisionRowsOf(ar.Revisions),
			headers: ar.Headers, tableRows: ar.Rows, holeQuantity: holeQuantity}
		restoreAnnotationGeometry(a, ar)
		as.items = append(as.items, a)
	}
}

// restoreAnnotationGeometry rebuilds a restored annotation's curves and labels from its recipe (the
// kinds whose glyph is a pure function of persisted fields; the model-associative ones re-derive on
// the next RecomputeViews instead).
func restoreAnnotationGeometry(a *DrawingAnnotation, ar annotationRecipe) {
	switch a.kind {
	case types.RevisionCloudAnnotation:
		a.curves = revisionCloudCurves(ar.X, ar.Y, ar.W, ar.H)
	case types.FeatureControlFrameAnnotation:
		a.curves, a.labels = featureControlFrameGeometry(ar.X, ar.Y, a.characteristic, ar.Tolerance, ar.Datums)
	case types.DatumFeatureAnnotation:
		a.curves, a.labels = datumFeatureGeometry(ar.X, ar.Y, ar.Tag)
	case types.SurfaceTextureAnnotation:
		variant, _ := types.ParseMaterialRemoval(ar.MaterialRemoval)
		a.materialRemoval = variant
		a.curves, a.labels = surfaceTextureGeometry(ar.X, ar.Y, ar.Tag, variant)
	case types.BalloonAnnotation:
		item, _ := strconv.Atoi(ar.Tag)
		a.curves, a.labels = balloonGeometry(ar.X, ar.Y, item, ar.W, ar.H)
	case types.RevisionTagAnnotation:
		a.curves, a.labels = revisionTagGeometry(ar.X, ar.Y, ar.Tag)
	case types.DrawingNoteAnnotation:
		a.curves, a.labels = noteGeometry(ar.X, ar.Y, ar.Tag, ar.W, ar.H)
	default:
		restoreAnnotationTableGeometry(a, ar)
	}
}

// restoreAnnotationTableGeometry rebuilds the user-supplied table annotations (revision and custom)
// from their persisted rows — split out of restoreAnnotationGeometry to keep each focused.
func restoreAnnotationTableGeometry(a *DrawingAnnotation, ar annotationRecipe) {
	switch a.kind {
	case types.RevisionTableAnnotation:
		a.rowCount = len(a.revisions)
		a.curves, a.labels = revisionTableGeometry(ar.X, ar.Y, a.revisions)
	case types.CustomTableAnnotation:
		a.rowCount = len(a.tableRows)
		a.curves, a.labels = customTableGeometry(ar.X, ar.Y, a.headers, a.tableRows)
	}
}
