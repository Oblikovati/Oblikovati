// SPDX-License-Identifier: GPL-2.0-only

//! Dimension arithmetic over values already expressed in document working units.
//! Named-unit conversion belongs at the expression and display boundaries.

mod dimension;
mod error;
pub use dimension::Dimension;
pub use error::QuantityError;

/// An immutable numeric value in the document's working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct Quantity {
    pub value: f64,
    pub dimension: Dimension,
}

impl Quantity {
    /// Construct a working-unit value, for example `Quantity::new(2.0, Dimension::Length)`.
    pub const fn new(value: f64, dimension: Dimension) -> Self {
        Self { value, dimension }
    }

    /// Add equal categories, for example `length.checked_add(other_length)`.
    pub fn checked_add(self, other: Self) -> Result<Self, QuantityError> {
        self.same_dimension(other, "+", self.value + other.value)
    }

    /// Subtract equal categories, for example `length.checked_sub(other_length)`.
    pub fn checked_sub(self, other: Self) -> Result<Self, QuantityError> {
        self.same_dimension(other, "-", self.value - other.value)
    }

    /// Multiply into a named dimension, for example `length.checked_mul(length)`.
    pub fn checked_mul(self, other: Self) -> Result<Self, QuantityError> {
        self.combine(other, "*", 1, self.value * other.value)
    }

    /// Divide into a named dimension, for example `volume.checked_div(area)`.
    pub fn checked_div(self, other: Self) -> Result<Self, QuantityError> {
        if other.value == 0.0 {
            return Err(QuantityError::ZeroDivisor {
                dividend: self,
                divisor: other,
            });
        }
        self.combine(other, "/", -1, self.value / other.value)
    }

    /// Change the sign without converting units, for example `length.negated()`.
    pub fn negated(self) -> Self {
        Self::new(-self.value, self.dimension)
    }

    fn same_dimension(
        self,
        other: Self,
        operator: &'static str,
        value: f64,
    ) -> Result<Self, QuantityError> {
        if self.dimension != other.dimension {
            return Err(QuantityError::Incompatible {
                operator,
                left: self.dimension,
                right: other.dimension,
            });
        }
        // Go permits same-category sums, including flag/text categories. The
        // expression evaluator owns the restriction on authored nonnumeric values.
        Ok(Self::new(value, self.dimension))
    }

    fn combine(
        self,
        other: Self,
        operator: &'static str,
        sign: i8,
        value: f64,
    ) -> Result<Self, QuantityError> {
        let dimension =
            self.dimension
                .combine(other.dimension, sign)
                .ok_or(QuantityError::Incompatible {
                    operator,
                    left: self.dimension,
                    right: other.dimension,
                })?;
        Ok(Self::new(value, dimension))
    }
}
