// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"encoding/binary"
	"hash/fnv"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// PointSample is one decoded scan point with optional display channels.
type PointSample struct {
	Point        math.Point3
	RGB          [3]float32
	HasRGB       bool
	Intensity    float64
	HasIntensity bool
}

// PointCloud is one attached scan: its points in cloud-local coordinates, a placement transform
// and uniform scale into model space, a display point budget, and visibility (M17-F06, #645). It
// is a referenced object, not B-rep geometry — a design is modeled against it. The scan bytes
// live in the document resource table (ADR-0031); resourceID addresses them so the cloud
// re-derives its points on open without the original file.
type PointCloud struct {
	name              string
	visible           bool
	displayMode       types.PointCloudDisplayMode
	transform         math.Matrix4 // cloud-space → model-space placement
	scale             float64      // uniform cloud→model scale (UnitsFactor folded in)
	samples           []PointSample
	source            string // SourceFullFileName (display / re-link metadata)
	resourceID        string // ADR-0031 resource UUID addressing the scan bytes
	maxPoints         int    // display budget; 0 = show every point
	crops             PointCloudCrops
	intensityMin      float64
	intensityMax      float64
	hasIntensityRange bool

	// Display cache (#645 perf): DisplayedSamples transforms every scan point to model space each
	// call, which the head invokes every frame — ~2 ms for a 266k-point scan. The result depends
	// only on the placement, scale, budget and crops, not the camera, so it is cached and returned
	// unchanged while you orbit; displaySig is the signature of those inputs it was built for.
	displayCache      []PointSample
	displaySig        uint64
	displayValid      bool
	displayPointCache []math.Point3
	displayPointSig   uint64
	displayPointValid bool
}

// Crops returns the cloud's crop-volume collection — the model-space boxes that limit display
// (#645). AddCrop is the factory: it mints a unique name and starts the crop active.
func (pc *PointCloud) Crops() *PointCloudCrops { return &pc.crops }

// AddCrop adds an active crop over the given model-space box under a freshly minted name.
func (pc *PointCloud) AddCrop(box math.Box) *PointCloudCrop {
	c, _ := pc.crops.Add(pc.crops.uniqueName("Crop"), box)
	return c
}

// New creates a cloud from decoded cloud-local points. It starts visible, unscaled (factor 1),
// identity-placed, with no display cap and RGB display mode. source/resourceID
// record where the scan came from and where its bytes are embedded.
//
// Example: pc := pointcloud.New("scan", "/scans/room.xyz", "uuid", pts)
func New(name, source, resourceID string, points []math.Point3) *PointCloud {
	samples := make([]PointSample, len(points))
	for i, p := range points {
		samples[i] = PointSample{Point: p}
	}
	return newFromSamples(name, source, resourceID, samples)
}

// NewWithSamples creates a cloud from decoded scan samples, preserving any RGB or intensity
// channels the reader found.
func NewWithSamples(name, source, resourceID string, samples []PointSample) *PointCloud {
	return newFromSamples(name, source, resourceID, samples)
}

func newFromSamples(name, source, resourceID string, samples []PointSample) *PointCloud {
	pc := &PointCloud{
		name:        name,
		visible:     true,
		displayMode: types.PointCloudDisplayModeRGB,
		transform:   math.Identity4(),
		scale:       1,
		samples:     samples,
		source:      source,
		resourceID:  resourceID,
	}
	pc.updateIntensityRange()
	return pc
}

// Name/SetName get and set the cloud's unique browser name.
func (pc *PointCloud) Name() string     { return pc.name }
func (pc *PointCloud) SetName(n string) { pc.name = n }

// Visible/SetVisible get and set whether the cloud renders in the viewport.
func (pc *PointCloud) Visible() bool     { return pc.visible }
func (pc *PointCloud) SetVisible(v bool) { pc.visible = v }

// DisplayMode/SetDisplayMode get and set how the cloud renders in the viewport.
func (pc *PointCloud) DisplayMode() types.PointCloudDisplayMode { return pc.displayMode }
func (pc *PointCloud) SetDisplayMode(m types.PointCloudDisplayMode) bool {
	if !m.IsValid() {
		return false
	}
	pc.displayMode = m
	return true
}

// Transform/SetTransform get and set the cloud→model placement matrix.
func (pc *PointCloud) Transform() math.Matrix4     { return pc.transform }
func (pc *PointCloud) SetTransform(m math.Matrix4) { pc.transform = m }

// Scale/SetScale get and set the uniform cloud→model scale; a non-positive value is rejected.
func (pc *PointCloud) Scale() float64 { return pc.scale }
func (pc *PointCloud) SetScale(s float64) bool {
	if s <= 0 {
		return false
	}
	pc.scale = s
	return true
}

// SourceFullFileName returns the scan's original path; ResourceID returns its embedded-bytes id.
func (pc *PointCloud) SourceFullFileName() string { return pc.source }
func (pc *PointCloud) ResourceID() string         { return pc.resourceID }

// TotalPointCount returns how many points the scan holds; DisplayedPointCount returns how many
// render after the active crops and the display budget (MaximumPointCount) are applied.
func (pc *PointCloud) TotalPointCount() int { return len(pc.samples) }
func (pc *PointCloud) DisplayedPointCount() int {
	n := pc.croppedCount()
	if pc.maxPoints > 0 && pc.maxPoints < n {
		return pc.maxPoints
	}
	return n
}

// croppedCount returns how many points pass the active crops (all of them when none is active).
func (pc *PointCloud) croppedCount() int {
	if !pc.crops.anyActive() {
		return len(pc.samples)
	}
	n := 0
	for _, s := range pc.samples {
		if pc.crops.Admits(pc.ToModelSpace(s.Point)) {
			n++
		}
	}
	return n
}

// MaximumPointCount/SetMaximumPointCount get and set the display budget (0 = unbounded). A
// negative budget clamps to 0.
func (pc *PointCloud) MaximumPointCount() int { return pc.maxPoints }
func (pc *PointCloud) SetMaximumPointCount(n int) {
	if n < 0 {
		n = 0
	}
	pc.maxPoints = n
}

// ToModelSpace maps a cloud-local point to model space (scale then place); FromModelSpace is its
// inverse, returning false when the placement is non-invertible (a degenerate transform).
func (pc *PointCloud) ToModelSpace(p math.Point3) math.Point3 {
	return pc.transform.TransformPoint(scalePoint(p, pc.scale))
}

func (pc *PointCloud) FromModelSpace(p math.Point3) (math.Point3, bool) {
	inv, ok := pc.transform.Inverse()
	if !ok || pc.scale == 0 {
		return math.Point3{}, false
	}
	return scalePoint(inv.TransformPoint(p), 1/pc.scale), true
}

// CloudPoints returns the scan points in cloud-local space (the decoded coordinates).
func (pc *PointCloud) CloudPoints() []math.Point3 {
	out := make([]math.Point3, len(pc.samples))
	for i, s := range pc.samples {
		out[i] = s.Point
	}
	return out
}

// IntensityRange returns the cloud's decoded intensity range, if any.
func (pc *PointCloud) IntensityRange() (min, max float64, ok bool) {
	return pc.intensityMin, pc.intensityMax, pc.hasIntensityRange
}

// DisplayedSamples returns the rendered sample set in MODEL space — the points passing the active
// crops, then strided evenly to the display budget so the sample stays spatially representative
// rather than a truncated prefix. The result is cached and reused while the placement, scale,
// budget and crops are unchanged (the head rebuilds the overlay every frame, but the cloud is
// static as the camera orbits), so a static 266k-point scan costs O(1) per frame after the first.
func (pc *PointCloud) DisplayedSamples() []PointSample {
	sig := pc.displaySignature()
	if pc.displayValid && sig == pc.displaySig {
		return pc.displayCache
	}
	pc.displayCache = pc.buildDisplayed()
	pc.displaySig, pc.displayValid = sig, true
	return pc.displayCache
}

// DisplayedPoints returns the rendered point set in MODEL space.
func (pc *PointCloud) DisplayedPoints() []math.Point3 {
	sig := pc.displaySignature()
	if pc.displayPointValid && sig == pc.displayPointSig {
		return pc.displayPointCache
	}
	samples := pc.DisplayedSamples()
	pc.displayPointCache = make([]math.Point3, len(samples))
	for i, s := range samples {
		pc.displayPointCache[i] = s.Point
	}
	pc.displayPointSig, pc.displayPointValid = sig, true
	return pc.displayPointCache
}

// buildDisplayed transforms the displayed set. When the cloud is budgeted and uncropped it strides
// the cloud-local points to the budget FIRST and transforms only those, so a 50k-budget rebuild
// touches 50k points, not all 266k. With an active crop the crop test is in model space, so every
// point must be transformed.
func (pc *PointCloud) buildDisplayed() []PointSample {
	if pc.maxPoints > 0 && pc.maxPoints < len(pc.samples) && !pc.crops.anyActive() {
		sampled := strideSampleSamples(pc.samples, pc.maxPoints)
		out := make([]PointSample, len(sampled))
		for i, s := range sampled {
			out[i] = s
			out[i].Point = pc.ToModelSpace(s.Point)
		}
		return out
	}
	return strideSampleSamples(pc.croppedModelSamples(), pc.maxPoints)
}

// displaySignature hashes the inputs that determine the displayed set — the placement, scale,
// budget, and each active crop — so DisplayedPoints can detect when its cache is stale without
// threading invalidation through every setter.
func (pc *PointCloud) displaySignature() uint64 {
	h := fnv.New64a()
	var buf [8]byte
	putF := func(f float64) { binary.LittleEndian.PutUint64(buf[:], stdmath.Float64bits(f)); _, _ = h.Write(buf[:]) }
	for _, c := range pc.transform.Cells() {
		putF(float64(c))
	}
	putF(pc.scale)
	putF(float64(pc.maxPoints))
	pc.hashActiveCrops(putF)
	return h.Sum64()
}

// hashActiveCrops feeds each active crop's box into the display signature.
func (pc *PointCloud) hashActiveCrops(putF func(float64)) {
	for i := 0; i < pc.crops.Count(); i++ {
		c := pc.crops.Item(i)
		if !c.Active() {
			continue
		}
		b := c.Box()
		putF(float64(b.Min.X))
		putF(float64(b.Min.Y))
		putF(float64(b.Min.Z))
		putF(float64(b.Max.X))
		putF(float64(b.Max.Y))
		putF(float64(b.Max.Z))
	}
}

// CroppedModelPoints returns every point in MODEL space that passes the active crops, at full
// density (unlike DisplayedPoints it is not strided to the display budget). It is the input a
// best-fit consumer reads — fitting a work plane to the cloud's current working region (#645).
func (pc *PointCloud) CroppedModelPoints() []math.Point3 {
	samples := pc.croppedModelSamples()
	out := make([]math.Point3, len(samples))
	for i, s := range samples {
		out[i] = s.Point
	}
	return out
}

// NearestModelPoint returns the cloud's scan point in MODEL space closest to query (snapping a
// model coordinate onto the as-built data), searching the full cloud — placement-transformed but
// not crop-limited — so a snap finds a point even where the display is cropped away. Found is false
// only for an empty cloud.
func (pc *PointCloud) NearestModelPoint(query math.Point3) (math.Point3, bool) {
	if len(pc.samples) == 0 {
		return math.Point3{}, false
	}
	best := pc.ToModelSpace(pc.samples[0].Point)
	bestD := query.DistanceSquaredTo(best)
	for _, s := range pc.samples[1:] {
		m := pc.ToModelSpace(s.Point)
		if d := query.DistanceSquaredTo(m); d < bestD {
			best, bestD = m, d
		}
	}
	return best, true
}

// croppedModelSamples returns every sample in MODEL space that passes the active crops.
func (pc *PointCloud) croppedModelSamples() []PointSample {
	out := make([]PointSample, 0, len(pc.samples))
	for _, s := range pc.samples {
		m := pc.ToModelSpace(s.Point)
		if pc.crops.Admits(m) {
			s.Point = m
			out = append(out, s)
		}
	}
	return out
}

// strideSampleSamples returns samples capped to max entries, taken at an even stride (the whole
// slice when max is 0 or already within budget).
func strideSampleSamples(pts []PointSample, max int) []PointSample {
	if max <= 0 || max >= len(pts) {
		return pts
	}
	stride := len(pts) / max
	out := make([]PointSample, 0, max)
	for i := 0; i < len(pts) && len(out) < max; i += stride {
		out = append(out, pts[i])
	}
	return out
}

// RangeBox returns the cloud's axis-aligned bounds in MODEL space; CloudRangeBox returns them in
// cloud-local space. Both are empty for a cloud with no points.
func (pc *PointCloud) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, s := range pc.samples {
		box = box.ExtendPoint(pc.ToModelSpace(s.Point))
	}
	return box
}

func (pc *PointCloud) CloudRangeBox() math.Box {
	box := math.EmptyBox()
	for _, s := range pc.samples {
		box = box.ExtendPoint(s.Point)
	}
	return box
}

// updateIntensityRange captures the decoded intensity min/max once at construction time.
func (pc *PointCloud) updateIntensityRange() {
	first := true
	for _, s := range pc.samples {
		if !s.HasIntensity {
			continue
		}
		if first {
			pc.intensityMin, pc.intensityMax = s.Intensity, s.Intensity
			pc.hasIntensityRange = true
			first = false
			continue
		}
		if s.Intensity < pc.intensityMin {
			pc.intensityMin = s.Intensity
		}
		if s.Intensity > pc.intensityMax {
			pc.intensityMax = s.Intensity
		}
	}
}

// scalePoint scales a point's coordinates about the origin.
func scalePoint(p math.Point3, s float64) math.Point3 {
	f := math.Scalar(s)
	return math.P3(p.X*f, p.Y*f, p.Z*f)
}
