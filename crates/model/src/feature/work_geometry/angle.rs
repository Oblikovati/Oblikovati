// SPDX-License-Identifier: GPL-2.0-only
use super::{DatumLength, WorkGeometryError};
use crate::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
};

/// A finite radian constant or the current angular model value of one stable parameter.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(transparent)]
pub struct DatumAngle(DatumLength);
impl DatumAngle {
    /// Check a radian constant before authoring; example: `DatumAngle::constant(std::f64::consts::FRAC_PI_2)?`.
    pub fn constant(radians: f64) -> Result<Self, WorkGeometryError> {
        let source = Self(DatumLength::Constant(radians));
        source.validate_source()?;
        Ok(source)
    }
    /// Retain a stable parameter source; example: `DatumAngle::parameter(42)`.
    pub fn parameter(id: u64) -> Self {
        Self(DatumLength::Parameter(id))
    }
    /// Resolve current angular model units; example: `angle.resolve(&parameters)?` returns radians.
    pub fn resolve(self, parameters: &ParameterCollection<u64>) -> Result<f64, WorkGeometryError> {
        let quantity = self
            .0
            .source_quantity(parameters, Dimension::Angle)
            .map_err(WorkGeometryError::Parameter)?;
        require_angle(quantity, self.parameter_id())
    }
    /// Refuse nonfinite authored constants before adoption; example: `angle.validate_source()?` after decoding.
    pub fn validate_source(self) -> Result<(), WorkGeometryError> {
        if let DatumLength::Constant(value) = self.0 {
            require_angle(Quantity::new(value, Dimension::Angle), None)?;
        }
        Ok(())
    }
    /// Attribute the stable angular driver; example: `angle.parameter_id()` for the read graph.
    pub fn parameter_id(self) -> Option<u64> {
        self.0.parameter()
    }
    /// Compare authored scalar bits and references; example: `before.same_recipe(after)`.
    pub fn same_recipe(self, other: Self) -> bool {
        self.0.same_recipe(other.0)
    }
}
fn require_angle(quantity: Quantity, parameter: Option<u64>) -> Result<f64, WorkGeometryError> {
    if quantity.dimension != Dimension::Angle || !quantity.value.is_finite() {
        return Err(WorkGeometryError::Angle(DatumAngleError {
            parameter,
            quantity,
        }));
    }
    Ok(quantity.value)
}

/// An angular decline retains the driver and the offending model quantity.
#[derive(Clone, Debug, PartialEq)]
pub struct DatumAngleError {
    pub parameter: Option<u64>,
    pub quantity: Quantity,
}
impl std::fmt::Display for DatumAngleError {
    fn fmt(&self, output: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let Self {
            parameter,
            quantity,
        } = self;
        write!(
            output,
            "datum angle parameter {parameter:?} value {quantity:?}: expected finite Angle in radians"
        )
    }
}
impl std::error::Error for DatumAngleError {}
