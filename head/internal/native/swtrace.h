// Software ray-tracing scene: a plain compute-shader traversal of a CPU-built BVH
// (renderer.BuildBVH), with no ray-tracing extensions at all (M45-F01 PBI-334,
// ADR-0053) — the always-available fallback obk_head_ray_tracing_support/
// SupportsHardwareRayTracing say to fall back to when the hardware checkbox is off or
// unsupported. Mirrors raytrace.h's RTScene shape (create/build/trace/destroy) but needs
// none of its acceleration-structure machinery: BVH nodes and triangles are plain bound
// storage buffers, no buffer-device-address or extension-function loading required.
#pragma once
#include "head.h"

struct SWScene;

extern "C" {

// obk_sw_scene_create borrows h's Vulkan device/queue and returns an opaque SWScene
// handle. Always succeeds if h has a valid device — unlike hardware RT, this backend
// has no capability precondition.
void* obk_sw_scene_create(void* h);

// obk_sw_scene_build uploads a CPU-built BVH (nodes, tightly packed as
// renderer.BVHNode: 6 floats + 2 int32 = 32 bytes each) + its triangle reorder index
// (int32 each) + the original-order triangle list (tightly packed as renderer.Triangle:
// 9 floats + 2 uint32 = 44 bytes each) and creates the traversal compute pipeline from
// the caller-supplied SPIR-V (embedded Go-side, mirroring obk_rt_scene_build). Returns 0
// on success.
int obk_sw_scene_build(void* scene, const void* nodes, int nodeCount, const int32_t* triOrder,
                       int triOrderCount, const void* triangles, int triangleCount, const uint32_t* spv,
                       int spvLen);

// obk_sw_scene_trace dispatches a single ray query against the built BVH and reads the
// nearest-hit result back — identical signature/semantics to obk_rt_scene_trace.
void obk_sw_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                        float tMin, float tMax, int* hit, float* t, float* px, float* py, float* pz,
                        float* nx, float* ny, float* nz, uint32_t* instanceID, uint32_t* primitiveID);

void obk_sw_scene_destroy(void* scene);

} // extern "C"
