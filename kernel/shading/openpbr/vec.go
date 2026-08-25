// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// Vec3 is a local-shading-space direction or color-independent 3-vector. Kept
// self-contained (not the CAD kernel's arbitrary-precision geometry types) because BSDF
// evaluation is an ordinary floating-point numerical problem, not exact geometry.
type Vec3 struct{ X, Y, Z float64 }

// Color3 is an RGB reflectance/radiance value in the working color space (ACEScg for
// OpenPBR — see [github.com/oblikovati.org/api/types.Color3]; this package stays
// color-space-agnostic and just does the arithmetic).
type Color3 struct{ R, G, B float64 }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

// Normalize returns a unit vector, or the zero vector for a (near-)zero input.
func (a Vec3) Normalize() Vec3 {
	l := math.Sqrt(a.Dot(a))
	if l == 0 {
		return a
	}
	return a.Scale(1 / l)
}

// CosTheta is the local-space polar cosine — the convention throughout this package (Z
// is the macrosurface normal), matching Adobe's reference's direction_local.z.
func (a Vec3) CosTheta() float64 { return a.Z }

func NewColor3(r, g, b float64) Color3 { return Color3{R: r, G: g, B: b} }

// Gray returns an achromatic color with every channel equal to v.
func Gray(v float64) Color3 { return Color3{R: v, G: v, B: v} }

func (c Color3) Add(o Color3) Color3 { return Color3{c.R + o.R, c.G + o.G, c.B + o.B} }

func (c Color3) Mul(o Color3) Color3 { return Color3{c.R * o.R, c.G * o.G, c.B * o.B} }

func (c Color3) Scale(s float64) Color3 { return Color3{c.R * s, c.G * s, c.B * s} }

// MaxChannel returns the largest of the three channels (used for lobe-contribution /
// energy-conservation checks, mirroring Adobe's openpbr_max3).
func (c Color3) MaxChannel() float64 { return math.Max(c.R, math.Max(c.G, c.B)) }
