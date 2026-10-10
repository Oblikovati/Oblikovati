// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, UnitsOfMeasure, registry::require_category};
use crate::quantity::Dimension;

impl UnitsOfMeasure {
    /// Read the stored length scale. Example: `assert_eq!(UnitsOfMeasure::default().working_scale(), 1.0);`
    pub fn working_scale(&self) -> f64 {
        self.centimetres_per_unit
    }

    /// Build a conversion context at a known scale; example: `units.with_working_scale(1e-4)`.
    /// The document owner must prevent scale changes after geometry is stored.
    pub fn with_working_scale(mut self, scale: f64) -> Result<Self, UnitError> {
        let cube = scale * scale * scale;
        if !scale.is_finite() || scale <= 0.0 || !cube.is_finite() || cube == 0.0 {
            return Err(UnitError::InvalidScale(scale));
        }
        self.centimetres_per_unit = scale;
        Ok(self)
    }

    /// Use a named length as the working unit; example: `units.centered_on_length("µm")`.
    pub fn centered_on_length(self, name: &str) -> Result<Self, UnitError> {
        self.with_working_scale(require_category(name, Dimension::Length)?.factor)
    }

    /// Select the scale for a new document; example: `UnitsOfMeasure::recommended_scale("km")`.
    pub fn recommended_scale(name: &str) -> Result<f64, UnitError> {
        let factor = require_category(name, Dimension::Length)?.factor;
        // Preserve the Go conditioning band; ordinary documents retain centimetres.
        if !(1e-3..=1e4).contains(&factor) {
            return Ok(factor);
        }
        Ok(1.0)
    }

    pub(super) fn scale_factor(&self, dimension: Dimension) -> f64 {
        let scale = self.centimetres_per_unit;
        match dimension {
            Dimension::Length => scale,
            Dimension::Area => scale * scale,
            Dimension::Volume => scale * scale * scale,
            _ => 1.0,
        }
    }
}
