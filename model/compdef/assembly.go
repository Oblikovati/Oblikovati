// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/param"
)

// AssemblyComponentDefinition is an assembly's modeling content — the assembly
// analogue of [PartComponentDefinition]. Where a part owns bodies, an assembly owns
// component occurrences (the placements of parts and sub-assemblies), its structure,
// a range box unioned from those occurrences, and a change-detection version. It is
// the reference API's AssemblyComponentDefinition (M11-F01, #345); constraints,
// joints, and representations attach to it from M12.
type AssemblyComponentDefinition struct {
	occurrences *occurrence.Occurrences
	units       param.UnitsOfMeasure // document display units (length/angle/…)
}

// NewAssemblyComponentDefinition returns an empty assembly content object: no
// occurrences and default (metric) display units.
func NewAssemblyComponentDefinition() *AssemblyComponentDefinition {
	return &AssemblyComponentDefinition{
		occurrences: occurrence.NewOccurrences(),
		units:       param.DefaultUnitsOfMeasure(),
	}
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
// sub-assemblies placed in it. Placement and editing operations land in M11-F02; this
// is the live collection the assembly's structure and range box read from.
func (a *AssemblyComponentDefinition) Occurrences() *occurrence.Occurrences {
	return a.occurrences
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
