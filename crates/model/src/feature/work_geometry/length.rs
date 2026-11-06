// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
};
mod error;
pub use error::DatumLengthError;

/// A document working-unit length or the current model value of a stable parameter.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum DatumLength {
    Constant(f64),
    Parameter(u64),
}
impl DatumLength {
    /// Resolve current length without a stored callback; example: `DatumLength::Parameter(1).resolve(&parameters)?`.
    pub fn resolve(self, parameters: &ParameterCollection<u64>) -> Result<f64, DatumLengthError> {
        let parameter = self.parameter();
        let quantity = self
            .source_quantity(parameters, Dimension::Length)
            .map_err(DatumLengthError::Parameter)?;
        if quantity.dimension != Dimension::Length || !quantity.value.is_finite() {
            return Err(DatumLengthError::Quantity {
                parameter,
                quantity,
            });
        }
        Ok(quantity.value)
    }
    /// Validate authored constants before recipe adoption; example: `coordinate.validate_source()?`.
    pub fn validate_source(self) -> Result<(), DatumLengthError> {
        if let Self::Constant(value) = self
            && !value.is_finite()
        {
            return Err(DatumLengthError::Quantity {
                parameter: None,
                quantity: Quantity::new(value, Dimension::Length),
            });
        }
        Ok(())
    }
    /// Attribute an optional parameter dependency; example: `coordinate.parameter()`.
    pub fn parameter(self) -> Option<u64> {
        match self {
            Self::Constant(_) => None,
            Self::Parameter(id) => Some(id),
        }
    }
    /// Compare exact scalar sources including signed zero; example: `before.same_recipe(after)`.
    pub fn same_recipe(self, other: Self) -> bool {
        match (self, other) {
            (Self::Constant(first), Self::Constant(second)) => first.to_bits() == second.to_bits(),
            (Self::Parameter(first), Self::Parameter(second)) => first == second,
            _ => false,
        }
    }
}

impl DatumLength {
    pub(super) fn source_quantity(
        self,
        parameters: &ParameterCollection<u64>,
        dimension: Dimension,
    ) -> Result<Quantity, crate::parameter::ParameterError<u64>> {
        match self {
            Self::Constant(value) => Ok(Quantity::new(value, dimension)),
            Self::Parameter(id) => parameters.model_value(id),
        }
    }
}
