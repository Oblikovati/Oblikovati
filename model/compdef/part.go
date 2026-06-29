// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/depend"
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
	// wholesaleParams are the dependency keys whose change must rebuild the whole feature
	// program because they reach geometry through a path the engine does not model
	// per-feature — a 3D-sketch dimension, a host-plane closure not yet attributed. Captured
	// each Recompute as (all keys read during recompute) − (those precisely attributable to a
	// sketch or a feature's direct reads), so any unmodelled path is conservatively wholesale
	// and a change never silently leaves stale geometry (Oblikovati#1414, ADR-0044).
	wholesaleParams map[depend.Key]bool
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
	d.work.SetFootprintTracker(params)                                                   // each work plane records its offset parameter, so an offset edit targets its hosted sketch (ADR-0044)
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
	// Capture every parameter read during the recompute (total footprint); the difference
	// against what is precisely attributable to a 2D sketch or a feature's direct reads is
	// the wholesale set a future parameter edit must rebuild the whole program for (#1414).
	total := d.params.TrackKeys(d.recomputeGeometry)
	d.recordWholesaleParams(total)
	// Drop any change records produced before now (parameters added while building
	// sketches/features, an earlier untargeted recompute) so the NEXT parameter edit sees
	// only its own changes and can target precisely (Oblikovati#1414).
	d.params.DrainChanged()
	d.MarkChanged()
}

// recomputeGeometry runs the feature program and syncs the resulting bodies; Recompute wraps
// it in a parameter-read capture (see Recompute).
func (d *PartComponentDefinition) recomputeGeometry() {
	// Re-solve sketches first so dimension expressions that reference parameters
	// (e.g. "od/2") propagate into the geometry: a parameter edit updates the
	// dimension's target value, and only a solve moves the profile to match.
	// Without this, features downstream rebuild on stale, pre-edit geometry.
	d.solveSketches()
	d.solveSketches3D()
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
}

// recordWholesaleParams stores (total reads − precisely-targetable reads) as the set a
// change must rebuild the whole program for. The precise set is every sketch's footprint
// (its own solve plus, for a work-plane-hosted sketch, the plane's offset footprint) plus
// every feature's direct reads — exactly what MarkDirtyForChange can attribute to a feature.
// Any other read during recompute is conservatively wholesale, so no path is ever silently
// skipped (Oblikovati#1414, ADR-0044). (3D-sketch dimensions are not in `total` today: 3D
// sketches are not re-solved during recompute — see solveSketches, 2D only — so their
// parameters are never read here. When 3D re-solving lands, its reads fall safely into the
// wholesale set until attributed to consuming features.)
func (d *PartComponentDefinition) recordWholesaleParams(total []depend.Key) {
	precise := map[depend.Key]bool{}
	for i := 0; i < d.sketches.Count(); i++ {
		for _, k := range d.sketches.Item(i).ParameterFootprint() {
			precise[k] = true
		}
	}
	for _, k := range d.features.DependencyReads() {
		precise[k] = true
	}
	d.wholesaleParams = map[depend.Key]bool{}
	for _, k := range total {
		if !precise[k] {
			d.wholesaleParams[k] = true
		}
	}
}

// RecomputeAfterChange rebuilds the part after a parameter value/equation/bool edit (and, in
// future, a cross-part adaptive-reference change — ADR-0044). Such a change can alter a feature's
// LIVE inputs — sketch dimensions, sheet-metal thicknesses, work-plane-offset closures — which the
// feature engine does not see as ordinary feature dependencies, so a plain Recompute would find
// nothing dirty and hand back the cached, pre-edit bodies (silent stale geometry). This is the
// single invalidation seam every change path shares — the UI verb, XML import, the wire router — so
// they cannot diverge (Oblikovati#1413). It takes no change-set argument: it drains its own change
// sources (the parameter graph today), keeping callers ignorant of attribution.
//
// It invalidates only the affected tail (Oblikovati#1414): the changed keys (and their transitive
// dependents) come from the parameter graph; if any reaches geometry through a path the engine
// cannot attribute to a feature (the wholesale set captured last recompute) it falls back to a full
// rebuild, otherwise it dirties only the features whose consumed-sketch footprint or direct reads
// the change touched. A work-plane offset is attributed through the hosted sketch's footprint, so
// editing one fillet's (or one offset's) driving parameter on a 1000-feature part rebuilds that
// feature's tail, not all 1000.
func (d *PartComponentDefinition) RecomputeAfterChange() {
	changed := d.params.DrainChangedKeys()
	if d.changeNeedsFullRebuild(changed) {
		d.features.MarkAllDirty()
	} else {
		d.features.MarkDirtyForChange(changed)
	}
	d.Recompute()
}

// changeNeedsFullRebuild reports whether a change must rebuild the whole program rather than a
// targeted tail: when nothing recorded as changed (an edit path that bypassed the graph, or the
// first edit after load — rebuild conservatively), or when a changed key is in the wholesale set
// (reaches geometry through an unmodelled path). Otherwise the changed keys are precisely
// attributable to features and MarkDirtyForChange handles them.
func (d *PartComponentDefinition) changeNeedsFullRebuild(changed []depend.Key) bool {
	if len(changed) == 0 {
		return true
	}
	for _, k := range changed {
		if d.wholesaleParams[k] {
			return true
		}
	}
	return false
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
// under-constrained one keeps its free DOFs. Each solve is wrapped in a parameter-read
// capture so the sketch records the dimension parameters that drive it (its footprint),
// letting a parameter edit rebuild only the features that consume an affected sketch
// (Oblikovati#1414).
func (d *PartComponentDefinition) solveSketches() {
	for i := 0; i < d.sketches.Count(); i++ {
		sk := d.sketches.Item(i)
		sk.SetParameterFootprint(d.params.TrackKeys(func() { sk.Solve() }))
	}
}

// solveSketches3D refreshes each 3D sketch's included reference geometry, then re-solves it,
// walking the collection in creation order. Refreshing includes BEFORE the solve is what makes
// a 3D sketch that dimensions against included geometry correct in a single recompute: its
// sources — a 2D sketch (already solved by solveSketches above) or an earlier 3D sketch
// (already solved earlier in this loop, creation order) — are current when the dependent
// sketch solves, so a dependency is computed before its dependents. Parameter-driven 3D
// dimensions then move geometry on recompute, mirroring solveSketches for 2D (Oblikovati#1566).
//
// The dimension reads bubble into the part's whole-recompute footprint (see Recompute's Track);
// not being attributable to a feature yet, they land in the wholesale set, so a 3D-dimension
// edit conservatively rebuilds the whole program — correct, with precise feature attribution a
// follow-up on the depend.Key seam (ADR-0044). Includes sourced from part EDGES are refreshed
// again post-features in refreshSketchReferences, where the body result exists.
func (d *PartComponentDefinition) solveSketches3D() {
	for i := 0; i < d.sketches3D.Count(); i++ {
		sk := d.sketches3D.Item(i)
		sk.UpdateIncluded()
		sk.Solve()
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

// WorkingScale is the centimetre size of one of the part's stored (working) length units
// (ADR-0042 Phase 2). The assembly placement walker reads it to convert a component placed
// into an assembly of a different working unit (1.0 ⇒ the centimetre default).
func (d *PartComponentDefinition) WorkingScale() float64 { return d.units.WorkingScale() }

// SetLengthUnit sets the document's preferred length unit (e.g. "mm", "in"). On a still-empty
// document it also centres the working scale on that unit (ADR-0042 Phase 2) so coordinates
// authored next are O(1) — the activation that lets a µm/pm or km document model without the
// cm-storage extremes. Once geometry exists the working scale is left alone (changing it would
// reinterpret stored coordinates); a unit change is then display-only.
func (d *PartComponentDefinition) SetLengthUnit(name string) error {
	if err := d.units.SetPreferred(param.Length, name); err != nil {
		return err
	}
	d.centerWorkingScaleIfEmpty(name)
	return nil
}

// SetUnits replaces the document's display units wholesale — the way the units API and the Units
// settings dialog apply an edited preferences object (build it from Units().Clone()). On a
// still-empty document whose incoming working scale is the centimetre default, it centres the
// working scale on the chosen length unit (ADR-0042 Phase 2); an explicitly-set working scale
// (e.g. via CenteredOnLength) is respected.
func (d *PartComponentDefinition) SetUnits(u param.UnitsOfMeasure) {
	d.units = u
	if u.WorkingScale() == 1 {
		d.centerWorkingScaleIfEmpty(u.PreferredName(param.Length))
	}
}

// centerWorkingScaleIfEmpty centres the working scale on the named length unit when the part has
// no modeled content yet, so re-scaling cannot reinterpret existing geometry (ADR-0042 Phase 2).
func (d *PartComponentDefinition) centerWorkingScaleIfEmpty(lengthUnit string) {
	if !d.isEmptyForRescale() {
		return
	}
	if ws, ok := param.RecommendedWorkingScale(lengthUnit); ok {
		_ = d.units.SetWorkingScale(ws)
	}
}

// isEmptyForRescale reports whether the part holds no modeled content yet, so re-centring the
// working scale cannot reinterpret existing geometry (ADR-0042 Phase 2).
func (d *PartComponentDefinition) isEmptyForRescale() bool {
	return d.features.Count() == 0 && len(d.bodies.All()) == 0 &&
		d.sketches.Count() == 0 && d.sketches3D.Count() == 0
}

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
