// Hardware ray-tracing scene: BLAS-per-body + TLAS acceleration structures and a
// compute-shader ray-query Intersector (M45-F01 PBI-333, ADR-0053). Declared separately
// from head.h/HeadContext because an RTScene is not part of the live per-frame render
// loop yet (that integration is a later PBI) — it borrows a Window's device/queue and is
// otherwise self-contained, so it can be created, queried, and destroyed independently.
#pragma once
#include "head.h"

struct RTScene;

extern "C" {

// obk_rt_scene_create borrows h's Vulkan device/queue (h must have
// hwRayTracingAvailable) and returns an opaque RTScene handle, or null if RT is
// unavailable or resource creation fails.
void* obk_rt_scene_create(void* h);

// obk_rt_scene_add_mesh appends one BLAS-source mesh (vertices as tightly packed xyz
// float triples, indices as triangle-list uint32) to the scene, tagged with
// instanceCustomIndex (surfaces as renderer.Hit.InstanceID) — call once per unique body
// before obk_rt_scene_build. Returns 0 on success.
int obk_rt_scene_add_mesh(void* scene, const float* vertices, int vertexCount,
                          const uint32_t* indices, int indexCount, uint32_t instanceCustomIndex);

// obk_rt_scene_build builds every accumulated mesh's BLAS and the scene's single TLAS
// (one identity-transform instance per mesh, in add_mesh call order), then creates the
// raytrace.comp compute pipeline from the caller-supplied SPIR-V (embedded Go-side via
// go:embed, mirroring obk_viewport_init's shader-loading pattern — this file has no
// compile-time access to the .spv). Returns 0 on success. Call once after every
// obk_rt_scene_add_mesh call.
int obk_rt_scene_build(void* scene, const uint32_t* spv, int spvLen);

// obk_rt_scene_trace dispatches a single ray query against the built TLAS and reads the
// nearest-hit result back. hit/instanceID/primitiveID/t/point/normal mirror
// renderer.Hit's fields exactly (out params optional — pass null to skip).
void obk_rt_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                        float tMin, float tMax, int* hit, float* t, float* px, float* py, float* pz,
                        float* nx, float* ny, float* nz, uint32_t* instanceID, uint32_t* primitiveID);

void obk_rt_scene_destroy(void* scene);

// --- Full RT pipeline (M45-F04 PBI-345): ray-gen/closest-hit/miss over the SAME
// BLAS/TLAS obk_rt_scene_build already built — a second query path alongside the
// ray-query compute shader (PBI-333), not a replacement for it. Call after
// obk_rt_scene_build.

// obk_rt_scene_build_pipeline creates the ray-gen/miss/shadow-miss/closest-hit shader
// binding table and pipeline from the caller-supplied SPIR-V (embedded Go-side). Returns
// 0 on success.
int obk_rt_scene_build_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen, const uint32_t* missSpv,
                                int missLen, const uint32_t* shadowMissSpv, int shadowMissLen,
                                const uint32_t* chitSpv, int chitLen);

// obk_rt_scene_trace_pipeline dispatches one vkCmdTraceRaysKHR call (a single ray) and
// reads back the resulting single-bounce direct-lighting radiance. params is the 16
// floats matching pathtrace.rchit's Params UBO exactly: [lightPos.xyz, lightIntensity,
// lightColor.rgb, pad, baseColor.rgb, baseWeight, specularRoughness, specularIOR,
// baseMetalness, pad].
void obk_rt_scene_trace_pipeline(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                                 float tMin, float tMax, const float* params, float* outR, float* outG,
                                 float* outB);

// --- Live per-pixel Realistic-viewport pipeline (M45-F05 PBI-350): a per-pixel
// pinhole-camera path tracer over the SAME BLAS/TLAS, independent of the single-ray
// harness pipeline above (own SBT/descriptor set) so that pipeline's already-verified
// tests are never at risk from this one's changes.

// obk_rt_scene_build_realistic_pipeline creates the ray-gen/miss/shadow-miss/
// closest-hit pipeline for the live per-pixel viewport pass. Returns 0 on success.
int obk_rt_scene_build_realistic_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen,
                                          const uint32_t* missSpv, int missLen,
                                          const uint32_t* shadowMissSpv, int shadowMissLen,
                                          const uint32_t* chitSpv, int chitLen);

// obk_rt_scene_trace_realistic_image dispatches one vkCmdTraceRaysKHR(width,height,1)
// call. camera is 16 floats matching pathtrace_realistic.rgen's CameraParams UBO
// exactly: [eye.xyz, tMin, forward.xyz, tMax, right.xyz, tanHalfFovY, up.xyz, aspect].
// params is 56 floats (#2148) matching pathtrace_realistic.rchit's Params UBO exactly —
// see raytrace.go's RealisticLightParams.floats() for the authoritative field order
// (base lobes' original 16 floats, then Coat/Fuzz/ThinFilm/Transmission+dispersion/
// Subsurface, 8/8/4/8/12 floats respectively). outPixels must have room for
// width*height*3 floats (RGB, row-major, alpha dropped). Returns 0 on success.
int obk_rt_scene_trace_realistic_image(void* scene, int width, int height, const float* camera,
                                       const float* params, float* outPixels);

} // extern "C"
