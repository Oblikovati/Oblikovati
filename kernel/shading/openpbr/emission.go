// SPDX-License-Identifier: GPL-2.0-only

package openpbr

// Emission evaluates the OpenPBR Surface Emission group (spec §Emission): self-emitted
// radiance, independent of every reflective/transmissive lobe — luminance (nits) times
// color, with no BSDF evaluation (it is added to a path's radiance, not reflected).
func Emission(luminance float64, color Color3) Color3 { return color.Scale(luminance) }
