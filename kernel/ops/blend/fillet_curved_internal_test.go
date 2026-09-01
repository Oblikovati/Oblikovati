// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// TestSurfaceKindNames covers every branch of the error-message surface namer, including the
// %T fallback for a surface the switch does not special-case.
func TestSurfaceKindNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    geom.Surface
		want string
	}{
		{geom.Cylinder{}, "cylinder"},
		{geom.Cone{}, "cone"},
		{geom.Sphere{}, "sphere"},
		{geom.Torus{}, "torus"},
		{geom.Plane{}, "geom.Plane"}, // default: the %T type name
	}
	for _, c := range cases {
		if got := surfaceKind(c.s); got != c.want {
			t.Errorf("surfaceKind(%T) = %q, want %q", c.s, got, c.want)
		}
	}
}
