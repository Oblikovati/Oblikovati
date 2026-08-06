// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// This file holds the YAML codecs for the placed/boolean solid features: holes,
// bosses (both placed on a face by reference key, like the dress-up family) and
// combine (a boolean between two running bodies by index). Keys re-bind to the
// regenerated topology after recompute; body indices address the running state.

// HoleData is a hole's recipe: the placement face (reference key), diameter/depth,
// the hole geometry type, and optional tap data.
type HoleData struct {
	Face            string  `yaml:"face"`
	Diameter        float64 `yaml:"diameter"`
	Depth           float64 `yaml:"depth"`
	ThroughAll      bool    `yaml:"throughAll,omitempty"`
	CounterDiameter float64 `yaml:"counterDiameter,omitempty"`
	CounterDepth    float64 `yaml:"counterDepth,omitempty"`
	CounterAngle    float64 `yaml:"counterAngle,omitempty"`
	PointAngle      float64 `yaml:"pointAngle,omitempty"`
	Type            string  `yaml:"type"`
	Tapped          bool    `yaml:"tapped,omitempty"`
	Designation     string  `yaml:"designation,omitempty"`
	// GeomFace selects the placement face by a GEOMETRIC descriptor (ADR-0040) when the
	// hole is externally authored (NX exporter) and Face (the lineage key) is empty.
	GeomFace *GeomFaceRefData `yaml:"geomFace,omitempty"`
	// Center is the explicit drill point [x,y,z] in model space (externally-authored holes).
	// Empty means drill at the placement face centroid.
	Center []float64 `yaml:"center,omitempty,flow"`
	// Tapered / Class / LeftHanded complete the tap alongside Designation (#1862): the tap is an
	// axis of its own, orthogonal to Type (the seat), so a tapped counterbore round-trips as both.
	Tapered    bool   `yaml:"taperTapped,omitempty"`
	Class      string `yaml:"threadClass,omitempty"`
	LeftHanded bool   `yaml:"leftHanded,omitempty"`
	// Clearance sizes the bore from a fastener table rather than Diameter (#1862). Persisting the
	// FASTENER rather than the resolved diameter is the whole point — a reopened part must still
	// resize when the fastener changes.
	ClearanceStandard string `yaml:"clearanceStandard,omitempty"`
	ClearanceFastener string `yaml:"clearanceFastener,omitempty"`
	ClearanceFit      string `yaml:"clearanceFit,omitempty"`
	// Termination is where the bore stops (#1863): empty ⇒ the plain Depth, else "to-face" or
	// "from-to" bottoming on the named work planes.
	Termination string `yaml:"termination,omitempty"`
	ToPlane     string `yaml:"toPlane,omitempty"`
	FromPlane   string `yaml:"fromPlane,omitempty"`
	// Placement is the rule locating the bores (#1861); absent ⇒ the single bore Face/Center name.
	Placement *HolePlacementData `yaml:"placement,omitempty"`
}

// BossData is a boss's recipe: a raised cylinder on a placement face.
type BossData struct {
	Face        string           `yaml:"face"`
	Diameter    float64          `yaml:"diameter"`
	Height      float64          `yaml:"height"`
	FaceAnchors []FaceAnchorData `yaml:"faceAnchors,omitempty"`
}

// CombineData booleans two running bodies (by index) under an operation.
type CombineData struct {
	Target    int    `yaml:"target"`
	Tool      int    `yaml:"tool"`
	Operation string `yaml:"operation"`
}

func serializeHole(def *HoleDefinition, sk SketchIndexer) (*HoleData, error) {
	kind, err := holeTypeName(def.Type)
	if err != nil {
		return nil, err
	}
	placement, err := serializeHolePlacement(def.Placement, sk)
	if err != nil {
		return nil, err
	}
	return &HoleData{
		Placement:       placement,
		Face:            encodeKey(def.PlacementFaceKey),
		Diameter:        evalFloat(def.Diameter),
		Depth:           evalFloat(def.Depth),
		ThroughAll:      def.ThroughAll,
		CounterDiameter: evalFloat(def.CounterDiameter),
		CounterDepth:    evalFloat(def.CounterDepth),
		CounterAngle:    evalFloat(def.CounterAngle),
		PointAngle:      evalFloat(def.PointAngle),
		Type:            kind,
		Tapped:          def.Tap.Tapped,
		Designation:     def.Tap.Designation,
		GeomFace:        encodeGeomFacePtr(def.GeomFace),
		Center:          encodePoint3Ptr(def.Center),
		Tapered:         def.Tap.Tapered,
		Class:           def.Tap.Class,
		LeftHanded:      def.Tap.LeftHanded,

		ClearanceStandard: def.Clearance.Standard,
		ClearanceFastener: def.Clearance.Fastener,
		ClearanceFit:      def.Clearance.Fit,

		Termination: revolveExtentName(def.Termination),
		ToPlane:     planeRefOf(def.ToPlane),
		FromPlane:   planeRefOf(def.FromPlane),
	}, nil
}

func restoreHole(fs *PartFeatures, h *HoleData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if h == nil {
		return nil, fmt.Errorf("hole feature is missing its payload")
	}
	key, err := decodeKey(h.Face)
	if err != nil {
		return nil, err
	}
	seat, err := parseHoleType(h.Type)
	if err != nil {
		return nil, err
	}
	pf := restoreHoleSeat(NewHoleFeatures(fs), h, key, seat)
	def := pf.feature.(*HoleFeature).def
	def.Type = seat
	def.ThroughAll = h.ThroughAll
	def.PointAngle = constFloat(h.PointAngle)
	def.GeomFace = decodeGeomFacePtr(h.GeomFace)
	def.Center = decodePoint3Ptr(h.Center)
	def.Tap = HoleTapInfo{Tapped: h.Tapped, Designation: h.Designation,
		Tapered: h.Tapered, Class: h.Class, LeftHanded: h.LeftHanded}
	def.Clearance = HoleClearanceInfo{Standard: h.ClearanceStandard, Fastener: h.ClearanceFastener, Fit: h.ClearanceFit}
	if def.Placement, err = restoreHolePlacement(h.Placement, def.faceRef(), sk, work); err != nil {
		return nil, err
	}
	return pf, restoreHoleTermination(def, h, work)
}

// restoreHoleSeat rebuilds the hole by its SEAT alone. The tap is applied by the caller, on top of
// whatever seat was restored — before #1862 a tapped hole was rebuilt through AddTapped, which is a
// DRILLED constructor, so a tapped counterbore came back without its recess.
func restoreHoleSeat(holes *HoleFeatures, h *HoleData, key []byte, seat HoleType) *PartFeature {
	switch {
	case seat == CounterboreHole || seat == SpotFaceHole:
		return holes.AddCounterbore(key, constFloat(h.Diameter), constFloat(h.Depth), constFloat(h.CounterDiameter), constFloat(h.CounterDepth))
	case seat == CountersinkHole:
		return holes.AddCountersink(key, constFloat(h.Diameter), constFloat(h.Depth), constFloat(h.CounterDiameter), constFloat(h.CounterAngle))
	case h.ThroughAll:
		return holes.AddDrilledThrough(key, constFloat(h.Diameter))
	default:
		return holes.AddDrilled(key, constFloat(h.Diameter), constFloat(h.Depth))
	}
}

// restoreHoleTermination puts a persisted geometric termination back on a restored hole (#1863).
func restoreHoleTermination(def *HoleDefinition, h *HoleData, work *WorkGeometry) error {
	if h.Termination == "" {
		return nil
	}
	def.Termination = parseExtentName(h.Termination)
	var err error
	if def.ToPlane, err = restorePlaneRef(work, h.ToPlane); err != nil {
		return err
	}
	def.FromPlane, err = restorePlaneRef(work, h.FromPlane)
	return err
}

func restoreBoss(fs *PartFeatures, b *BossData) (*PartFeature, error) {
	if b == nil {
		return nil, fmt.Errorf("boss feature is missing its payload")
	}
	key, err := decodeKey(b.Face)
	if err != nil {
		return nil, err
	}
	anchors, err := decodeFaceAnchors(b.FaceAnchors)
	if err != nil {
		return nil, err
	}
	return NewBossFeatures(fs).addBoss(&BossDefinition{
		PlacementFaceKey: key, Diameter: constFloat(b.Diameter), Height: constFloat(b.Height), FaceAnchors: anchors,
	}), nil
}

func restoreCombine(fs *PartFeatures, c *CombineData) (*PartFeature, error) {
	if c == nil {
		return nil, fmt.Errorf("combine feature is missing its payload")
	}
	op, err := parseOperation(c.Operation)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddCombine(c.Target, c.Tool, op), nil
}

// holeTypeName / parseHoleType map the hole geometry type to/from a stable name.
func holeTypeName(t HoleType) (string, error) {
	switch t {
	case DrilledHole:
		return "drilled", nil
	case CounterboreHole:
		return "counterbore", nil
	case CountersinkHole:
		return "countersink", nil
	case SpotFaceHole:
		return "spotface", nil
	default:
		return "", fmt.Errorf("unknown hole type %d", t)
	}
}

func parseHoleType(name string) (HoleType, error) {
	switch name {
	case "drilled":
		return DrilledHole, nil
	case "counterbore":
		return CounterboreHole, nil
	case "countersink":
		return CountersinkHole, nil
	case "spotface":
		return SpotFaceHole, nil
	default:
		return 0, fmt.Errorf("unknown hole type %q (want drilled|counterbore|countersink|spotface)", name)
	}
}
