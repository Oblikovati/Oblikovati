// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/material"
	"oblikovati.org/model/param"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/model/sheetmetal"
	"oblikovati.org/model/sketch"
)

// endOfPartAtEnd is the EOP marker value meaning "evaluate the whole feature program".
const endOfPartAtEnd = -1

// PartComponentDefinition is a part's modeling content. It is the container the
// feature engine fills (sketches → features → bodies); here it owns the result
// (bodies), the inputs (parameters, sketches), the bounding boxes, a change-
// detection version, and the rollback marker.
type PartComponentDefinition struct {
	bodies     *topo.SurfaceBodies
	params     *param.Parameters
	sketches   *sketch.Sketches
	sketches3D *sketch.Sketches3D
	features   *feature.PartFeatures
	keys       *identity.KeyManager
	work       *feature.WorkGeometry // origin coordinate frame + user work planes/axes/points
	surfaces   *feature.WorkSurfaces // construction surfaces wrapping the result's sheet bodies (M20-F16)
	// pointClouds are the part's attached laser-scan / photogrammetry references (M17-F06,
	// #645): transformable display objects a design is modeled against, not B-rep geometry. Their
	// scan bytes live in resources (ADR-0031); the cloud cites the resource UUID. Document-level
	// input, NOT part of the recipe — they survive a recipe reset (undo/reopen).
	pointClouds *pointcloud.PointClouds
	units       param.UnitsOfMeasure // document display units (length/angle/…)
	version     uint64
	eop         int // end-of-part feature index; endOfPartAtEnd ⇒ full program
	// assignments survive Recompute (keyed by persistent reference key, not body id), so a
	// material/appearance stays put when the body it is on is regenerated.
	assignments *material.AssignmentStore
	// assets are the document's embedded appearance/material copies (the non-built-in
	// assets it uses), making the .obk self-contained (ADR-0022).
	assets *material.AssetSet
	// resources are the document's embedded imported files (meshes, STEP, fonts), keyed by a
	// per-import UUID that features cite, making the .obk self-contained (ADR-0031). Document-
	// level input, NOT part of the recipe — it survives a recipe reset (undo/reopen).
	resources map[string]doc.Resource
	// props are the document's iProperties (metadata sets) — title/author/part number/custom,
	// which feed BOMs and title blocks (#156).
	props *attr.PropertySets
	// sheetMetal is the active sheet-metal rule when this part is in the sheet-metal
	// environment (M13-F01), else nil. A sheet-metal part is still a PartComponentDefinition
	// (so every part feature works on it); the rule adds the constant-thickness invariant and
	// the unfold method that bend/flat-pattern features consult. It is parameter-backed — its
	// thickness/bend-radius read the part's Thickness/BendRadius parameters.
	sheetMetal *sheetmetal.Rule
	// flatOrientations holds the part's flat-pattern orientations (M13-F05) — named alignment
	// states that frame the developed flat. Seeded with the default orientation when the part
	// enters the sheet-metal environment; nil for ordinary parts.
	flatOrientations *sheetmetal.Orientations
	// flatSettings is the per-part flat-pattern settings (M13-F05): currently the deferred-update
	// flag so a heavy flat only develops on demand.
	flatSettings sheetmetal.FlatPatternSettings
	// bendOrder is the press-brake bend sequence (M13-F06) as an overlay of bend feature names;
	// empty means the natural (creation) order. Bends not listed follow the listed ones.
	bendOrder []string
	// centerlines are the flat pattern's cosmetic centerlines (M13-F06) — annotation lines that
	// persist with the part.
	centerlines []sheetmetal.CosmeticCenterline
}

// NewPartComponentDefinition returns an empty part content object with its feature
// engine wired to the part's parameters and key manager.
func NewPartComponentDefinition() *PartComponentDefinition {
	params := param.NewParameters()
	keys := identity.NewKeyManager()
	d := &PartComponentDefinition{
		bodies:      topo.NewSurfaceBodies(),
		params:      params,
		sketches:    sketch.NewSketches(),
		sketches3D:  sketch.NewSketches3D(),
		features:    feature.NewPartFeatures(params, keys),
		keys:        keys,
		work:        feature.NewWorkGeometry(),
		surfaces:    feature.NewWorkSurfaces(),
		units:       param.DefaultUnitsOfMeasure(),
		eop:         endOfPartAtEnd,
		assignments: material.NewAssignmentStore(),
		assets:      material.NewAssetSet(),
		resources:   map[string]doc.Resource{},
		props:       attr.NewPropertySets(),
		pointClouds: pointcloud.NewPointClouds(),
	}
	d.features.SetResourceStore(d)                                                       // re-derive imported bodies from the resource table on open
	d.features.SetFontResolver(d)                                                        // resolve text/emboss fonts from embedded/app-provided resources
	d.features.SetWorkingScaleResolver(func() float64 { return d.units.WorkingScale() }) // re-import scales into the doc working unit
	d.sketches.ShareParameters(params)                                                   // dimension expressions resolve against user params (live + restore)
	d.sketches3D.ShareParameters(params)                                                 // same for 3D sketches
	return d
}

// Assignments returns the part's material/appearance assignment store. It is keyed by
// persistent reference keys, so it is unaffected by Recompute.
func (d *PartComponentDefinition) Assignments() *material.AssignmentStore { return d.assignments }

// Assets returns the part's embedded appearance/material set (the document-owned copies of
// the non-built-in assets it uses).
func (d *PartComponentDefinition) Assets() *material.AssetSet { return d.assets }

// Properties returns the part's iProperties (document metadata sets) — the standard Summary /
// Document Summary / Design Tracking sets plus user-defined custom properties (#156).
func (d *PartComponentDefinition) Properties() *attr.PropertySets { return d.props }

// AddPart creates a new part document in ws with a realized part component
// definition installed (not the bare doc-package placeholder), makes it active, and
// returns it. Callers that need a usable part — the head's New Part menu and demo
// seed, the automation bridge's documents.create — go through here so the
// Add-then-SetContent pairing lives in one place.
func AddPart(ws *doc.Workspace, fullDocumentName string, visible bool) (*doc.Document, error) {
	d, err := ws.Add(doc.Part, fullDocumentName, visible)
	if err != nil {
		return nil, err
	}
	d.SetContent(NewPartComponentDefinition())
	return d, nil
}

// Features returns the part's feature program (the recompute engine).
func (d *PartComponentDefinition) Features() *feature.PartFeatures { return d.features }

// Recompute runs the feature program and syncs the resulting bodies into
// SurfaceBodies, advancing the geometry version. This is the part-level
// orchestration that turns the feature history into the evaluated result.
func (d *PartComponentDefinition) Recompute() {
	// Re-solve sketches first so dimension expressions that reference parameters
	// (e.g. "od/2") propagate into the geometry: a parameter edit updates the
	// dimension's target value, and only a solve moves the profile to match.
	// Without this, features downstream rebuild on stale, pre-edit geometry.
	d.solveSketches()
	// Two passes around the feature program: the first so features can reference work
	// axes/planes (e.g. a revolve axis); the second so surface-tangent work planes can
	// resolve their picked faces against the freshly built body. The first pass sees the
	// previous result, which is correct for body-independent work features and harmless
	// for body-dependent ones (refreshed by the second pass).
	d.work.Recompute(d.features.Result())
	// Carry sketches with their host work planes before features consume them, so a
	// moved work plane (e.g. its offset parameter changed) reshapes dependent features.
	d.refreshSketchPlanes()
	d.features.Recompute()
	d.work.Recompute(d.features.Result())
	d.refreshSketchPlanes()
	d.bodies = topo.NewSurfaceBodies()
	for _, b := range d.features.Result() {
		d.bodies.Add(b)
	}
	d.surfaces.Sync(d.features.Result()) // gather the result's sheet bodies as work surfaces (M20-F16)
	d.refreshSketchReferences()
	d.MarkChanged()
}

// refreshSketchPlanes re-reads each sketch's host work plane (if any), so sketches on
// datum planes follow them when they move.
func (d *PartComponentDefinition) refreshSketchPlanes() {
	for i := 0; i < d.sketches.Count(); i++ {
		d.sketches.Item(i).RefreshPlane()
	}
}

// solveSketches re-solves every 2D sketch so parameter-driven dimensions move the
// geometry on recompute. Solving a fully-constrained sketch is idempotent; an
// under-constrained one keeps its free DOFs.
func (d *PartComponentDefinition) solveSketches() {
	for i := 0; i < d.sketches.Count(); i++ {
		d.sketches.Item(i).Solve()
	}
}

// refreshSketchReferences re-projects each sketch's reference geometry (2D projections /
// 3D includes) against the freshly rebuilt B-rep, so included geometry tracks the model
// associatively (M22). Self-resolving sources re-bind by reference key; a lost reference
// freezes its geometry. Surface-derived 3D curves are lazy (they re-resolve on Evaluate),
// so they need no refresh here.
func (d *PartComponentDefinition) refreshSketchReferences() {
	for i := 0; i < d.sketches.Count(); i++ {
		d.sketches.Item(i).UpdateProjections()
	}
	for i := 0; i < d.sketches3D.Count(); i++ {
		d.sketches3D.Item(i).UpdateIncluded()
	}
}

// DocumentType identifies this as part content, satisfying doc.Content.
func (d *PartComponentDefinition) DocumentType() doc.DocumentType { return doc.Part }

// SurfaceBodies returns the part's bodies (the evaluated result).
func (d *PartComponentDefinition) SurfaceBodies() *topo.SurfaceBodies { return d.bodies }

// Parameters returns the part's parameter set (shared with its sketches/features).
func (d *PartComponentDefinition) Parameters() *param.Parameters { return d.params }

// Units returns the document's display units (default metric — mm). The sketch grid
// and dimensions present values in these units.
func (d *PartComponentDefinition) Units() param.UnitsOfMeasure { return d.units }

// SetLengthUnit sets the document's preferred length unit (e.g. "mm", "in").
func (d *PartComponentDefinition) SetLengthUnit(name string) error {
	return d.units.SetPreferred(param.Length, name)
}

// SetUnits replaces the document's display units wholesale — the way the units
// API and the Units settings dialog apply an edited preferences object (build
// it from Units().Clone()).
func (d *PartComponentDefinition) SetUnits(u param.UnitsOfMeasure) { d.units = u }

// FeatureScaleWarning returns a non-empty diagnostic when a feature of the given size (in the
// part's working unit) is below what this model can resolve at its current extent, and "" when
// it is fine (ADR-0042 §Phase 2). float64 caps a single model at ~15 orders of magnitude, so a
// feature far smaller than the part would be silently merged; the head/API surfaces this so the
// user re-scales or splits the design rather than losing the feature. An empty part (no extent)
// never warns — there is nothing to be dwarfed by yet.
func (d *PartComponentDefinition) FeatureScaleWarning(featureSize float64) string {
	box := d.RangeBox()
	if box.IsEmpty() {
		return ""
	}
	return geom.SpanCeilingWarning(box, featureSize)
}

// Sketches returns the part's planar (2D) sketches.
func (d *PartComponentDefinition) Sketches() *sketch.Sketches { return d.sketches }

// PointClouds returns the part's attached scan collection (M17-F06, #645).
func (d *PartComponentDefinition) PointClouds() *pointcloud.PointClouds { return d.pointClouds }

// SketchBlocks returns the part's block-definition registry — the reference
// API's SketchBlocks on the component definition (M06-F07, #622).
func (d *PartComponentDefinition) SketchBlocks() *sketch.BlockDefinitions {
	return d.sketches.BlockDefinitions()
}

// Sketches3D returns the part's non-planar (3D) sketches.
func (d *PartComponentDefinition) Sketches3D() *sketch.Sketches3D { return d.sketches3D }

// RangeBox returns the axis-aligned bounding box of all bodies (empty if none).
func (d *PartComponentDefinition) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, b := range d.bodies.All() {
		box = box.Union(b.RangeBox())
	}
	return box
}

// PreciseRangeBox returns the tight bounding box. With analytic faces it equals
// [RangeBox]; once curved faces bulge beyond their vertices it tightens via
// evaluation (kernel phase B).
func (d *PartComponentDefinition) PreciseRangeBox() math.Box { return d.RangeBox() }

// OrientedMinimumRangeBox returns an oriented bounding box. Phase A returns the
// axis-aligned box as an oriented box; the true minimum-volume OBB is a later
// optimization.
func (d *PartComponentDefinition) OrientedMinimumRangeBox() math.OrientedBox {
	box := d.RangeBox()
	half := box.Diagonal().Scale(0.5)
	x, _ := math.NewUnitVector3(1, 0, 0)
	y, _ := math.NewUnitVector3(0, 1, 0)
	z, _ := math.NewUnitVector3(0, 0, 1)
	return math.NewOrientedBox(box.Center(), x, y, z, [3]math.Scalar{half.X, half.Y, half.Z})
}

// ModelGeometryVersion is a string that changes on every model edit, so consumers
// (assemblies, drawings) can detect when they must update. Bumped by [MarkChanged].
func (d *PartComponentDefinition) ModelGeometryVersion() string {
	return "v" + strconv.FormatUint(d.version, 10)
}

// MarkChanged records a model edit, advancing the geometry version.
func (d *PartComponentDefinition) MarkChanged() { d.version++ }
