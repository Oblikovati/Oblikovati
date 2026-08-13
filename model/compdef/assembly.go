// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	"strconv"

	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/attr"
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
	units       param.UnitsOfMeasure    // document display units (length/angle/…)
	events      *AssemblyEvents         // occurrence-lifecycle event source (M11-F07)
	constraints *assembly.ConstraintSet // constraint relationships + positioning solver (M12-F01)
	joints      *assembly.JointSet      // joint relationships (reduced-DOF, M12-F02)
	dsJoints    *assembly.DSJointSet    // DS-joint (DOF/imposed-motion) view (M12-F02)

	representations *assembly.Representations // design-view/positional/LOD override layers + model states (M12-F04)
	contacts        *assembly.ContactSolver   // contact sets + interpenetration toggle (M12-F05)
	features        *AssemblyFeatures         // assembly-authored machining features (M11-F08)
	params          *param.Parameters         // parameter DAG for assembly sketch dimensions
	work            *feature.WorkGeometry     // origin frame + user work planes, in assembly space
	sketches        *sketch.Sketches          // sketches authored in the assembly (profile inputs)
	// pending holds occurrence records parsed by ApplyRecipe but not yet bound to live
	// component definitions — ApplyRecipe has no workspace, so binding waits for
	// ResolveReferences once the document is registered (#715). Empty outside a reopen.
	pending []occurrenceRecipe
	// pendingFeatures holds the machining program parsed by ApplyRecipe but not yet rebuilt —
	// features bind after the occurrences resolve (they snapshot participation), so they wait
	// for ResolveReferences like the occurrences do (#785). Empty outside a reopen/restore.
	pendingFeatures []assemblyFeatureProgramRecipe
	// props are the assembly document's iProperties (metadata sets), like a part's (#156).
	props *attr.PropertySets
	// options are the assembly-editing defaults (#1981); updatePending records a recompute deferred
	// while options.DeferUpdate was set, flushed when it is cleared.
	options       AssemblyOptions
	updatePending bool
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
		props:       attr.NewPropertySets(),
		options:     defaultAssemblyOptions(),
	}
	occ.SetListener(a.events)
	a.features.SetBus(a.events.Bus())    // feature-program events ride the assembly's occurrence bus
	a.sketches.ShareParameters(a.params) // restored sketches bind dimension expressions before their dims rebuild (#1557)
	a.constraints = assembly.NewConstraintSet(occ, a.events)
	a.joints = assembly.NewJointSet(occ, a.events)
	a.dsJoints = assembly.NewDSJointSet()
	a.representations = assembly.NewRepresentations(occ, a.constraints, a.joints)
	a.contacts = assembly.NewContactSolver()
	// Accept occurrence-qualified datum refs ("occ/<path>/plane/N") as inputs to the assembly's
	// own sketch/feature tools, resolved through the occurrence tree + transform (#1857).
	a.work.SetExternalDatumResolver(occurrenceDatumResolver{asm: a})
	return a
}

// An assembly persists its occurrence structure as a recipe and rebinds each occurrence
// to its component document through the reference graph on reopen (#715), so it is both
// recipe-bearing content and a reference resolver (the marshal/apply live in
// assembly_serialize.go).
var (
	_ doc.RecipeContent     = (*AssemblyComponentDefinition)(nil)
	_ doc.ReferenceResolver = (*AssemblyComponentDefinition)(nil)
)

// The real assembly content reaches the document layer through the composition
// root (model/contentset.Default → doc.NewWorkspace, #1617). Its recipe
// (assembly_serialize.go) restores the occurrence structure; the component
// documents those occurrences instance are resolved through the reference graph
// after the assembly is registered (#715).

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
	o := a.occurrences.AddByComponentDefinition(name, def, transform)
	a.groundFirstComponent(o) // #1981
	return o
}

// PlaceComponent places the component held by componentDoc — an open part or assembly
// document — into the assembly under name at transform. The document's content
// definition is shared (the flyweight), so every placement tracks edits to that
// component. It errors if the content is not a placeable component definition (e.g. a
// reference stub or a drawing).
//
// NOTE: this links the in-memory definitions only — the occurrence carries no persistent
// component link. Use [PlaceComponentFromFile] to place an occurrence that survives a
// save/reopen (it records the assembly→component reference and the document name).
func (a *AssemblyComponentDefinition) PlaceComponent(name string, componentDoc *doc.Document, transform math.Matrix4) (*occurrence.Occurrence, error) {
	def, err := placeableDefinition(componentDoc)
	if err != nil {
		return nil, err
	}
	return a.Place(name, def, transform), nil
}

// placeableDefinition extracts the component definition a document holds, erroring when
// its content is not placeable (a reference stub, a drawing). Shared by the in-memory and
// file-backed place paths.
func placeableDefinition(componentDoc *doc.Document) (occurrence.Definition, error) {
	def, ok := componentDoc.Content().(occurrence.Definition)
	if !ok {
		return nil, fmt.Errorf("compdef: cannot place %q: its content %T is not a placeable component definition",
			componentDoc.DisplayName(), componentDoc.Content())
	}
	return def, nil
}

// PlaceComponentFromFile places componentDoc into owner's assembly under name at
// transform AS A PERSISTENT placement: the occurrence records the component's document
// name, and owner records the assembly→component reference (deduped) so the placement is
// saved and restored across a reopen (#715). owner is the assembly's own document (the
// one whose content is this definition); componentDoc is the component to instance.
func (a *AssemblyComponentDefinition) PlaceComponentFromFile(owner, componentDoc *doc.Document, name string, transform math.Matrix4) (*occurrence.Occurrence, error) {
	def, err := placeableDefinition(componentDoc)
	if err != nil {
		return nil, err
	}
	componentName := componentDoc.FullDocumentName()
	occ := a.occurrences.AddByComponentName(name, def, componentName, transform)
	a.groundFirstComponent(occ) // #1981
	// Record (and resolve to the already-open) reference so save snapshots the edge.
	owner.OpenReference(componentName)
	return occ, nil
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

// Constraints returns the assembly's relationship set — the mate/flush/angle/... that
// position occurrences relative to each other (M12-F01). Author with its Add* methods,
// then position the components with [AssemblyComponentDefinition.SolveConstraints].
func (a *AssemblyComponentDefinition) Constraints() *assembly.ConstraintSet { return a.constraints }

// Joints returns the assembly's joint set — the simplified joints (rigid/rotational/…) that
// establish a degree-of-freedom set between occurrences (M12-F02). Author with its Add*
// methods, then position with [AssemblyComponentDefinition.SolveConstraints].
func (a *AssemblyComponentDefinition) Joints() *assembly.JointSet { return a.joints }

// DSJoints returns the assembly's DS-joint (degrees-of-freedom / imposed-motion) set (M12-F02).
func (a *AssemblyComponentDefinition) DSJoints() *assembly.DSJointSet { return a.dsJoints }

// Representations returns the assembly's representation hub — the design-view, positional, and
// level-of-detail override layers plus model states (M12-F04).
func (a *AssemblyComponentDefinition) Representations() *assembly.Representations {
	return a.representations
}

// ContactSolver returns the assembly's contact solver — the contact sets and the
// interpenetration-on-drag toggle (M12-F05).
func (a *AssemblyComponentDefinition) ContactSolver() *assembly.ContactSolver {
	return a.contacts
}

// SolveConstraints positions the assembly's occurrences to satisfy BOTH its constraints and
// its joints in one solve and returns the health/DOF report (M12-F01/F02 — joints are
// reduced-DOF residual bundles on the same solver, ADR-0011). It is the assembly analogue of
// recomputing a part: a relationship edit or a component move calls it to re-resolve.
func (a *AssemblyComponentDefinition) SolveConstraints() assembly.SolveReport {
	return assembly.SolveAssembly(a.constraints, a.joints)
}

// AssemblyHealth reports the assembly's combined constraint+joint health and DOF without
// moving any occurrence.
func (a *AssemblyComponentDefinition) AssemblyHealth() assembly.SolveReport {
	return assembly.AssemblyHealth(a.constraints, a.joints)
}

// DriveJoint sweeps the joint with the given id through settings — re-solving the assembly at
// each step with the driven variable pinned — and returns the resulting motion frames
// (M12-F03, Oblikovati/Oblikovati#366). It is non-destructive: the assembly is left in its
// pre-drive pose. Errors when the joint is unknown or its kind/variable is not drivable.
func (a *AssemblyComponentDefinition) DriveJoint(jointID uint64, settings assembly.DriveSettings) (assembly.DriveResult, error) {
	return assembly.DriveJoint(a.occurrences, a.constraints, a.joints, jointID, settings)
}

// AddFeature appends an assembly machining feature wrapping f, defaulting its
// participation to every component currently present (the reference API's behavior:
// components added later do not participate unless added to the feature). The
// aggregate owns the attach policy so no driver can miss it (#1612, audit B1):
// the feature is named uniquely by kind (a load path overrides with the
// persisted name), and a proxy cut never machines its own source component. It
// returns the hosted feature so the caller can adjust participation or suppression.
func (a *AssemblyComponentDefinition) AddFeature(f feature.Feature) *AssemblyFeature {
	af := a.features.Add(f, distinctSources(a.PlacedBodies()))
	af.SetName(a.features.UniqueName(af.Kind()))
	if pc, ok := f.(*feature.AssemblyProxyCutFeature); ok {
		af.RemoveParticipant(pc.Source())
	}
	return af
}

// Parameters returns the assembly's parameter DAG (shared with its sketches so
// dimension expressions resolve against the same table).
func (a *AssemblyComponentDefinition) Parameters() *param.Parameters { return a.params }

// Properties returns the assembly's iProperties (document metadata sets), like a part's (#156).
func (a *AssemblyComponentDefinition) Properties() *attr.PropertySets { return a.props }

// Recompute re-solves the assembly (its sketches, work geometry, and feature program) — the
// sketch-host-uniform name the sketch environment calls on Finish, aliasing RecomputeFeatures so
// a part and an assembly present one recompute method (#766).
func (a *AssemblyComponentDefinition) Recompute() { a.RecomputeFeatures() }

// RecomputeAfterChange re-solves the assembly after a parameter edit (or, later, a cross-part
// adaptive-reference change). An assembly has no incremental fast path (unlike a part, #1414), so
// this is a full re-solve. Wholesale is a VALID point on the [ParameterHolder] invalidation
// contract — "attribute nothing, rebuild everything" — so the seam exists and is exercised now,
// and the adaptivity milestone narrows it from inside without reshaping the seam (M39-F02 #1558,
// ADR-0044).
func (a *AssemblyComponentDefinition) RecomputeAfterChange() { a.RecomputeFeatures() }

// WorkGeometry returns the assembly's origin coordinate frame and user work
// planes/axes/points, expressed in assembly space — the planes a sketch is authored on.
func (a *AssemblyComponentDefinition) WorkGeometry() *feature.WorkGeometry { return a.work }

// WorkPlanes returns the assembly's datum planes (origin + user), so the shared sketch
// surface can resolve "sketch on work plane N" the same way it does for a part.
func (a *AssemblyComponentDefinition) WorkPlanes() *feature.WorkPlanes { return a.work.WorkPlanes() }

// WorkAxes returns the assembly's datum axes (origin + user), mirroring the part so the work-
// feature wire surface can author and list them against an assembly.
func (a *AssemblyComponentDefinition) WorkAxes() *feature.WorkAxes { return a.work.WorkAxes() }

// WorkPoints returns the assembly's datum points (origin + user), mirroring the part so the work-
// feature wire surface (e.g. workPoints.create) can author them against an assembly.
func (a *AssemblyComponentDefinition) WorkPoints() *feature.WorkPoints { return a.work.WorkPoints() }

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
	if a.options.DeferUpdate {
		a.updatePending = true // batched until DeferUpdate is cleared (#1981)
		return
	}
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

// WorkingScale is the centimetre size of one of the assembly's stored (working) length units
// (ADR-0042 Phase 2) — the owner scale the placement walker converts each component into, and
// the scale a parent assembly converts THIS sub-assembly's frame from (1.0 ⇒ centimetre).
func (a *AssemblyComponentDefinition) WorkingScale() float64 { return a.units.WorkingScale() }

// SetLengthUnit sets the assembly's preferred length unit (e.g. "mm", "in").
func (a *AssemblyComponentDefinition) SetLengthUnit(name string) error {
	return a.units.SetPreferred(param.Length, name)
}

// SetUnits replaces the document's display units wholesale (build from
// Units().Clone()).
func (a *AssemblyComponentDefinition) SetUnits(u param.UnitsOfMeasure) { a.units = u }

// ModelGeometryVersion is a string that changes whenever the assembly's occurrences
// change (add/remove/move/suppress), so consumers (drawings, parent assemblies) can
// detect when they must update. It is derived from the occurrence collection's
// revision, the assembly analogue of the part's edit counter.
func (a *AssemblyComponentDefinition) ModelGeometryVersion() string {
	return "v" + strconv.FormatUint(a.occurrences.Revision(), 10)
}
