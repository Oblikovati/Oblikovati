// SPDX-License-Identifier: GPL-2.0-only

package icon

// Role is one color layer of an icon. An SVG asset assigns its elements to roles by
// painting them with the role's sentinel color (see sentinelPaint); the rasterizer
// splits the glyph into one coverage mask per role and Compose colors them from the
// theme (ADR-0033). Roles are declared in draw order: the background plate first, the
// primary linework on top.
type Role int

const (
	RoleBackground Role = iota // backplate behind the glyph
	RoleTertiary               // supporting detail: anchors, construction marks
	RoleSecondary              // accent: the action/result element of the glyph
	RolePrimary                // main linework
	RoleCount
)

// sentinelPaint is the SVG source color that assigns an element to each role. These
// are authoring placeholders, never shown on screen — the theme replaces them at
// composite time. Primary is black so the pre-existing monochrome art reads as
// primary without edits.
var sentinelPaint = [RoleCount]string{
	RoleBackground: "#00ff00",
	RoleTertiary:   "#0000ff",
	RoleSecondary:  "#ff0000",
	RolePrimary:    "#000000",
}

// String names a role for error messages and tests.
func (r Role) String() string {
	switch r {
	case RoleBackground:
		return "background"
	case RoleTertiary:
		return "tertiary"
	case RoleSecondary:
		return "secondary"
	case RolePrimary:
		return "primary"
	default:
		return "invalid"
	}
}
