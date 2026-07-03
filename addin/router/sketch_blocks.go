// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// Sketch blocks over the wire (M06-F07, #622): definitions are part-level
// (the component definition's SketchBlocks registry); instances are sketch
// entities placed with insertion point / rotation / uniform scale.

// createBlockDefinition serves wire.MethodSketchBlockDefinitionCreate.
func createBlockDefinition(_ *app.Session, part *compdef.PartComponentDefinition, in wire.CreateBlockDefinitionArgs) (wire.SketchBlockDefinitionInfo, error) {
	reg := part.SketchBlocks()
	def, err := definitionFromArgs(part, reg, in)
	if err != nil {
		return wire.SketchBlockDefinitionInfo{}, err
	}
	return blockDefinitionInfo(indexOfDefinition(reg, def), def), nil
}

// definitionFromArgs creates an empty definition, or — given a source
// selection — moves it into a new definition (create-from-selection).
func definitionFromArgs(part *compdef.PartComponentDefinition, reg *sketch.BlockDefinitions, in wire.CreateBlockDefinitionArgs) (*sketch.BlockDefinition, error) {
	if len(in.EntityRefs) == 0 {
		return reg.Define(in.Name)
	}
	sk, err := sketchAtIndex(part, in.SourceSketchIndex)
	if err != nil {
		return nil, err
	}
	ents := make([]sketch.Entity, 0, len(in.EntityRefs))
	for _, id := range in.EntityRefs {
		e, ok := sk.EntityByID(sketch.ID(id))
		if !ok {
			return nil, fmt.Errorf("entity %d not found in sketch %d", id, in.SourceSketchIndex)
		}
		ents = append(ents, e)
	}
	def, _, err := sk.Blocks().CreateFromSelection(reg, in.Name, ents)
	return def, err
}

// listBlockDefinitions serves wire.MethodSketchBlockDefinitionList.
func listBlockDefinitions(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ListBlockDefinitionsResult, error) {
	return wire.ListBlockDefinitionsResult{Definitions: projectAll(part.SketchBlocks(), blockDefinitionInfo)}, nil
}

// deleteBlockDefinition serves wire.MethodSketchBlockDefinitionDelete.
func deleteBlockDefinition(_ *app.Session, part *compdef.PartComponentDefinition, in wire.DeleteBlockDefinitionArgs) (wire.OKResult, error) {
	if err := part.SketchBlocks().Delete(in.Name); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// addBlockInstance serves wire.MethodSketchAddBlockInstance.
func addBlockInstance(_ *app.Session, part *compdef.PartComponentDefinition, in wire.AddSketchBlockArgs) (wire.AddSketchBlockResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.AddSketchBlockResult{}, err
	}
	def, ok := part.SketchBlocks().ByName(in.Definition)
	if !ok {
		return wire.AddSketchBlockResult{}, fmt.Errorf("block definition %q does not exist", in.Definition)
	}
	if len(in.Position) != 2 {
		return wire.AddSketchBlockResult{}, fmt.Errorf("a block placement needs position [x, y], got %d components", len(in.Position))
	}
	rotation, err := angleArg(part, "rotationAngle", in.RotationAngle)
	if err != nil {
		return wire.AddSketchBlockResult{}, err
	}
	pos := math.P2(math.Scalar(in.Position[0]), math.Scalar(in.Position[1]))
	inst := sk.Blocks().Insert(def, sketch.PlacementTransform(pos, rotation, in.Scale))
	return wire.AddSketchBlockResult{EntityID: uint64(inst.EntityID())}, nil
}

// listBlockInstances serves wire.MethodSketchListBlockInstances.
func listBlockInstances(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.ListBlockInstancesResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListBlockInstancesResult{}, err
	}
	blocks := sk.Blocks()
	out := make([]wire.SketchBlockInfo, blocks.InstanceCount())
	for i := range out {
		out[i] = blockInstanceInfo(i, blocks.Item(i))
	}
	return wire.ListBlockInstancesResult{Instances: out}, nil
}

// blockDefinitionInfo renders a definition as its wire summary.
func blockDefinitionInfo(index int, def *sketch.BlockDefinition) wire.SketchBlockDefinitionInfo {
	return wire.SketchBlockDefinitionInfo{
		Index: index, Name: def.Name(),
		EntityCount: def.EntityCount(), InstanceCount: def.InstanceCount(),
	}
}

// blockInstanceInfo renders one placed instance, decomposing its transform
// back into insertion point / rotation / uniform scale.
func blockInstanceInfo(index int, inst *sketch.BlockInstance) wire.SketchBlockInfo {
	cells := inst.Transform().Cells()
	pos := inst.Transform().Translation()
	rotation := stdmath.Atan2(float64(cells[3]), float64(cells[0]))
	scale := stdmath.Hypot(float64(cells[0]), float64(cells[3]))
	return wire.SketchBlockInfo{
		Index: index, ID: uint64(inst.EntityID()), Definition: inst.DefinitionName(),
		Position: []float64{float64(pos.X), float64(pos.Y)},
		Rotation: rotation, Scale: scale,
		EntityCount: inst.EntityCount(),
	}
}

// indexOfDefinition finds a definition's registry index.
func indexOfDefinition(reg *sketch.BlockDefinitions, def *sketch.BlockDefinition) int {
	for i := 0; i < reg.Count(); i++ {
		if reg.Item(i) == def {
			return i
		}
	}
	return -1
}
