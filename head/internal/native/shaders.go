//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import _ "embed"

// Compiled SPIR-V for the viewport mesh pipeline, embedded into the binary so the head
// needs no shader files at runtime. Regenerate from the .vert/.frag with
// `make shaders` (glslangValidator) after editing them.

//go:embed shaders/mesh.vert.spv
var meshVertSPV []byte

//go:embed shaders/mesh.frag.spv
var meshFragSPV []byte

// Point-cloud pipeline SPIR-V (#645): one GPU vertex per scan point (POINT_LIST), replacing the
// per-frame 3-axis-cross line expansion.
//
//go:embed shaders/point.vert.spv
var pointVertSPV []byte

//go:embed shaders/point.frag.spv
var pointFragSPV []byte

// Wide-line pipeline SPIR-V (#2015): expands a stroked segment into a screen-space quad, so a line
// weight renders at a constant pixel width without the non-portable wideLines Vulkan feature.
//
//go:embed shaders/wideline.vert.spv
var wideLineVertSPV []byte

//go:embed shaders/wideline.frag.spv
var wideLineFragSPV []byte

//go:embed shaders/skybox.vert.spv
var skyboxVertSPV []byte

//go:embed shaders/skybox.frag.spv
var skyboxFragSPV []byte

// Hardware ray-query Intersector compute shader (M45-F01 PBI-333, ADR-0053). --target-env
// vulkan1.3 (not the others' implicit vulkan1.0) because GL_EXT_ray_query needs SPIR-V 1.4+.
//
//go:embed shaders/raytrace.comp.spv
var raytraceCompSPV []byte

// Software BVH-traversal compute shader (M45-F01 PBI-334, ADR-0053): no ray-tracing
// extensions, so no --target-env bump is strictly required, but built the same way as
// raytrace.comp.spv for consistency (and SPIR-V 1.4+'s Shader Storage Buffer improvements).
//
//go:embed shaders/swtrace.comp.spv
var swtraceCompSPV []byte

// Full hardware RT pipeline: ray-gen/miss/shadow-miss/closest-hit (M45-F04 PBI-345,
// ADR-0053) — a single-bounce, single-light direct-lighting harness wiring the F03
// OpenPBR GLSL library to the F01 hardware backend's acceleration structures.

//go:embed shaders/pathtrace.rgen.spv
var pathtraceRgenSPV []byte

//go:embed shaders/pathtrace.rmiss.spv
var pathtraceRmissSPV []byte

//go:embed shaders/shadow.rmiss.spv
var shadowRmissSPV []byte

//go:embed shaders/pathtrace.rchit.spv
var pathtraceRchitSPV []byte

// Software single-bounce path-tracing harness (M45-F04 PBI-346): the same shading math
// as pathtrace.rchit, but as one compute shader over the software BVH (PBI-334)
// instead of a full RT pipeline.
//
//go:embed shaders/swpathtrace.comp.spv
var swpathtraceCompSPV []byte
