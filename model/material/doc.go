// SPDX-License-Identifier: GPL-2.0-only

// Package material owns the application's appearances and materials (ADR-0022).
//
// An Appearance is a metallic-roughness PBR description (what the renderer shows); a
// Material is a physical-world material (density + mechanical/thermal/electrical
// properties) that references an Appearance by id. Both are assets with a [Source]
// (built-in, project, or document), held in a per-session [Library] that mirrors the
// theme library: lookups by id, duplicate/edit/remove, and a monotonic revision counter
// the head watches to refresh the viewport live.
//
// Inventor models these as generic Asset/AssetValue bags; we use concrete typed structs
// (consistent with model/param and model/sketch), keeping Inventor's names and the
// appearance source/override precedence. The package is pure data with no cgo/renderer
// dependency, so it is fully headless-testable; the head maps an effective Appearance to
// the renderer's PBR surface.
package material

import "oblikovati.org/api/types"

// Aliases of the canonical Apache-2.0 value types so callers program against the material
// package without importing api/types directly (ADR-0018).
type (
	Source             = types.AssetSource
	Rgba               = types.Rgba
	Mechanical         = types.Mechanical
	Thermal            = types.Thermal
	Electrical         = types.Electrical
	Magnetic           = types.Magnetic
	MagneticClass      = types.MagneticClass
	IsotropyClass      = types.IsotropyClass
	AnisotropicElastic = types.AnisotropicElastic
	PhysicalProperties = types.PhysicalProperties
)

const (
	SourceBuiltin  = types.AssetBuiltin
	SourceProject  = types.AssetProject
	SourceDocument = types.AssetDocument

	Isotropic             = types.Isotropic
	Orthotropic           = types.Orthotropic
	TransverselyIsotropic = types.TransverselyIsotropic

	NonMagnetic  = types.NonMagnetic
	SoftMagnetic = types.SoftMagnetic
	HardMagnetic = types.HardMagnetic
)

// ParseColor parses a "#RRGGBBAA" (or "#RRGGBB") hex color into an Rgba — re-exported
// from the canonical types parser so the head editors and recipe code share one path.
func ParseColor(s string) (Rgba, error) { return types.ParseHex(s) }
