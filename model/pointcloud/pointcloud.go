// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import "oblikovati.org/math"

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
// render after the display budget (MaximumPointCount) is applied.
func (pc *PointCloud) TotalPointCount() int { return len(pc.points) }
func (pc *PointCloud) DisplayedPointCount() int {
	if pc.maxPoints <= 0 || pc.maxPoints >= len(pc.points) {
		return len(pc.points)
	}
	return pc.maxPoints
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

// DisplayedPoints returns the budgeted point set in MODEL space — what the renderer draws. When
// a display budget is set it strides evenly across the scan so the sample stays spatially
// representative rather than a truncated prefix.
func (pc *PointCloud) DisplayedPoints() []math.Point3 {
	out := make([]math.Point3, 0, pc.DisplayedPointCount())
	for _, p := range pc.sampledCloudPoints() {
		out = append(out, pc.ToModelSpace(p))
	}
	return out
}

// sampledCloudPoints returns the cloud-local points after the display budget, strided evenly.
func (pc *PointCloud) sampledCloudPoints() []math.Point3 {
	if pc.maxPoints <= 0 || pc.maxPoints >= len(pc.points) {
		return pc.points
	}
	stride := len(pc.points) / pc.maxPoints
	out := make([]math.Point3, 0, pc.maxPoints)
	for i := 0; i < len(pc.points) && len(out) < pc.maxPoints; i += stride {
		out = append(out, pc.points[i])
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
