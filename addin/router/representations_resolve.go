// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
)

// Encoders from the engine's representation read interfaces into the wire info rows (M12-F04).

// designViewInfo encodes a design-view representation.
func designViewInfo(d assembly.DesignViewRep) wire.DesignViewInfo {
	return wire.DesignViewInfo{
		RepresentationInfo: wire.RepresentationInfo{ID: d.ID(), Name: d.Name(), Kind: d.Kind().String(), Active: d.Active()},
		HiddenCount:        d.HiddenCount(),
		AppearanceCount:    d.AppearanceCount(),
		SectionPlanes:      d.SectionPlanes(),
		Camera:             cameraViewFromCaptured(d.Camera()),
	}
}

// positionalInfo encodes a positional representation.
func positionalInfo(p assembly.PositionalRep) wire.PositionalInfo {
	return wire.PositionalInfo{
		RepresentationInfo: wire.RepresentationInfo{ID: p.ID(), Name: p.Name(), Kind: p.Kind().String(), Active: p.Active()},
		OverrideCount:      p.OverrideCount(),
	}
}

// lodInfo encodes a level-of-detail representation.
func lodInfo(l assembly.LODRep) wire.LODInfo {
	return wire.LODInfo{
		RepresentationInfo: wire.RepresentationInfo{ID: l.ID(), Name: l.Name(), Kind: l.Kind().String(), Active: l.Active()},
		SuppressedCount:    l.SuppressedCount(),
	}
}

// modelStateInfo encodes a model state.
func modelStateInfo(m assembly.ModelStateRep) wire.ModelStateInfo {
	return wire.ModelStateInfo{
		ID: m.ID(), Name: m.Name(), Active: m.Active(),
		DesignView: m.DesignViewName(), Positional: m.PositionalName(), LevelOfDetail: m.LevelOfDetailName(),
	}
}

// cameraViewFromCaptured maps a captured camera to the wire view (nil when none was captured).
func cameraViewFromCaptured(c *assembly.CapturedCamera) *wire.CameraView {
	if c == nil {
		return nil
	}
	return &wire.CameraView{
		Eye:    types.Point{X: c.Eye.X, Y: c.Eye.Y, Z: c.Eye.Z},
		Target: types.Point{X: c.Target.X, Y: c.Target.Y, Z: c.Target.Z},
		Up:     types.Vector{X: c.Up.X, Y: c.Up.Y, Z: c.Up.Z},
		FOV:    c.FOV,
	}
}

// designViewResult / positionalResult / lodResult marshal the representation with the given id
// from the active assembly (after a set* mutation).
func designViewResult(asm *compdef.AssemblyComponentDefinition, id uint64) (json.RawMessage, error) {
	return json.Marshal(wire.DesignViewResult{Representation: designViewInfo(asm.Representations().DesignViewByID(id))})
}

func positionalResult(asm *compdef.AssemblyComponentDefinition, id uint64) (json.RawMessage, error) {
	return json.Marshal(wire.PositionalResult{Representation: positionalInfo(asm.Representations().PositionalByID(id))})
}

func lodResult(asm *compdef.AssemblyComponentDefinition, id uint64) (json.RawMessage, error) {
	return json.Marshal(wire.LODResult{Representation: lodInfo(asm.Representations().LODByID(id))})
}
