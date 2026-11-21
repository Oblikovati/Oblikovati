// SPDX-License-Identifier: GPL-2.0-only
use super::{DerivativeError, DualNumber};

impl DualNumber {
    /// Sum residual terms; example: `a.plus(&b)?`.
    pub fn plus(&self, other: &Self) -> Result<Self, DerivativeError> {
        self.binary(other, self.value + other.value, 1.0, 1.0, "sum")
    }
    /// Subtract a target; example: `a.minus(&target)?`.
    pub fn minus(&self, other: &Self) -> Result<Self, DerivativeError> {
        self.binary(other, self.value - other.value, 1.0, -1.0, "difference")
    }
    /// Apply the product rule; example: `a.product(&b)?`.
    pub fn product(&self, other: &Self) -> Result<Self, DerivativeError> {
        self.binary(
            other,
            self.value * other.value,
            other.value,
            self.value,
            "product",
        )
    }
    /// Apply the quotient rule, refusing undefined or non-finite results; example: `a.quotient(&b)?`.
    pub fn quotient(&self, other: &Self) -> Result<Self, DerivativeError> {
        let value = self.value / other.value;
        self.binary(
            other,
            value,
            1.0 / other.value,
            -value / other.value,
            "quotient",
        )
    }
    /// Reverse residual orientation; example: `a.negated()?`.
    pub fn negated(&self) -> Result<Self, DerivativeError> {
        self.unary(-self.value, -1.0, "negation")
    }
    /// Add a fixed target offset; example: `a.shifted(-3.0)?`.
    pub fn shifted(&self, offset: f64) -> Result<Self, DerivativeError> {
        Self::checked(self.value + offset, self.gradient.clone(), "offset")
    }
    /// Scale value and partials; example: `a.scaled(0.01)?`.
    pub fn scaled(&self, factor: f64) -> Result<Self, DerivativeError> {
        self.unary(self.value * factor, factor, "scale")
    }
}
