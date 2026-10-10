// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{fits::fit_band, quantity::Dimension};
use oblikovati_api::types::{TOLERANCE_LIMITS_FITS_STACKED, Tolerance};

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Store fit annotations and deviations; for example `parameters.set_fits(id, "H7", "g6")`.
    /// The hole governs when both classes are present. Both classes must be supported.
    pub fn set_fits(&mut self, id: Id, hole: &str, shaft: &str) -> Result<(), ParameterError<Id>> {
        let nominal_mm = self.fit_nominal_mm(id)?;
        let scale = self.units.working_scale();
        let tolerance = resolve_tolerance(nominal_mm, hole, shaft, scale).map_err(|error| {
            ParameterError::Value {
                id,
                found: error.to_string(),
                expected: "supported fit classes and nominal size",
            }
        })?;
        self.store_tolerance(id, tolerance)
    }

    fn fit_nominal_mm(&self, id: Id) -> Result<f64, ParameterError<Id>> {
        let nominal = self.require_numeric_value(id)?;
        if nominal.dimension != Dimension::Length {
            return Err(ParameterError::Value {
                id,
                found: format!("{:?}", nominal.dimension),
                expected: "a length parameter for a fit",
            });
        }
        Ok(nominal.value * 10.0 * self.units.working_scale())
    }
}

fn resolve_tolerance(
    nominal_mm: f64,
    hole: &str,
    shaft: &str,
    scale: f64,
) -> Result<Tolerance, crate::fits::FitError> {
    let governing = if hole.is_empty() { shaft } else { hole };
    let band = fit_band(nominal_mm, governing)?;
    if !hole.is_empty() && !shaft.is_empty() {
        fit_band(nominal_mm, shaft)?;
    }
    Ok(Tolerance {
        type_: TOLERANCE_LIMITS_FITS_STACKED,
        upper: band.upper_mm / 10.0 / scale,
        lower: band.lower_mm / 10.0 / scale,
        hole_tolerance: hole.into(),
        shaft_tolerance: shaft.into(),
    })
}
