// SPDX-License-Identifier: GPL-2.0-only

//! Forward-mode residual derivatives; inputs and gradients are owned and immutable.
mod arithmetic;
mod nonlinear;
use std::{error::Error, fmt};

/// A scalar and its partial derivatives; an empty gradient denotes a constant.
#[derive(Clone, Debug, PartialEq)]
pub struct DualNumber {
    value: f64,
    gradient: Vec<f64>,
}

/// Invalid seed, gradient dimensions, or arithmetic domain.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DerivativeError(String);
impl fmt::Display for DerivativeError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(output)
    }
}
impl Error for DerivativeError {}

impl DualNumber {
    /// Construct a finite value with no active variables; example: `DualNumber::constant(2.0)?`.
    pub fn constant(value: f64) -> Result<Self, DerivativeError> {
        Self::checked(value, Vec::new(), "constant")
    }

    /// Seed one variable with a unit gradient; example: `DualNumber::variable(3.0, 0, 2)?`.
    pub fn variable(value: f64, index: usize, count: usize) -> Result<Self, DerivativeError> {
        if index >= count {
            return Err(DerivativeError(format!(
                "variable index {index}: expected index below variable count {count}"
            )));
        }
        let mut gradient = vec![0.0; count];
        gradient[index] = 1.0;
        Self::checked(value, gradient, "variable")
    }

    /// Seed all coordinates without changing the input; example: `let vars = DualNumber::seed(&[3.0, 4.0])?;`.
    pub fn seed(values: &[f64]) -> Result<Vec<Self>, DerivativeError> {
        values
            .iter()
            .enumerate()
            .map(|(index, &value)| Self::variable(value, index, values.len()))
            .collect()
    }

    /// Inspect the residual value; example: `assert_eq!(DualNumber::constant(2.0)?.value(), 2.0);`.
    pub fn value(&self) -> f64 {
        self.value
    }

    /// Inspect immutable partials in seed order; example: `let row = residual.gradient();`.
    pub fn gradient(&self) -> &[f64] {
        &self.gradient
    }

    fn checked(value: f64, gradient: Vec<f64>, operation: &str) -> Result<Self, DerivativeError> {
        if !value.is_finite() || gradient.iter().any(|partial| !partial.is_finite()) {
            return Err(DerivativeError(format!(
                "{operation} value {value:?}, gradient {gradient:?}: expected finite value and partials"
            )));
        }
        Ok(Self { value, gradient })
    }

    fn unary(&self, value: f64, partial: f64, operation: &str) -> Result<Self, DerivativeError> {
        let gradient = self.gradient.iter().map(|entry| partial * entry).collect();
        Self::checked(value, gradient, operation)
    }

    fn binary(
        &self,
        other: &Self,
        value: f64,
        left: f64,
        right: f64,
        operation: &str,
    ) -> Result<Self, DerivativeError> {
        let (a, b) = (self.gradient.len(), other.gradient.len());
        if a != 0 && b != 0 && a != b {
            return Err(DerivativeError(format!(
                "{operation} gradient lengths {a}, {b}: expected equal seeded dimensions or a constant"
            )));
        }
        let gradient = (0..a.max(b))
            .map(|i| {
                let first = self.gradient.get(i).map_or(0.0, |entry| left * entry);
                first + other.gradient.get(i).map_or(0.0, |entry| right * entry)
            })
            .collect();
        Self::checked(value, gradient, operation)
    }
}
