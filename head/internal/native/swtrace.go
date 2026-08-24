//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdint.h>
#include <stdlib.h>

void* obk_sw_scene_create(void* h);
int   obk_sw_scene_build(void* scene, const void* nodes, int nodeCount, const int32_t* triOrder,
                         int triOrderCount, const void* triangles, int triangleCount,
                         const uint32_t* spv, int spvLen);
void  obk_sw_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                         float tMin, float tMax, int* hit, float* t, float* px, float* py, float* pz,
                         float* nx, float* ny, float* nz, uint32_t* instanceID, uint32_t* primitiveID);
void  obk_sw_scene_destroy(void* scene);
*/
import "C"

import (
	"errors"
	"unsafe"
)

// SWScene is the software (compute-shader BVH) ray-tracing scene: no ray-tracing
// extensions, always available (M45-F01 PBI-334, ADR-0053) — the checkbox-off /
// unsupported-hardware fallback. Mirrors RTScene's shape (Build/Trace/Destroy); build it
// from a Go-side renderer.BVH (renderer.BuildBVH) instead of adding meshes one at a time,
// since BVH construction itself is this package's own pure-Go job, not native's.
type SWScene struct {
	handle unsafe.Pointer
	built  bool
}

// NewSWScene creates a software RT scene on w. Unlike NewRTScene this cannot fail for
// capability reasons — only a torn-down/invalid window returns an error.
func (w *Window) NewSWScene() (*SWScene, error) {
	h := C.obk_sw_scene_create(w.handle)
	if h == nil {
		return nil, errors.New("native: SWScene create failed (invalid window/device)")
	}
	return &SWScene{handle: h}, nil
}

// SWBuildInput is everything Build needs, pre-marshaled to the exact byte layouts
// swtrace.comp expects: nodes/triOrder/triangles are renderer.BVHNode / int32 /
// renderer.Triangle slices, uploaded as raw bytes (Go does not reorder struct fields, so
// their in-memory layout already matches the shader's tightly-packed scalar structs —
// see swtrace.h's doc comment).
type SWBuildInput struct {
	Nodes     []byte // renderer.BVHNode slice, reinterpreted (32 bytes each)
	TriOrder  []int32
	Triangles []byte // renderer.Triangle slice, reinterpreted (44 bytes each)
}

// Build uploads in's BVH and creates the traversal compute pipeline. Call once before
// Trace.
func (s *SWScene) Build(in SWBuildInput) error {
	rc := C.obk_sw_scene_build(s.handle,
		unsafe.Pointer(unsafe.SliceData(in.Nodes)), C.int(len(in.Nodes)/32),
		(*C.int32_t)(unsafe.Pointer(unsafe.SliceData(in.TriOrder))), C.int(len(in.TriOrder)),
		unsafe.Pointer(unsafe.SliceData(in.Triangles)), C.int(len(in.Triangles)/44),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(swtraceCompSPV))), C.int(len(swtraceCompSPV)))
	if rc != 0 {
		return errors.New("native: SWScene.Build failed (see stderr)")
	}
	s.built = true
	return nil
}

// Trace dispatches a single ray query against the built BVH and returns the nearest
// hit, or a zero-value RTHit (Hit=false) for a miss — same shape as RTScene.Trace.
func (s *SWScene) Trace(origin, direction [3]float32, tMin, tMax float32) RTHit {
	var cHit C.int
	var cT, cPX, cPY, cPZ, cNX, cNY, cNZ C.float
	var cInstanceID, cPrimitiveID C.uint32_t
	C.obk_sw_scene_trace(s.handle,
		C.float(origin[0]), C.float(origin[1]), C.float(origin[2]),
		C.float(direction[0]), C.float(direction[1]), C.float(direction[2]),
		C.float(tMin), C.float(tMax),
		&cHit, &cT, &cPX, &cPY, &cPZ, &cNX, &cNY, &cNZ, &cInstanceID, &cPrimitiveID)
	if cHit == 0 {
		return RTHit{}
	}
	return RTHit{
		Hit: true, T: float32(cT),
		Point:       [3]float32{float32(cPX), float32(cPY), float32(cPZ)},
		Normal:      [3]float32{float32(cNX), float32(cNY), float32(cNZ)},
		InstanceID:  uint32(cInstanceID),
		PrimitiveID: uint32(cPrimitiveID),
	}
}

// Destroy frees every Vulkan resource the scene owns.
func (s *SWScene) Destroy() {
	C.obk_sw_scene_destroy(s.handle)
	s.handle = nil
}
