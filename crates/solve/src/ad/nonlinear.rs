// SPDX-License-Identifier: GPL-2.0-only
use super::{DerivativeError, DualNumber};

impl DualNumber {
    /// Square root, with the Go zero-gradient convention at zero (not a derivative there).
    /// Example: `squared_distance.sqrt()?`.
    pub fn sqrt(&self) -> Result<Self, DerivativeError> {
        let root = self.value.sqrt();
        self.unary(root, if root == 0.0 { 0.0 } else { 0.5 / root }, "sqrt")
    }
    /// Euclidean norm with zero gradient at the nonsmooth origin; example: `x.hypot(&y)?`.
    pub fn hypot(&self, other: &Self) -> Result<Self, DerivativeError> {
        let length = self.value.hypot(other.value);
        let (left, right) = if length == 0.0 {
            (0.0, 0.0)
        } else {
            (self.value / length, other.value / length)
        };
        self.binary(other, length, left, right, "hypot")
    }
    /// Absolute residual, with zero gradient at the kink; example: `distance.abs()?`.
    pub fn abs(&self) -> Result<Self, DerivativeError> {
        let sign = if self.value == 0.0 {
            0.0
        } else {
            self.value.signum()
        };
        self.unary(self.value.abs(), sign, "absolute")
    }
    /// Angle atan2(self, x), with zero gradient at the undefined origin.
    /// Example: `y.atan2(&x)?`.
    pub fn atan2(&self, x: &Self) -> Result<Self, DerivativeError> {
        // Normalize first: x*x+y*y and even hypot(x,y) can overflow (#3698).
        let scale = self.value.abs().max(x.value.abs());
        let (dy, dx) = if scale == 0.0 {
            (0.0, 0.0)
        } else {
            let (sy, sx) = (self.value / scale, x.value / scale);
            let norm = sx.hypot(sy);
            ((sx / norm) / norm / scale, -(sy / norm) / norm / scale)
        };
        self.binary(x, self.value.atan2(x.value), dy, dx, "atan2")
    }
    /// Differentiate a radian sine; example: `angle.sin()?`.
    pub fn sin(&self) -> Result<Self, DerivativeError> {
        self.unary(self.value.sin(), self.value.cos(), "sine")
    }
    /// Differentiate a radian cosine; example: `angle.cos()?`.
    pub fn cos(&self) -> Result<Self, DerivativeError> {
        self.unary(self.value.cos(), -self.value.sin(), "cosine")
    }
}
