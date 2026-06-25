// SPDX-License-Identifier: GPL-2.0-only

package ad

// Vec2 and Vec3 are dual-number vectors — the planar and spatial points/directions
// constraint residuals are written in terms of, so a residual reads like its
// geometric definition (cross, dot, length) while carrying exact derivatives. They
// hold [Number] components, not raw floats, so every operation differentiates.

// Vec2 is a 2D dual vector.
type Vec2 struct{ X, Y Number }

// Vec3 is a 3D dual vector.
type Vec3 struct{ X, Y, Z Number }

// V2 builds a 2D dual vector from two components.
func V2(x, y Number) Vec2 { return Vec2{X: x, Y: y} }

// V3 builds a 3D dual vector from three components.
func V3(x, y, z Number) Vec3 { return Vec3{X: x, Y: y, Z: z} }

// Sub returns a−b componentwise.
func (a Vec2) Sub(b Vec2) Vec2 { return Vec2{X: a.X.Sub(b.X), Y: a.Y.Sub(b.Y)} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{X: a.X.Sub(b.X), Y: a.Y.Sub(b.Y), Z: a.Z.Sub(b.Z)} }

// Add returns a+b componentwise.
func (a Vec2) Add(b Vec2) Vec2 { return Vec2{X: a.X.Add(b.X), Y: a.Y.Add(b.Y)} }
func (a Vec3) Add(b Vec3) Vec3 { return Vec3{X: a.X.Add(b.X), Y: a.Y.Add(b.Y), Z: a.Z.Add(b.Z)} }

// Scale multiplies every component by the constant c.
func (a Vec2) Scale(c float64) Vec2 { return Vec2{X: a.X.Scale(c), Y: a.Y.Scale(c)} }
func (a Vec3) Scale(c float64) Vec3 { return Vec3{X: a.X.Scale(c), Y: a.Y.Scale(c), Z: a.Z.Scale(c)} }

// Dot returns a·b.
func (a Vec2) Dot(b Vec2) Number { return a.X.Mul(b.X).Add(a.Y.Mul(b.Y)) }
func (a Vec3) Dot(b Vec3) Number {
	return a.X.Mul(b.X).Add(a.Y.Mul(b.Y)).Add(a.Z.Mul(b.Z))
}

// Cross returns the 2D scalar cross product aₓbᵧ − aᵧbₓ (the signed parallelogram
// area), zero iff a and b are parallel.
func (a Vec2) Cross(b Vec2) Number { return a.X.Mul(b.Y).Sub(a.Y.Mul(b.X)) }

// Cross returns the 3D cross product a×b.
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y.Mul(b.Z).Sub(a.Z.Mul(b.Y)),
		Y: a.Z.Mul(b.X).Sub(a.X.Mul(b.Z)),
		Z: a.X.Mul(b.Y).Sub(a.Y.Mul(b.X)),
	}
}

// Length returns the Euclidean length √(x²+y²[+z²]).
func (a Vec2) Length() Number { return a.X.Hypot(a.Y) }
func (a Vec3) Length() Number { return a.Dot(a).Sqrt() }
