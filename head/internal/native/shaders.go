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
