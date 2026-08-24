// SPDX-License-Identifier: GPL-2.0-only

// Package openpbr is a pure-Go CPU reference for the OpenPBR Surface v1.1.1 BSDF lobes
// (github.com/AcademySoftwareFoundation/OpenPBR, M45, ADR-0053). It exists to be
// oracle-tested at fixed angles and to define the exact math the shared GLSL library
// (head/internal/native/shaders/openpbr) ports for the F04 path-tracing backends — this
// package is never itself on the hot rendering path.
//
// Formulas are ported from two Apache-2.0 sources, cited per function: the OpenPBR spec
// itself (github.com/AcademySoftwareFoundation/OpenPBR, index.html "model" sections) and
// Adobe's reference implementation (github.com/adobe/openpbr-bsdf, impl/*.h), which the
// spec's own multi-scatter section names as one of the "[Kulla2017]"/"[Turquin2019]"
// simpler-than-Monte-Carlo alternatives it explicitly sanctions.
//
// Directions (Vec3) are always in local shading space: Z is the macrosurface normal, and
// both wi ("view"/incident) and wo ("light"/outgoing) point AWAY from the surface (the
// convention Adobe's reference and PBRT share). World-space-to-local transforms are an
// integration concern for the F04 path integrator, not this package.
//
// Multi-scatter energy compensation here is the Kulla-Conty analytic form (average
// directional albedo × average Fresnel), not Adobe's baked 3D lookup table
// (impl/openpbr_microfacet_multiple_scattering_data.h) — the spec explicitly permits this
// as a "[Kulla2017]"-style alternative, and it avoids vendoring their large precomputed
// data arrays for a difference that only affects convergence smoothness at grazing
// angles, not energy conservation (see multiscatter.go).
package openpbr
