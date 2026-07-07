// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// ExtrudeData is an extrude's recipe: which sketch region(s), the boolean operation, and
// the full extent (type, direction, distances, taper, and any work-plane targets). The
// distances are evaluated values (a fixed value on reopen; parametric expressions arrive
// with the dimension-driven extent API).
type ExtrudeData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile,omitempty"`  // legacy single-region key; read for back-compat
	Profiles  []int   `yaml:"profiles,omitempty"` // one or more regions (current form)
	Operation string  `yaml:"operation"`
	Extent    string  `yaml:"extent,omitempty"`    // distance|through-all|to-next|to-face|from-to|distance-from-face
	Direction string  `yaml:"direction,omitempty"` // positive|negative|symmetric
	Distance  float64 `yaml:"distance,omitempty"`
	Distance2 float64 `yaml:"distance2,omitempty"` // asymmetric second-direction distance
	Taper     float64 `yaml:"taper,omitempty"`
	ToPlane   string  `yaml:"toPlane,omitempty"`   // WorkRef of the to-face / from-to end / distance-from-face target
	FromPlane string  `yaml:"fromPlane,omitempty"` // WorkRef of the from-to start
	// ProfilePoints selects the extruded region(s) by an interior seed point (sketch 2-D cm),
	// one per region, instead of by index. An external author (e.g. the Inventor exporter) cannot
	// predict the reader's DCEL region ordering, so it names regions by containment. When present
	// it wins over Profiles/Profile; absent, the index-based selection is used unchanged.
	ProfilePoints [][]float64 `yaml:"profilePoints,omitempty"`
}

// extrudeProfiles returns the region indices a payload selects, accepting both the
// current `profiles` list and an older file's scalar `profile` key.
func (ed *ExtrudeData) extrudeProfiles() []int {
	if len(ed.Profiles) > 0 {
		return ed.Profiles
	}
	return []int{ed.Profile}
}

var extentNames = map[ExtentType]string{
	DistanceExtent:         "distance",
	ToNextExtent:           "to-next",
	ThroughAllExtent:       "through-all",
	ToFaceExtent:           "to-face",
	FromToExtent:           "from-to",
	DistanceFromFaceExtent: "distance-from-face",
}

var directionNames = map[ExtentDirection]string{
	PositiveDir:  "positive",
	NegativeDir:  "negative",
	SymmetricDir: "symmetric",
}

func serializeExtrude(def *ExtrudeDefinition, sk SketchIndexer) (*ExtrudeData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	ename, ok := extentNames[def.Extent.Type]
	if !ok {
		return nil, fmt.Errorf("extrude: unknown extent type %d", def.Extent.Type)
	}
	d := &ExtrudeData{
		Sketch: idx, Profiles: append([]int(nil), def.ProfileIndices...), Operation: op,
		Extent: ename, Direction: directionNames[def.Extent.Direction],
		Distance: def.Extent.distance(), Distance2: def.Extent.distance2(), Taper: def.Taper,
	}
	if def.Extent.ToPlane != nil {
		d.ToPlane = string(def.Extent.ToPlane.Key())
	}
	if def.Extent.FromPlane != nil {
		d.FromPlane = string(def.Extent.FromPlane.Key())
	}
	return d, nil
}

// requireExtrude restores an extrude, erroring on a missing payload.
func requireExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if ed == nil {
		return nil, fmt.Errorf("extrude feature is missing its payload")
	}
	return restoreExtrude(fs, ed, sk, work)
}

func restoreExtrude(fs *PartFeatures, ed *ExtrudeData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	skt, ok := sk.At(ed.Sketch)
	if !ok {
		return nil, fmt.Errorf("extrude references sketch index %d, which does not exist", ed.Sketch)
	}
	op, err := parseOperation(ed.Operation)
	if err != nil {
		return nil, err
	}
	extent, err := restoreExtent(ed, work)
	if err != nil {
		return nil, err
	}
	profiles := resolveSeeds(skt, ed.ProfilePoints, ed.extrudeProfiles())
	return NewExtrudeFeatures(fs).AddExtrude(skt, profiles, op, extent, ed.Taper), nil
}

// resolveSeeds maps each interior seed point to the smallest closed region that contains it,
// falling back to the explicit index list when no seeds are given or none resolve. Region
// ordering is a DCEL-walk artifact (regions.go), so an author that knows only the region
// geometry must select by containment, not index. It never returns empty (an empty selection
// would extrude the whole body).
func resolveSeeds(skt *sketch.Sketch, seeds [][]float64, fallback []int) []int {
	if len(seeds) == 0 {
		return fallback
	}
	profs := skt.Profiles()
	var idx []int
	for _, s := range seeds {
		if len(s) != 2 {
			continue
		}
		q := math.P2(s[0], s[1])
		best, bestArea := -1, stdmath.Inf(1)
		for i := 0; i < profs.Count(); i++ {
			p := profs.Item(i)
			if p.IsClosed() && p.Contains(q) && p.Area() < bestArea {
				best, bestArea = i, p.Area()
			}
		}
		if best >= 0 {
			idx = append(idx, best)
		}
	}
	if len(idx) == 0 {
		return fallback
	}
	return idx
}

// restoreExtent rebuilds the extent from its recipe, resolving any work-plane targets.
func restoreExtent(ed *ExtrudeData, work *WorkGeometry) (Extent, error) {
	ext := Extent{Type: parseExtentName(ed.Extent), Direction: parseDirectionName(ed.Direction)}
	dist := ed.Distance
	ext.Distance = func() float64 { return dist }
	if ed.Distance2 != 0 {
		d2 := ed.Distance2
		ext.Distance2 = func() float64 { return d2 }
	}
	if ed.ToPlane != "" {
		wp, err := resolvePlaneRef(work, ed.ToPlane)
		if err != nil {
			return Extent{}, err
		}
		ext.ToPlane = wp
	}
	if ed.FromPlane != "" {
		wp, err := resolvePlaneRef(work, ed.FromPlane)
		if err != nil {
			return Extent{}, err
		}
		ext.FromPlane = wp
	}
	return ext, nil
}

// resolvePlaneRef resolves a work-plane reference string against the part's work geometry.
func resolvePlaneRef(work *WorkGeometry, ref string) (*WorkPlane, error) {
	if work == nil {
		return nil, fmt.Errorf("extrude: plane reference %q needs the part's work geometry", ref)
	}
	return work.WorkPlaneByRef(WorkRef(ref))
}

// parseExtentName maps a recipe extent name to its type (empty/unknown ⇒ distance, the
// back-compatible default for files written before extent modes).
func parseExtentName(name string) ExtentType {
	for t, n := range extentNames {
		if n == name {
			return t
		}
	}
	return DistanceExtent
}

// parseDirectionName maps a recipe direction name to its value (empty/unknown ⇒ positive).
func parseDirectionName(name string) ExtentDirection {
	for d, n := range directionNames {
		if n == name {
			return d
		}
	}
	return PositiveDir
}
