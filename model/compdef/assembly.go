// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	"strconv"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// An assembly definition is a composite component — a placeable definition that owns
// its own occurrences — so it can nest inside a parent assembly; a part definition is
// a plain (leaf) placeable definition. These assertions pin both, the flyweight
// vocabulary the occurrence model places against (M11-F02).
var (
	_ occurrence.Composite  = (*AssemblyComponentDefinition)(nil)
	_ occurrence.Definition = (*PartComponentDefinition)(nil)
)

// AssemblyComponentDefinition is an assembly's modeling content — the assembly
// analogue of [PartComponentDefinition]. Where a part owns bodies, an assembly owns
// component occurrences (the placements of parts and sub-assemblies), its structure,
// a range box unioned from those occurrences, and a change-detection version. It is
// the reference API's AssemblyComponentDefinition (M11-F01, #345); constraints,
// joints, and representations attach to it from M12.
type AssemblyComponentDefinition struct {
	occurrences *occurrence.Occurrences
	units       param.UnitsOfMeasure  // document display units (length/angle/…)
	events      *AssemblyEvents       // occurrence-lifecycle event source (M11-F07)
	features    *AssemblyFeatures     // assembly-authored machining features (M11-F08)
	params      *param.Parameters     // parameter DAG for assembly sketch dimensions
	work        *feature.WorkGeometry // origin frame + user work planes, in assembly space
	sketches    *sketch.Sketches      // sketches authored in the assembly (profile inputs)
}

// NewAssemblyComponentDefinition returns an empty assembly content object: no
// occurrences, default (metric) display units, a live event source wired to its
// occurrence collection so placements/moves/suppression raise domain events (M11-F07),
// and an empty sketch/work-geometry surface so assembly features can be authored from
// profiles sketched in assembly space (the assembly sketching subsystem).
func NewAssemblyComponentDefinition() *AssemblyComponentDefinition {
	occ := occurrence.NewOccurrences()
	a := &AssemblyComponentDefinition{
		occurrences: occ,
		units:       param.DefaultUnitsOfMeasure(),
		events:      newAssemblyEvents(),
		features:    NewAssemblyFeatures(),
		params:      param.NewParameters(),
		work:        feature.NewWorkGeometry(),
		sketches:    sketch.NewSketches(),
	}
	occ.SetListener(a.events)
	a.features.SetBus(a.events.Bus()) // feature-program events ride the assembly's occurrence bus
	return a
}

// An assembly definition is plain document content, not recipe-bearing content: it
// has no recipe to persist until its occurrences reference documents (M11-F02), so
// it intentionally does NOT implement doc.RecipeContent (contrast PartComponentDefinition).
var _ doc.Content = (*AssemblyComponentDefinition)(nil)

// init registers the real assembly content with the document layer so opening or
// creating an assembly document yields a live AssemblyComponentDefinition, not the
// identity-only stub (see doc.RegisterContentFactory). The assembly does not yet
// implement doc.RecipeContent: its occurrences reference documents through the
// reference graph that arrives in M11-F02, so there is nothing recipe-persistable
// until then — today an assembly round-trips its identity (manifest) alone.
func init() {
	doc.RegisterContentFactory(doc.Assembly, func() doc.Content { return NewAssemblyComponentDefinition() })
}

// AddAssembly creates a new assembly document in ws with a realized assembly
// component definition installed (not the bare doc-package placeholder), makes it
// active, and returns it — the assembly counterpart of [AddPart], so the host's New
// Assembly path and documents.create go through one place.
func AddAssembly(ws *doc.Workspace, fullDocumentName string, visible bool) (*doc.Document, error) {
	d, err := ws.Add(doc.Assembly, fullDocumentName, visible)
	if err != nil {
		return nil, err
	}
	d.SetContent(NewAssemblyComponentDefinition())
	return d, nil
}

// DocumentType identifies this as assembly content, satisfying doc.Content.
func (a *AssemblyComponentDefinition) DocumentType() doc.DocumentType { return doc.Assembly }

// Occurrences returns the assembly's component occurrences — the parts and
// sub-assemblies placed in it. This is the live collection the assembly's structure
// and range box read from, and the one a parent assembly navigates into when this
// assembly is itself placed (nesting). It also makes the definition a Composite.
func (a *AssemblyComponentDefinition) Occurrences() *occurrence.Occurrences {
	return a.occurrences
}

// Place adds an occurrence of def to the assembly under name at transform, returning
// it — the assembly-level entry behind "place component". def is shared: place the
// same definition twice and both occurrences track its edits (the flyweight). To
// place an open document's component, use [PlaceComponent].
func (a *AssemblyComponentDefinition) Place(name string, def occurrence.Definition, transform math.Matrix4) *occurrence.Occurrence {
	return a.occurrences.AddByComponentDefinition(name, def, transform)
}

// PlaceComponent places the component held by componentDoc — an open part or assembly
// document — into the assembly under name at transform. The document's content
// definition is shared (the flyweight), so every placement tracks edits to that
// component. It errors if the content is not a placeable component definition (e.g. a
// reference stub or a drawing).
//
// NOTE: this links the in-memory definitions only. Recording the assembly→component
// document reference (the reference graph) and persisting placements arrive in a
// follow-up — see the assembly-persistence note on the package factory.
func (a *AssemblyComponentDefinition) PlaceComponent(name string, componentDoc *doc.Document, transform math.Matrix4) (*occurrence.Occurrence, error) {
	def, ok := componentDoc.Content().(occurrence.Definition)
	if !ok {
		return nil, fmt.Errorf("compdef: cannot place %q: its content %T is not a placeable component definition",
			componentDoc.DisplayName(), componentDoc.Content())
	}
	return a.Place(name, def, transform), nil
}

// Events returns the assembly's occurrence-lifecycle event source — subscribe add-ins
// and observers to its bus (M11-F07). Placements, moves, replacements and suppression
// changes raise typed events; delete and suppress are also vetoable in the Before
// phase via [AssemblyComponentDefinition.DeleteOccurrence] and SetOccurrenceSuppressed.
func (a *AssemblyComponentDefinition) Events() *AssemblyEvents { return a.events }

// Features returns the assembly's machining-feature program — the features authored in
// the assembly that cut/modify placed component geometry in place (M11-F08). Add a
// feature with [AssemblyComponentDefinition.AddFeature] so it picks up default
// participation; evaluate the program with [AssemblyComponentDefinition.RecomputeFeatures].
func (a *AssemblyComponentDefinition) Features() *AssemblyFeatures { return a.features }

// AddFeature appends an assembly machining feature wrapping f, defaulting its
// participation to every component currently present (the reference API's behavior:
// components added later do not participate unless added to the feature). It returns
// the hosted feature so the caller can adjust participation or suppression.
func (a *AssemblyComponentDefinition) AddFeature(f feature.Feature) *AssemblyFeature {
	return a.features.Add(f, distinctSources(a.PlacedBodies()))
}

// Parameters returns the assembly's parameter DAG (shared with its sketches so
// dimension expressions resolve against the same table).
func (a *AssemblyComponentDefinition) Parameters() *param.Parameters { return a.params }

// WorkGeometry returns the assembly's origin coordinate frame and user work
// planes/axes/points, expressed in assembly space — the planes a sketch is authored on.
func (a *AssemblyComponentDefinition) WorkGeometry() *feature.WorkGeometry { return a.work }

// WorkPlanes returns the assembly's datum planes (origin + user), so the shared sketch
// surface can resolve "sketch on work plane N" the same way it does for a part.
func (a *AssemblyComponentDefinition) WorkPlanes() *feature.WorkPlanes { return a.work.WorkPlanes() }

// Sketches returns the assembly's planar sketches — the profile inputs an assembly
// feature (e.g. an extrude) is authored from, sketched in assembly space.
func (a *AssemblyComponentDefinition) Sketches() *sketch.Sketches { return a.sketches }

// AddSketch creates a sketch on plane and shares the assembly's parameter DAG so its
// dimensions resolve against the same table (mirroring the part). Pass a host closure
// (non-nil) to track a moving work plane so the sketch follows it.
func (a *AssemblyComponentDefinition) AddSketch(plane sketch.Plane, host func() sketch.Plane) *sketch.Sketch {
	sk := a.sketches.Add(plane)
	sk.SetParameters(a.params)
	if host != nil {
		sk.SetPlaneHost(host)
	}
	return sk
}

// RecomputeFeatures re-solves the assembly's sketches and work geometry, then evaluates
// the feature program against the current placed geometry — machining each unsuppressed
// feature into its participants' assembly-space bodies (not the shared part definitions)
// up to the end-of-features marker. Solving first means a profile-based feature (an
// extrude) reads an up-to-date profile. Read per-occurrence results with [AssemblyFeatures.Result].
func (a *AssemblyComponentDefinition) RecomputeFeatures() {
	a.solveSketches()
	a.work.Recompute(nil) // origin + offset planes need no body; tangent-to-face planes are part-only
	a.refreshSketchPlanes()
	a.features.Recompute(a.PlacedBodies())
}

// solveSketches re-solves every assembly sketch so parameter-driven dimensions move the
// geometry before a profile is read.
func (a *AssemblyComponentDefinition) solveSketches() {
	for i := 0; i < a.sketches.Count(); i++ {
		a.sketches.Item(i).Solve()
	}
}

// refreshSketchPlanes re-reads each sketch's host work plane so sketches on datum planes
// follow them when they move.
func (a *AssemblyComponentDefinition) refreshSketchPlanes() {
	for i := 0; i < a.sketches.Count(); i++ {
		a.sketches.Item(i).RefreshPlane()
	}
}

// DeleteOccurrence removes o from the assembly, first raising a vetoable OccurrenceDelete
// in the Before phase; on commit the removal raises OccurrenceDelete After. It returns
// an [AssemblyVetoError] if a Before handler cancelled the delete, or nil otherwise
// (removing an occurrence not in this assembly is a silent no-op, as for the reference
// API). M11-F07.
func (a *AssemblyComponentDefinition) DeleteOccurrence(o *occurrence.Occurrence) error {
	if err := raiseBefore(a.events.bus, "delete occurrence", OccurrenceDelete{Occurrence: o}); err != nil {
		return err
	}
	a.occurrences.Remove(o)
	return nil
}

// SetOccurrenceSuppressed toggles o's suppression, first raising a vetoable
// OccurrenceSuppress in the Before phase; on commit the change raises OccurrenceSuppress
// After. A Before handler may veto (e.g. to keep a required component active), in which
// case it returns an [AssemblyVetoError] and o is unchanged. A no-op toggle (already in
// the requested state) raises no event and returns nil. M11-F07.
func (a *AssemblyComponentDefinition) SetOccurrenceSuppressed(o *occurrence.Occurrence, suppressed bool) error {
	if o.Suppressed() == suppressed {
		return nil
	}
	ev := OccurrenceSuppress{Occurrence: o, Suppressed: suppressed}
	if err := raiseBefore(a.events.bus, "suppress occurrence", ev); err != nil {
		return err
	}
	o.SetSuppressed(suppressed)
	return nil
}

// RangeBox returns the axis-aligned bounding box enclosing every unsuppressed
// occurrence (empty when the assembly has none).
func (a *AssemblyComponentDefinition) RangeBox() math.Box { return a.occurrences.RangeBox() }

// PreciseRangeBox returns the tight bounding box. With axis-aligned occurrence boxes
// it equals [RangeBox]; it tightens once occurrences expose curved-face evaluation
// (kernel phase B), mirroring the part.
func (a *AssemblyComponentDefinition) PreciseRangeBox() math.Box { return a.RangeBox() }

// OrientedMinimumRangeBox returns an oriented bounding box. Phase A returns the
// axis-aligned box as an oriented box; the true minimum-volume OBB is a later
// optimization, as for the part.
func (a *AssemblyComponentDefinition) OrientedMinimumRangeBox() math.OrientedBox {
	box := a.RangeBox()
	half := box.Diagonal().Scale(0.5)
	x, _ := math.NewUnitVector3(1, 0, 0)
	y, _ := math.NewUnitVector3(0, 1, 0)
	z, _ := math.NewUnitVector3(0, 0, 1)
	return math.NewOrientedBox(box.Center(), x, y, z, [3]math.Scalar{half.X, half.Y, half.Z})
}

// Units returns the document's display units (default metric — mm).
func (a *AssemblyComponentDefinition) Units() param.UnitsOfMeasure { return a.units }

// SetLengthUnit sets the assembly's preferred length unit (e.g. "mm", "in").
func (a *AssemblyComponentDefinition) SetLengthUnit(name string) error {
	return a.units.SetPreferred(param.Length, name)
}

// ModelGeometryVersion is a string that changes whenever the assembly's occurrences
// change (add/remove/move/suppress), so consumers (drawings, parent assemblies) can
// detect when they must update. It is derived from the occurrence collection's
// revision, the assembly analogue of the part's edit counter.
func (a *AssemblyComponentDefinition) ModelGeometryVersion() string {
	return "v" + strconv.FormatUint(a.occurrences.Revision(), 10)
}
