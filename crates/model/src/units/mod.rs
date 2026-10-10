// SPDX-License-Identifier: GPL-2.0-only

//! Convert named user units to document working units without changing stored quantities.
mod display;
mod error;
pub use display::{AngleFormat, DisplayOptions};
mod literal;
mod registry;
mod scale;
mod service;
mod transfer;
pub use service::{category_name, convert_value, parse_category, unit_category};

use crate::quantity::{Dimension, Quantity};
pub use error::UnitError;
pub(crate) use literal::numeric_prefix_len;
use registry::{NamedUnit, require_category};

/// Document unit preferences. Changes affect presentation, never existing quantities.
#[derive(Clone, Debug, PartialEq)]
pub struct UnitsOfMeasure {
    preferences: Vec<NamedUnit>,
    centimetres_per_unit: f64,
    display: DisplayOptions,
}

impl Default for UnitsOfMeasure {
    fn default() -> Self {
        let preferences = ["mm", "deg", "mm^2", "mm^3", "kg", "s"]
            .into_iter()
            .map(|name| registry::lookup(name).expect("default unit is registered"))
            .collect();
        Self {
            preferences,
            centimetres_per_unit: 1.0,
            display: DisplayOptions::default(),
        }
    }
}

impl UnitsOfMeasure {
    /// Set a display unit without rescaling storage; example: `units.set_preferred(Dimension::Length, "in")`.
    pub fn set_preferred(&mut self, dimension: Dimension, name: &str) -> Result<(), UnitError> {
        let replacement = require_category(name, dimension)?;
        let index = self
            .preferences
            .iter()
            .position(|unit| unit.dimension == dimension)
            .expect("each registered dimension has a default preference");
        self.preferences[index] = replacement;
        Ok(())
    }

    /// Read a display suffix; example: `units.preferred_name(Dimension::Length)`.
    pub fn preferred_name(&self, dimension: Dimension) -> &'static str {
        self.preferred(dimension).map_or("", |unit| unit.name)
    }

    /// Convert a user value to working units; example: `units.from_preferred(25.0, Dimension::Length)`.
    pub fn from_preferred(&self, value: f64, dimension: Dimension) -> Quantity {
        self.to_working(value, dimension, self.preferred_factor(dimension))
    }

    /// Convert a stored value for display; example: `units.to_preferred(length)`.
    pub fn to_preferred(&self, quantity: Quantity) -> f64 {
        quantity.value * self.scale_factor(quantity.dimension)
            / self.preferred_factor(quantity.dimension)
    }

    fn preferred(&self, dimension: Dimension) -> Option<NamedUnit> {
        self.preferences
            .iter()
            .copied()
            .find(|unit| unit.dimension == dimension)
    }

    fn preferred_factor(&self, dimension: Dimension) -> f64 {
        self.preferred(dimension).map_or(1.0, |unit| unit.factor)
    }

    fn to_working(&self, value: f64, dimension: Dimension, factor: f64) -> Quantity {
        Quantity::new(value * factor / self.scale_factor(dimension), dimension)
    }
}
