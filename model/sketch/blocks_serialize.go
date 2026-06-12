// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Block persistence (M06-F07, #622). Definitions are part-level: they
// serialize as their own recipe section (the component definition owns it),
// each carrying its self-contained points and entities in definition
// coordinates. Instances are sketch entities (kind "blockInstance") that
// re-bind to their definition by name on restore — so definitions must be
// applied before sketches.

// BlockDefinitionData is one persisted block definition. Points/Entities
// reuse the sketch row shapes; nested instances appear as blockInstance rows.
type BlockDefinitionData struct {
	Name     string       `yaml:"name"`
	Points   []PointData  `yaml:"points,omitempty"`
	Entities []EntityData `yaml:"entities,omitempty"`
}

// MarshalBlockDefinitions renders the registry for the part recipe.
func (sc *Sketches) MarshalBlockDefinitions() ([]BlockDefinitionData, error) {
	out := make([]BlockDefinitionData, 0, sc.blockDefs.Count())
	for _, def := range sc.blockDefs.defs {
		bd, err := serializeBlockDefinition(def)
		if err != nil {
			return nil, fmt.Errorf("block %q: %w", def.name, err)
		}
		out = append(out, bd)
	}
	return out, nil
}

// serializeBlockDefinition renders one definition's points and entities. A
// *Point entity (a moved standalone sketch point) persists as a standalone
// point row, the same convention SketchData uses.
func serializeBlockDefinition(def *BlockDefinition) (BlockDefinitionData, error) {
	bd := BlockDefinitionData{Name: def.name}
	standalone := map[ID]bool{}
	for _, e := range def.ents {
		if p, ok := e.(*Point); ok {
			standalone[p.id] = true
		}
	}
	for _, p := range def.pts {
		bd.Points = append(bd.Points, PointData{
			ID: int(p.id), X: float64(p.X), Y: float64(p.Y), Standalone: standalone[p.id],
		})
	}
	for _, e := range def.ents {
		if _, ok := e.(*Point); ok {
			continue // captured as a standalone point row above
		}
		ed, err := serializeEntity(e)
		if err != nil {
			return BlockDefinitionData{}, err
		}
		bd.Entities = append(bd.Entities, ed)
	}
	return bd, nil
}

// ApplyBlockDefinitions rebuilds the registry from the part recipe. Each
// definition's rows are restored through a scratch sketch — reusing every
// entity codec — and the rebuilt geometry is moved into the definition.
func (sc *Sketches) ApplyBlockDefinitions(rows []BlockDefinitionData) error {
	for _, bd := range rows {
		def, err := sc.blockDefs.Define(bd.Name)
		if err != nil {
			return err
		}
		if err := rebuildBlockDefinition(sc.blockDefs, def, bd); err != nil {
			return fmt.Errorf("block %q: %w", bd.Name, err)
		}
	}
	return nil
}

// rebuildBlockDefinition restores one definition's geometry via a scratch
// sketch sharing the registry (so nested blockInstance rows re-bind).
func rebuildBlockDefinition(reg *BlockDefinitions, def *BlockDefinition, bd BlockDefinitionData) error {
	scratch := NewSketches()
	scratch.blockDefs = reg
	if err := scratch.ApplyRecipe([]SketchData{{Plane: serializePlane(XYPlane()), Points: bd.Points, Entities: bd.Entities}}); err != nil {
		return err
	}
	s := scratch.Item(0)
	def.pts = append([]*Point(nil), s.pts...)
	for _, e := range s.Entities() {
		if inst, ok := e.(*BlockInstance); ok {
			inst.owner = nil // nested: owned by the definition, not a sketch
		}
		if err := def.Add(e); err != nil {
			return err
		}
	}
	return nil
}

// matrixCells flattens a placement transform for persistence (row-major).
func matrixCells(m math.Matrix3) []float64 {
	cells := m.Cells()
	out := make([]float64, len(cells))
	for i, c := range cells {
		out[i] = float64(c)
	}
	return out
}

// matrixFromCells rebuilds a persisted placement transform.
func matrixFromCells(cells []float64) (math.Matrix3, error) {
	if len(cells) != 9 {
		return math.Matrix3{}, fmt.Errorf("a block transform needs 9 cells, got %d", len(cells))
	}
	var scalars [9]math.Scalar
	for i, c := range cells {
		scalars[i] = math.Scalar(c)
	}
	return math.Matrix3FromCells(scalars), nil
}
