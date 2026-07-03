// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The WorkSurface wire surface (M20-F16): enumerate the part's construction surfaces (the
// result's sheet bodies, wrapped as named, visibility-controlled objects), get one, and
// set its visibility or name. The collection is produced by surface features, so there is
// no create method (it mirrors how datum surfaces appear in the reference API).

// listWorkSurfaces serves wire.MethodWorkSurfacesList.
func listWorkSurfaces(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ListWorkSurfacesResult, error) {
	return wire.ListWorkSurfacesResult{Surfaces: projectAll(part.WorkSurfaces(), workSurfaceInfo)}, nil
}

// getWorkSurface serves wire.MethodWorkSurfacesGet.
func getWorkSurface(_ *app.Session, part *compdef.PartComponentDefinition, in wire.WorkSurfaceRefArgs) (wire.WorkSurfaceDetailResult, error) {
	w := part.WorkSurfaces().Item(in.Index)
	if w == nil {
		return wire.WorkSurfaceDetailResult{}, noSuchWorkSurface(part, in.Index)
	}
	return wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)}, nil
}

// setWorkSurfaceVisible serves wire.MethodWorkSurfacesSetVisible.
func setWorkSurfaceVisible(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetWorkSurfaceVisibleArgs) (wire.WorkSurfaceDetailResult, error) {
	w := part.WorkSurfaces().Item(in.Index)
	if w == nil {
		return wire.WorkSurfaceDetailResult{}, noSuchWorkSurface(part, in.Index)
	}
	w.SetVisible(in.Visible)
	part.MarkChanged() // bump the version so the viewport re-renders the surface's new state
	return wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)}, nil
}

// renameWorkSurface serves wire.MethodWorkSurfacesRename.
func renameWorkSurface(_ *app.Session, part *compdef.PartComponentDefinition, in wire.RenameWorkSurfaceArgs) (wire.WorkSurfaceDetailResult, error) {
	col := part.WorkSurfaces()
	w := col.Item(in.Index)
	if w == nil {
		return wire.WorkSurfaceDetailResult{}, noSuchWorkSurface(part, in.Index)
	}
	if col.HasName(in.Name, in.Index) {
		return wire.WorkSurfaceDetailResult{}, fmt.Errorf("workSurfaces.rename: name %q is already used by another surface", in.Name)
	}
	if err := w.SetName(in.Name); err != nil {
		return wire.WorkSurfaceDetailResult{}, fmt.Errorf("workSurfaces.rename: %w", err)
	}
	part.MarkChanged()
	return wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)}, nil
}

// noSuchWorkSurface is the shared "index out of range" error, naming the bound.
func noSuchWorkSurface(part *compdef.PartComponentDefinition, index int) error {
	return fmt.Errorf("workSurfaces: no surface at index %d (have %d)", index, part.WorkSurfaces().Count())
}

// workSurfaceInfo projects one surface to its wire row.
func workSurfaceInfo(index int, w *feature.WorkSurface) wire.WorkSurfaceInfo {
	bodies := 0
	if w.Body() != nil {
		bodies = 1
	}
	return wire.WorkSurfaceInfo{
		Index:       index,
		Name:        w.Name(),
		Ref:         fmt.Sprintf("surface/%d", index),
		Visible:     w.Visible(),
		Translucent: w.Translucent(),
		Bodies:      bodies,
		Source:      w.Source(),
	}
}
