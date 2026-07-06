// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/binary"
	"hash/fnv"
	"io"
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
	low, high := s.PointCloudIntensityRamp()
	var verts []float32
	count := 0
	for i := 0; i < clouds.Count(); i++ {
		pc := clouds.Item(i)
		samples := s.cloudRenderSamples(pc)
		if len(samples) == 0 {
			continue
		}
		verts = appendCloudVertices(verts, pc, samples, low, high)
		count += len(samples)
	}
	return verts, count
}

// cloudRenderSamples returns the samples a cloud contributes to the GPU buffer: its displayed set,
// capped for VRAM (unbudgeted clouds) then thinned to the session render density. Nil for a hidden or
// empty cloud, so the caller skips it.
func (s *Session) cloudRenderSamples(pc *pointcloud.PointCloud) []pointcloud.PointSample {
	if !pc.Visible() {
		return nil
	}
	samples := capForRender(pc, pc.DisplayedSamples())
	return densityFilteredSamples(pc, samples, s.PointCloudRenderDensity())
}

// appendCloudVertices appends one cloud's interleaved [pos.xyz, rgba] vertices (model space) to dst,
// colouring each sample by the cloud's display mode and the session intensity ramp.
func appendCloudVertices(dst []float32, pc *pointcloud.PointCloud, samples []pointcloud.PointSample, low, high [4]float32) []float32 {
	if dst == nil {
		dst = make([]float32, 0, len(samples)*pointCloudGPUStride)
	}
	for _, sm := range samples {
		c := displaySampleColor(pc, sm, low, high)
		dst = append(dst, float32(sm.Point.X), float32(sm.Point.Y), float32(sm.Point.Z),
			c[0], c[1], c[2], c[3])
	}
	return dst
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
	put := floatWriter(h)
	put(float64(s.PointCloudRenderDensity()))
	low, high := s.PointCloudIntensityRamp()
	writeFloat32s(put, low[:])
	writeFloat32s(put, high[:])
	part, err := activePart(s)
	if err != nil {
		return h.Sum64()
	}
	clouds := part.PointClouds()
	for i := 0; i < clouds.Count(); i++ {
		pc := clouds.Item(i)
		if pc.Visible() {
			hashCloudDisplay(h, pc)
		}
	}
	return h.Sum64()
}

// floatWriter returns a closure that folds a float64 into h as little-endian IEEE-754 bytes — the
// shared, run-stable encoding the display-key hashes use.
func floatWriter(h io.Writer) func(float64) {
	var buf [8]byte
	return func(f float64) {
		binary.LittleEndian.PutUint64(buf[:], stdmath.Float64bits(f))
		_, _ = h.Write(buf[:])
	}
}

// writeFloat32s folds each float32 into the key via put (widened to float64).
func writeFloat32s(put func(float64), cs []float32) {
	for _, c := range cs {
		put(float64(c))
	}
}

// hashCloudDisplay folds one visible cloud's display-determining inputs into h. resourceID keeps two
// distinct scans from colliding; the rest mirror pointcloud.displaySignature (placement/scale/budget/
// crops) plus the color inputs (display mode, intensity range) the GPU vertices bake in.
func hashCloudDisplay(h io.Writer, pc *pointcloud.PointCloud) {
	put := floatWriter(h)
	putStr := func(str string) { _, _ = h.Write([]byte(str)); _, _ = h.Write([]byte{0}) }

	putStr(pc.ResourceID())
	putStr(string(pc.DisplayMode()))
	for _, c := range pc.Transform().Cells() {
		put(float64(c))
	}
	put(pc.Scale())
	put(float64(pc.MaximumPointCount()))
	if lo, hi, ok := pc.IntensityRange(); ok {
		put(lo)
		put(hi)
	}
	hashActiveCropBoxes(put, pc.Crops())
}

// hashActiveCropBoxes folds each active crop's model-space box corners into the key via put, so a
// change to which region is cropped invalidates the retained GPU buffer.
func hashActiveCropBoxes(put func(float64), crops *pointcloud.PointCloudCrops) {
	for i := 0; i < crops.Count(); i++ {
		c := crops.Item(i)
		if !c.Active() {
			continue
		}
		b := c.Box()
		put(float64(b.Min.X))
		put(float64(b.Min.Y))
		put(float64(b.Min.Z))
		put(float64(b.Max.X))
		put(float64(b.Max.Y))
		put(float64(b.Max.Z))
	}
}
