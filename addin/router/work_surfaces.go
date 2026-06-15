// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
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
func listWorkSurfaces(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	col := part.WorkSurfaces()
	out := make([]wire.WorkSurfaceInfo, col.Count())
	for i := 0; i < col.Count(); i++ {
		out[i] = workSurfaceInfo(i, col.Item(i))
	}
	return json.Marshal(wire.ListWorkSurfacesResult{Surfaces: out})
}

// getWorkSurface serves wire.MethodWorkSurfacesGet.
func getWorkSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.WorkSurfaceRefArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	w := part.WorkSurfaces().Item(in.Index)
	if w == nil {
		return nil, noSuchWorkSurface(part, in.Index)
	}
	return json.Marshal(wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)})
}

// setWorkSurfaceVisible serves wire.MethodWorkSurfacesSetVisible.
func setWorkSurfaceVisible(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetWorkSurfaceVisibleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	w := part.WorkSurfaces().Item(in.Index)
	if w == nil {
		return nil, noSuchWorkSurface(part, in.Index)
	}
	w.SetVisible(in.Visible)
	part.MarkChanged() // bump the version so the viewport re-renders the surface's new state
	return json.Marshal(wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)})
}

// renameWorkSurface serves wire.MethodWorkSurfacesRename.
func renameWorkSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.RenameWorkSurfaceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	col := part.WorkSurfaces()
	w := col.Item(in.Index)
	if w == nil {
		return nil, noSuchWorkSurface(part, in.Index)
	}
	if col.HasName(in.Name, in.Index) {
		return nil, fmt.Errorf("workSurfaces.rename: name %q is already used by another surface", in.Name)
	}
	if err := w.SetName(in.Name); err != nil {
		return nil, fmt.Errorf("workSurfaces.rename: %w", err)
	}
	part.MarkChanged()
	return json.Marshal(wire.WorkSurfaceDetailResult{Surface: workSurfaceInfo(in.Index, w)})
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
