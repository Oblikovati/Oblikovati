// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/identity"
	"github.com/Oblikovati/oblikovati/model/material"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// endOfPartAtEnd is the EOP marker value meaning "evaluate the whole feature program".
const endOfPartAtEnd = -1

// PartComponentDefinition is a part's modeling content. It is the container the
// feature engine fills (sketches → features → bodies); here it owns the result
// (bodies), the inputs (parameters, sketches), the bounding boxes, a change-
// detection version, and the rollback marker.
type PartComponentDefinition struct {
	bodies   *topo.SurfaceBodies
	params   *param.Parameters
	sketches *sketch.Sketches
	features *feature.PartFeatures
	keys     *identity.KeyManager
	work     *feature.WorkGeometry // origin coordinate frame + user work planes/axes/points
	units    param.UnitsOfMeasure  // document display units (length/angle/…)
	version  uint64
	eop      int // end-of-part feature index; endOfPartAtEnd ⇒ full program
	// assignments survive Recompute (keyed by persistent reference key, not body id), so a
	// material/appearance stays put when the body it is on is regenerated.
	assignments *material.AssignmentStore
}

// NewPartComponentDefinition returns an empty part content object with its feature
// engine wired to the part's parameters and key manager.
func NewPartComponentDefinition() *PartComponentDefinition {
	params := param.NewParameters()
	keys := identity.NewKeyManager()
	return &PartComponentDefinition{
		bodies:      topo.NewSurfaceBodies(),
		params:      params,
		sketches:    sketch.NewSketches(),
		features:    feature.NewPartFeatures(params, keys),
		keys:        keys,
		work:        feature.NewWorkGeometry(),
		units:       param.DefaultUnitsOfMeasure(),
		eop:         endOfPartAtEnd,
		assignments: material.NewAssignmentStore(),
	}
}

// Assignments returns the part's material/appearance assignment store. It is keyed by
// persistent reference keys, so it is unaffected by Recompute.
func (d *PartComponentDefinition) Assignments() *material.AssignmentStore { return d.assignments }

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
	d.work.Recompute()
	d.features.Recompute()
	d.bodies = topo.NewSurfaceBodies()
	for _, b := range d.features.Result() {
		d.bodies.Add(b)
	}
	d.MarkChanged()
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

// Sketches returns the part's sketches.
func (d *PartComponentDefinition) Sketches() *sketch.Sketches { return d.sketches }

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
