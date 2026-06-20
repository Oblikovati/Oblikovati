// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"encoding/binary"
	"hash/fnv"
	stdmath "math"

	"oblikovati.org/math"
)

// PointCloud is one attached scan: its points in cloud-local coordinates, a placement transform
// and uniform scale into model space, a display point budget, and visibility (M17-F06, #645). It
// is a referenced object, not B-rep geometry — a design is modeled against it. The scan bytes
// live in the document resource table (ADR-0031); resourceID addresses them so the cloud
// re-derives its points on open without the original file.
type PointCloud struct {
	name       string
	visible    bool
	transform  math.Matrix4  // cloud-space → model-space placement
	scale      float64       // uniform cloud→model scale (UnitsFactor folded in)
	points     []math.Point3 // cloud-local coordinates, as decoded
	source     string        // SourceFullFileName (display / re-link metadata)
	resourceID string        // ADR-0031 resource UUID addressing the scan bytes
	maxPoints  int           // display budget; 0 = show every point
	crops      PointCloudCrops

	// Display cache (#645 perf): DisplayedPoints transforms every scan point to model space each
	// call, which the head invokes every frame — ~2 ms for a 266k-point scan. The result depends
	// only on the placement, scale, budget and crops, not the camera, so it is cached and returned
	// unchanged while you orbit; displaySig is the signature of those inputs it was built for.
	displayCache []math.Point3
	displaySig   uint64
	displayValid bool
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
// identity-placed, with no display cap. source/resourceID record where the scan came from and
// where its bytes are embedded.
//
// Example: pc := pointcloud.New("scan", "/scans/room.xyz", "uuid", pts)
func New(name, source, resourceID string, points []math.Point3) *PointCloud {
	return &PointCloud{
		name:       name,
		visible:    true,
		transform:  math.Identity4(),
		scale:      1,
		points:     points,
		source:     source,
		resourceID: resourceID,
	}
}

// Name/SetName get and set the cloud's unique browser name.
func (pc *PointCloud) Name() string     { return pc.name }
func (pc *PointCloud) SetName(n string) { pc.name = n }

// Visible/SetVisible get and set whether the cloud renders in the viewport.
func (pc *PointCloud) Visible() bool     { return pc.visible }
func (pc *PointCloud) SetVisible(v bool) { pc.visible = v }

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
func (pc *PointCloud) TotalPointCount() int { return len(pc.points) }
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
		return len(pc.points)
	}
	n := 0
	for _, p := range pc.points {
		if pc.crops.Admits(pc.ToModelSpace(p)) {
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
func (pc *PointCloud) CloudPoints() []math.Point3 { return pc.points }

// DisplayedPoints returns the rendered point set in MODEL space — the points passing the active
// crops, then strided evenly to the display budget so the sample stays spatially representative
// rather than a truncated prefix. The result is cached and reused while the placement, scale,
// budget and crops are unchanged (the head rebuilds the overlay every frame, but the cloud is
// static as the camera orbits), so a static 266k-point scan costs O(1) per frame after the first.
func (pc *PointCloud) DisplayedPoints() []math.Point3 {
	sig := pc.displaySignature()
	if pc.displayValid && sig == pc.displaySig {
		return pc.displayCache
	}
	pc.displayCache = pc.buildDisplayed()
	pc.displaySig, pc.displayValid = sig, true
	return pc.displayCache
}

// buildDisplayed transforms the displayed set. When the cloud is budgeted and uncropped it strides
// the cloud-local points to the budget FIRST and transforms only those, so a 50k-budget rebuild
// touches 50k points, not all 266k. With an active crop the crop test is in model space, so every
// point must be transformed.
func (pc *PointCloud) buildDisplayed() []math.Point3 {
	if pc.maxPoints > 0 && pc.maxPoints < len(pc.points) && !pc.crops.anyActive() {
		sampled := strideSample(pc.points, pc.maxPoints)
		out := make([]math.Point3, len(sampled))
		for i, p := range sampled {
			out[i] = pc.ToModelSpace(p)
		}
		return out
	}
	return strideSample(pc.croppedModelPoints(), pc.maxPoints)
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
func (pc *PointCloud) CroppedModelPoints() []math.Point3 { return pc.croppedModelPoints() }

// NearestModelPoint returns the cloud's scan point in MODEL space closest to query (snapping a
// model coordinate onto the as-built data), searching the full cloud — placement-transformed but
// not crop-limited — so a snap finds a point even where the display is cropped away. Found is false
// only for an empty cloud.
func (pc *PointCloud) NearestModelPoint(query math.Point3) (math.Point3, bool) {
	if len(pc.points) == 0 {
		return math.Point3{}, false
	}
	best := pc.ToModelSpace(pc.points[0])
	bestD := query.DistanceSquaredTo(best)
	for _, p := range pc.points[1:] {
		m := pc.ToModelSpace(p)
		if d := query.DistanceSquaredTo(m); d < bestD {
			best, bestD = m, d
		}
	}
	return best, true
}

// croppedModelPoints returns every point in MODEL space that passes the active crops.
func (pc *PointCloud) croppedModelPoints() []math.Point3 {
	out := make([]math.Point3, 0, len(pc.points))
	for _, p := range pc.points {
		m := pc.ToModelSpace(p)
		if pc.crops.Admits(m) {
			out = append(out, m)
		}
	}
	return out
}

// strideSample returns pts capped to max entries, taken at an even stride (the whole slice when
// max is 0 or already within budget).
func strideSample(pts []math.Point3, max int) []math.Point3 {
	if max <= 0 || max >= len(pts) {
		return pts
	}
	stride := len(pts) / max
	out := make([]math.Point3, 0, max)
	for i := 0; i < len(pts) && len(out) < max; i += stride {
		out = append(out, pts[i])
	}
	return out
}

// RangeBox returns the cloud's axis-aligned bounds in MODEL space; CloudRangeBox returns them in
// cloud-local space. Both are empty for a cloud with no points.
func (pc *PointCloud) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, p := range pc.points {
		box = box.ExtendPoint(pc.ToModelSpace(p))
	}
	return box
}

func (pc *PointCloud) CloudRangeBox() math.Box {
	box := math.EmptyBox()
	for _, p := range pc.points {
		box = box.ExtendPoint(p)
	}
	return box
}

// scalePoint scales a point's coordinates about the origin.
func scalePoint(p math.Point3, s float64) math.Point3 {
	f := math.Scalar(s)
	return math.P3(p.X*f, p.Y*f, p.Z*f)
}
