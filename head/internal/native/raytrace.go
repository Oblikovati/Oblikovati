//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdint.h>
#include <stdlib.h>

void* obk_rt_scene_create(void* h);
int   obk_rt_scene_add_mesh(void* scene, const float* vertices, int vertexCount,
                            const uint32_t* indices, int indexCount, uint32_t instanceCustomIndex);
int   obk_rt_scene_build(void* scene, const uint32_t* spv, int spvLen);
void  obk_rt_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                         float tMin, float tMax, int* hit, float* t, float* px, float* py, float* pz,
                         float* nx, float* ny, float* nz, uint32_t* instanceID, uint32_t* primitiveID);
void  obk_rt_scene_destroy(void* scene);

int  obk_rt_scene_build_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen, const uint32_t* missSpv,
                                 int missLen, const uint32_t* shadowMissSpv, int shadowMissLen,
                                 const uint32_t* chitSpv, int chitLen);
void obk_rt_scene_trace_pipeline(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                                 float tMin, float tMax, const float* params, float* outR, float* outG,
                                 float* outB);

int obk_rt_scene_build_realistic_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen,
                                          const uint32_t* missSpv, int missLen,
                                          const uint32_t* shadowMissSpv, int shadowMissLen,
                                          const uint32_t* chitSpv, int chitLen);
int obk_rt_scene_trace_realistic_image(void* scene, int width, int height, const float* camera,
                                       const float* params, float* outPixels);
*/
import "C"

import (
	"errors"
	"unsafe"
)

// RTScene is a hardware ray-tracing scene: one BLAS per added mesh plus a TLAS
// (identity-transform instances, in add-order), queryable one ray at a time via the
// ray-query compute shader (M45-F01 PBI-333, ADR-0053). It borrows w's device and does
// not touch the live per-frame render loop — see raytrace.h's package doc.
type RTScene struct {
	handle unsafe.Pointer
	built  bool
}

// NewRTScene creates an RT scene on w, or an error if hardware ray tracing is
// unavailable on this device (RayTracingExtensionSupport reports why).
func (w *Window) NewRTScene() (*RTScene, error) {
	h := C.obk_rt_scene_create(w.handle)
	if h == nil {
		return nil, errors.New("native: hardware ray tracing unavailable on this device")
	}
	return &RTScene{handle: h}, nil
}

// AddMesh appends one BLAS-source mesh: vertices as tightly packed xyz float triples,
// indices as a triangle list, tagged with instanceCustomIndex (surfaces as
// renderer.Hit.InstanceID). Call before Build; primitiveID in a resulting Hit is the
// triangle's 0-based position within THIS mesh's index list, so callers must order
// indices to match how they'll interpret [renderer.Triangle.PrimitiveID].
func (s *RTScene) AddMesh(vertices []float32, indices []uint32, instanceCustomIndex uint32) error {
	if s.built {
		return errors.New("native: RTScene already built, cannot add more meshes")
	}
	vertexCount := len(vertices) / 3
	rc := C.obk_rt_scene_add_mesh(s.handle,
		(*C.float)(unsafe.Pointer(unsafe.SliceData(vertices))), C.int(vertexCount),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(indices))), C.int(len(indices)),
		C.uint32_t(instanceCustomIndex))
	if rc != 0 {
		return errors.New("native: RTScene.AddMesh failed (see stderr)")
	}
	return nil
}

// Build finalizes every added mesh's BLAS, the scene's TLAS, and the ray-query compute
// pipeline. Call once after every AddMesh call, before Trace.
func (s *RTScene) Build() error {
	rc := C.obk_rt_scene_build(s.handle,
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(raytraceCompSPV))), C.int(len(raytraceCompSPV)))
	if rc != 0 {
		return errors.New("native: RTScene.Build failed (see stderr)")
	}
	s.built = true
	return nil
}

// RTHit mirrors renderer.Hit's fields — the native ray-query result for one Trace call.
type RTHit struct {
	Hit                     bool
	T                       float32
	Point, Normal           [3]float32
	InstanceID, PrimitiveID uint32
}

// Trace dispatches a single ray query against the built TLAS and returns the nearest
// hit, or a zero-value RTHit (Hit=false) for a miss.
func (s *RTScene) Trace(origin, direction [3]float32, tMin, tMax float32) RTHit {
	var cHit C.int
	var cT, cPX, cPY, cPZ, cNX, cNY, cNZ C.float
	var cInstanceID, cPrimitiveID C.uint32_t
	C.obk_rt_scene_trace(s.handle,
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

// PipelineParams is pathtrace.rchit's Params UBO, laid out exactly (M45-F04 PBI-345):
// 16 float32 in this field order, matching the shader's std140 struct byte-for-byte
// (see raytrace.h's doc comment) — Pad0/Pad1 exist only to keep vec3 members aligned to
// 16 bytes and are never read by the shader.
type PipelineParams struct {
	LightPos                                            [3]float32
	LightIntensity                                      float32
	LightColor                                          [3]float32
	Pad0                                                float32
	BaseColor                                           [3]float32
	BaseWeight                                          float32
	SpecularRoughness, SpecularIOR, BaseMetalness, Pad1 float32
}

func (p PipelineParams) floats() [16]float32 {
	return [16]float32{
		p.LightPos[0], p.LightPos[1], p.LightPos[2], p.LightIntensity,
		p.LightColor[0], p.LightColor[1], p.LightColor[2], p.Pad0,
		p.BaseColor[0], p.BaseColor[1], p.BaseColor[2], p.BaseWeight,
		p.SpecularRoughness, p.SpecularIOR, p.BaseMetalness, p.Pad1,
	}
}

// BuildPipeline creates the ray-gen/miss/shadow-miss/closest-hit shader binding table
// and pipeline over the SAME BLAS/TLAS Build already created — call after Build.
func (s *RTScene) BuildPipeline(rgen, miss, shadowMiss, chit []byte) error {
	rc := C.obk_rt_scene_build_pipeline(s.handle,
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(rgen))), C.int(len(rgen)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(miss))), C.int(len(miss)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(shadowMiss))), C.int(len(shadowMiss)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(chit))), C.int(len(chit)))
	if rc != 0 {
		return errors.New("native: RTScene.BuildPipeline failed (see stderr)")
	}
	return nil
}

// TracePipeline dispatches one vkCmdTraceRaysKHR call (a single ray) through the full
// RT pipeline and returns the resulting single-bounce direct-lighting radiance.
func (s *RTScene) TracePipeline(origin, direction [3]float32, tMin, tMax float32, params PipelineParams) (r, g, b float32) {
	pf := params.floats()
	var cR, cG, cB C.float
	C.obk_rt_scene_trace_pipeline(s.handle,
		C.float(origin[0]), C.float(origin[1]), C.float(origin[2]),
		C.float(direction[0]), C.float(direction[1]), C.float(direction[2]),
		C.float(tMin), C.float(tMax),
		(*C.float)(unsafe.Pointer(&pf[0])), &cR, &cG, &cB)
	return float32(cR), float32(cG), float32(cB)
}

// CameraBasis is pathtrace_realistic.rgen/swpathtrace_realistic.comp's CameraParams
// UBO, laid out exactly (M45-F05 PBI-350): a pinhole eye/forward/right/up basis, unlike
// PipelineParams' single fixed ray — one call renders a whole image, not one pixel.
type CameraBasis struct {
	Eye         [3]float32
	TMin        float32
	Forward     [3]float32
	TMax        float32
	Right       [3]float32
	TanHalfFovY float32
	Up          [3]float32
	Aspect      float32
}

func (c CameraBasis) floats() [16]float32 {
	return [16]float32{
		c.Eye[0], c.Eye[1], c.Eye[2], c.TMin,
		c.Forward[0], c.Forward[1], c.Forward[2], c.TMax,
		c.Right[0], c.Right[1], c.Right[2], c.TanHalfFovY,
		c.Up[0], c.Up[1], c.Up[2], c.Aspect,
	}
}

// RealisticLightParams is pathtrace_realistic.rchit/swpathtrace_realistic.comp's Params
// UBO: LightDirection (a unit vector toward one directional light,
// renderer.SceneLight.Direction, not a point-light position — see those shaders' doc
// comments) plus the base lobes (the original 16-float/64-byte layout, unchanged) and,
// as of #2148, every extended OpenPBR lobe the live path tracer now shades: Coat,
// Fuzz/sheen, Thin-film, Transmission+dispersion, Subsurface — 56 floats/224 bytes
// total. A zero-value field group reproduces the prior lobe's output exactly (each
// lobe's own weight=0 short-circuit, mirrored from kernel/shading/openpbr's Layer*/Mix*
// functions), so callers that only set the base fields (e.g. the golden-harness base
// tier) are unaffected.
type RealisticLightParams struct {
	LightDirection                                      [3]float32
	LightIntensity                                      float32
	LightColor                                          [3]float32
	Pad0                                                float32
	BaseColor                                           [3]float32
	BaseWeight                                          float32
	SpecularRoughness, SpecularIOR, BaseMetalness, Pad1 float32

	CoatColor                             [3]float32
	CoatWeight                            float32
	CoatRoughness, CoatIOR, CoatDarkening float32
	Pad2                                  float32

	FuzzColor        [3]float32
	FuzzWeight       float32
	FuzzRoughness    float32
	Pad3, Pad4, Pad5 float32

	ThinFilmWeight, ThinFilmThicknessMicrons, ThinFilmIOR float32
	Pad6                                                  float32

	TransmissionColor                                              [3]float32
	TransmissionWeight                                             float32
	TransmissionDepth, DispersionScale, DispersionAbbeNumber, Pad7 float32

	SubsurfaceColor       [3]float32
	SubsurfaceWeight      float32
	SubsurfaceRadiusScale [3]float32
	SubsurfaceRadius      float32
	SubsurfaceAnisotropy  float32
	Pad8, Pad9, Pad10     float32
}

func (p RealisticLightParams) floats() [56]float32 {
	return [56]float32{
		p.LightDirection[0], p.LightDirection[1], p.LightDirection[2], p.LightIntensity,
		p.LightColor[0], p.LightColor[1], p.LightColor[2], p.Pad0,
		p.BaseColor[0], p.BaseColor[1], p.BaseColor[2], p.BaseWeight,
		p.SpecularRoughness, p.SpecularIOR, p.BaseMetalness, p.Pad1,

		p.CoatColor[0], p.CoatColor[1], p.CoatColor[2], p.CoatWeight,
		p.CoatRoughness, p.CoatIOR, p.CoatDarkening, p.Pad2,

		p.FuzzColor[0], p.FuzzColor[1], p.FuzzColor[2], p.FuzzWeight,
		p.FuzzRoughness, p.Pad3, p.Pad4, p.Pad5,

		p.ThinFilmWeight, p.ThinFilmThicknessMicrons, p.ThinFilmIOR, p.Pad6,

		p.TransmissionColor[0], p.TransmissionColor[1], p.TransmissionColor[2], p.TransmissionWeight,
		p.TransmissionDepth, p.DispersionScale, p.DispersionAbbeNumber, p.Pad7,

		p.SubsurfaceColor[0], p.SubsurfaceColor[1], p.SubsurfaceColor[2], p.SubsurfaceWeight,
		p.SubsurfaceRadiusScale[0], p.SubsurfaceRadiusScale[1], p.SubsurfaceRadiusScale[2], p.SubsurfaceRadius,
		p.SubsurfaceAnisotropy, p.Pad8, p.Pad9, p.Pad10,
	}
}

// BuildRealisticPipeline creates the live per-pixel Realistic-viewport pipeline over
// the same BLAS/TLAS Build already created — independent of BuildPipeline's single-ray
// test-harness pipeline (its own SBT/descriptor set), so that pipeline's already-verified
// PBI-345/346 tests are never at risk from a change here. Call after Build.
func (s *RTScene) BuildRealisticPipeline(rgen, miss, shadowMiss, chit []byte) error {
	rc := C.obk_rt_scene_build_realistic_pipeline(s.handle,
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(rgen))), C.int(len(rgen)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(miss))), C.int(len(miss)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(shadowMiss))), C.int(len(shadowMiss)),
		(*C.uint32_t)(unsafe.Pointer(unsafe.SliceData(chit))), C.int(len(chit)))
	if rc != 0 {
		return errors.New("native: RTScene.BuildRealisticPipeline failed (see stderr)")
	}
	return nil
}

// TraceRealisticImage dispatches one vkCmdTraceRaysKHR(width,height,1) call — one
// primary+shadow ray pair per pixel — and returns the resulting width*height RGB image
// (row-major, length 3*width*height, alpha dropped).
func (s *RTScene) TraceRealisticImage(width, height int, camera CameraBasis, params RealisticLightParams) ([]float32, error) {
	cf := camera.floats()
	pf := params.floats()
	out := make([]float32, width*height*3)
	rc := C.obk_rt_scene_trace_realistic_image(s.handle, C.int(width), C.int(height),
		(*C.float)(unsafe.Pointer(&cf[0])), (*C.float)(unsafe.Pointer(&pf[0])),
		(*C.float)(unsafe.Pointer(unsafe.SliceData(out))))
	if rc != 0 {
		return nil, errors.New("native: RTScene.TraceRealisticImage failed (see stderr)")
	}
	return out, nil
}

// Destroy frees every Vulkan resource the scene owns.
func (s *RTScene) Destroy() {
	C.obk_rt_scene_destroy(s.handle)
	s.handle = nil
}
