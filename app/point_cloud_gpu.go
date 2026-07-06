// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/binary"
	"hash/fnv"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/pointcloud"
)

// Retained GPU point-cloud path (M17-F06, #645 perf). The original display drew every scan point as
// a CPU-built 3-axis cross (6 line vertices) on the Lines pipeline, rebuilt and re-uploaded every
// frame — and because that marker batch rode the overlay atlas, a loaded scan also defeated the
// whole-model geometry-upload cache (#1422), collapsing frame rate for the entire scene. Instead the
// head now uploads each visible cloud's points ONCE to a native GL-points buffer (renderer point
// pipeline) and redraws them from VRAM as the camera orbits — the CloudCompare-style static buffer.
//
// This file assembles that buffer (interleaved x,y,z,r,g,b,a per point, model space) and a
// camera-independent content key. The head only rebuilds+uploads when the key changes, so a static
// scan costs nothing per frame. Point color reuses displaySampleColor (RGB / intensity / default),
// so the modes the display-mode work adds flow through unchanged.

// pointCloudGPUStride is the float count per GPU point vertex: pos.xyz + rgba. It must match the
// point pipeline's vertex input (kPointFloats in head/internal/native/viewport.cpp).
const pointCloudGPUStride = 7

// pointCloudRenderCap bounds how many points a single UNBUDGETED cloud pushes to the GPU — a VRAM /
// upload safety net for a pathologically large scan (e.g. a 50M-point aerial LiDAR tile). At
// pointCloudGPUStride*4 = 28 bytes/point this caps one cloud's buffer near 84 MB. A user-set
// MaximumPointCount always wins (the model already strided to it); this only applies when the cloud
// has no explicit budget (#645). It replaces the "budget on import" idea from the render-perf plan:
// the native points path draws millions fine, so the cap is a resource guard, not a frame-rate fix.
const pointCloudRenderCap = 3_000_000

// PointCloudGPUVertices returns the interleaved vertex data (x,y,z,r,g,b,a per point, model space)
// for the active part's visible attached clouds, drawn as native GL points, plus the point count.
// It is camera-independent: the head uploads it once and redraws from VRAM while orbiting (#645).
// Empty when the active document is not a part or has no visible, non-empty clouds.
//
// Example: verts, n := s.PointCloudGPUVertices() // n points, len(verts) == n*7
func (s *Session) PointCloudGPUVertices() ([]float32, int) {
	part, err := activePart(s)
	if err != nil {
		return nil, 0
	}
	clouds := part.PointClouds()
	var verts []float32
	count := 0
	low, high := s.PointCloudIntensityRamp()
	for i := 0; i < clouds.Count(); i++ {
		pc := clouds.Item(i)
		if !pc.Visible() {
			continue
		}
		samples := capForRender(pc, pc.DisplayedSamples())
		samples = densityFilteredSamples(pc, samples, s.PointCloudRenderDensity())
		if len(samples) == 0 {
			continue
		}
		if verts == nil {
			verts = make([]float32, 0, len(samples)*pointCloudGPUStride)
		}
		for _, sm := range samples {
			c := displaySampleColor(pc, sm, low, high)
			verts = append(verts, float32(sm.Point.X), float32(sm.Point.Y), float32(sm.Point.Z),
				c[0], c[1], c[2], c[3])
			count++
		}
	}
	return verts, count
}

// capForRender strides an unbudgeted cloud's samples down to pointCloudRenderCap; a cloud with an
// explicit MaximumPointCount is returned unchanged (the model already applied the user's budget).
// The result is deterministic given the cloud's data + budget + crops, so it does not perturb the
// display key (which those inputs already cover).
func capForRender(pc *pointcloud.PointCloud, samples []pointcloud.PointSample) []pointcloud.PointSample {
	if pc.MaximumPointCount() != 0 {
		return samples
	}
	return strideForCap(samples, pointCloudRenderCap)
}

// densityFilteredSamples returns a stable pseudo-random percentage of samples for viewport render
// density. It never mutates the cached point-cloud slice and keeps 100% as a zero-allocation fast
// path. The hash is deterministic, so orbiting the camera does not make the cloud sparkle.
func densityFilteredSamples(pc *pointcloud.PointCloud, samples []pointcloud.PointSample, density float32) []pointcloud.PointSample {
	if density >= 100 {
		return samples
	}
	if density <= 0 || len(samples) == 0 {
		return nil
	}
	threshold := uint64((float64(density) / 100) * float64(^uint64(0)))
	seed := pointCloudDensitySeed(pc)
	out := samples[:0:0]
	for _, sample := range samples {
		if densitySampleHash(seed, sample.Point) <= threshold {
			out = append(out, sample)
		}
	}
	return out
}

// pointCloudDensitySeed derives the density filter's per-cloud seed from ResourceID alone.
// Name is deliberately excluded: the display key (hashCloudDisplay) hashes only ResourceID, so
// seeding on Name would let a rename change which samples the filter keeps without invalidating
// the retained GPU buffer — the buffer would then disagree with the live filter (#645). ResourceID
// already distinguishes two attached scans.
func pointCloudDensitySeed(pc *pointcloud.PointCloud) uint64 {
	h := uint64(1469598103934665603)
	for _, b := range []byte(pc.ResourceID()) {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return h
}

func densitySampleHash(seed uint64, p math.Point3) uint64 {
	h := mixDensityHash(seed ^ stdmath.Float64bits(float64(p.X)))
	h = mixDensityHash(h ^ stdmath.Float64bits(float64(p.Y)))
	h = mixDensityHash(h ^ stdmath.Float64bits(float64(p.Z)))
	return h
}

func mixDensityHash(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}

// strideForCap returns samples capped to max entries taken at an even stride (spatially
// representative, not a truncated prefix), or the input unchanged when it already fits or max <= 0.
func strideForCap(samples []pointcloud.PointSample, max int) []pointcloud.PointSample {
	if max <= 0 || len(samples) <= max {
		return samples
	}
	stride := len(samples) / max
	out := make([]pointcloud.PointSample, 0, max)
	for i := 0; i < len(samples) && len(out) < max; i += stride {
		out = append(out, samples[i])
	}
	return out
}

// PointCloudDisplayKey returns a hash that changes only when the displayed point set or its colors
// change — the cloud placement, scale, display budget, display mode, intensity range, active crops
// and visibility, per cloud. It excludes the camera, so the head compares it frame to frame to skip
// the vertex rebuild + GPU upload while the scan is merely orbited (#645). A key of 0 is reserved by
// the renderer to mean "always upload", so an empty scene returns the FNV offset basis (non-zero).
func (s *Session) PointCloudDisplayKey() uint64 {
	h := fnv.New64a()
	var buf [8]byte
	putF := func(f float64) { binary.LittleEndian.PutUint64(buf[:], stdmath.Float64bits(f)); _, _ = h.Write(buf[:]) }
	putF(float64(s.PointCloudRenderDensity()))
	low, high := s.PointCloudIntensityRamp()
	for _, c := range low {
		putF(float64(c))
	}
	for _, c := range high {
		putF(float64(c))
	}
	part, err := activePart(s)
	if err != nil {
		return h.Sum64()
	}
	clouds := part.PointClouds()
	for i := 0; i < clouds.Count(); i++ {
		pc := clouds.Item(i)
		if !pc.Visible() {
			continue
		}
		hashCloudDisplay(h, pc)
	}
	return h.Sum64()
}

// hashCloudDisplay folds one visible cloud's display-determining inputs into h. resourceID keeps two
// distinct scans from colliding; the rest mirror pointcloud.displaySignature (placement/scale/budget/
// crops) plus the color inputs (display mode, intensity range) the GPU vertices bake in.
func hashCloudDisplay(h interface{ Write([]byte) (int, error) }, pc *pointcloud.PointCloud) {
	var buf [8]byte
	putF := func(f float64) { binary.LittleEndian.PutUint64(buf[:], stdmath.Float64bits(f)); _, _ = h.Write(buf[:]) }
	putStr := func(str string) { _, _ = h.Write([]byte(str)); _, _ = h.Write([]byte{0}) }

	putStr(pc.ResourceID())
	putStr(string(pc.DisplayMode()))
	for _, c := range pc.Transform().Cells() {
		putF(float64(c))
	}
	putF(pc.Scale())
	putF(float64(pc.MaximumPointCount()))
	if lo, hi, ok := pc.IntensityRange(); ok {
		putF(lo)
		putF(hi)
	}
	crops := pc.Crops()
	for i := 0; i < crops.Count(); i++ {
		c := crops.Item(i)
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
