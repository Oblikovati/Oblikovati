// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// ExtrudeData is an extrude's recipe: which sketch profile, the boolean operation, and
// the distance extent. The distance is the evaluated growth (a fixed value on reopen;
// parametric distance expressions arrive with the dimension-driven extent API).
type ExtrudeData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile"`
	Operation string  `yaml:"operation"`
	Distance  float64 `yaml:"distance"`
	Taper     float64 `yaml:"taper,omitempty"`
}

func serializeExtrude(def *ExtrudeDefinition, sk SketchIndexer) (*ExtrudeData, error) {
	if def.Extent.Type != DistanceExtent {
		return nil, fmt.Errorf("only distance extents are serializable (got extent type %d)", def.Extent.Type)
	}
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &ExtrudeData{
		Sketch:    idx,
		Profile:   def.ProfileIndex,
		Operation: op,
		Distance:  def.Extent.distance(),
		Taper:     def.Taper,
	}, nil
}

// requireExtrude restores an extrude, erroring on a missing payload.
func requireExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer) (*PartFeature, error) {
	if ed == nil {
		return nil, fmt.Errorf("extrude feature is missing its payload")
	}
	return restoreExtrude(fs, ed, sk)
}

func restoreExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer) (*PartFeature, error) {
	skt, ok := sk.At(ed.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references sketch index %d, which does not exist", ed.Sketch)
	}
	op, err := parseOperation(ed.Operation)
	if err != nil {
		return nil, err
	}
	dist := ed.Distance
	pf := NewExtrudeFeatures(fs).AddByDistanceExtent(skt, ed.Profile, op, func() float64 { return dist })
	if ed.Taper != 0 {
		pf.feature.(*ExtrudeFeature).def.Taper = ed.Taper
	}
	return pf, nil
}
