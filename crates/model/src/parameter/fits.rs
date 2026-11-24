// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{fits::fit_band, quantity::Dimension};
use oblikovati_api::types::{
    TOLERANCE_LIMITS_FITS_LINEAR, TOLERANCE_LIMITS_FITS_SHOW_SIZE,
    TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE, TOLERANCE_LIMITS_FITS_STACKED, Tolerance, ToleranceType,
};

/// Identify formats whose deviations come from ISO fit classes.
/// For example, `is_fit_tolerance(TOLERANCE_LIMITS_FITS_STACKED)` is true.
pub fn is_fit_tolerance(kind: ToleranceType) -> bool {
    matches!(
        kind,
        TOLERANCE_LIMITS_FITS_STACKED
            | TOLERANCE_LIMITS_FITS_LINEAR
            | TOLERANCE_LIMITS_FITS_SHOW_SIZE
            | TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE
    )
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Store fit annotations and deviations; for example `parameters.set_fits(id, "H7", "g6")`.
    /// The hole governs when both classes are present. Both classes must be supported.
    pub fn set_fits(&mut self, id: Id, hole: &str, shaft: &str) -> Result<(), ParameterError<Id>> {
        let nominal = self.require_numeric_value(id)?;
        if nominal.dimension != Dimension::Length {
            return Err(ParameterError::Value {
                id,
                found: format!("{:?}", nominal.dimension),
                expected: "a length parameter for a fit",
            });
        }
        let tolerance =
            resolve_fit_tolerance(nominal.value, self.units.working_scale(), hole, shaft).map_err(
                |error| ParameterError::Value {
                    id,
                    found: error.to_string(),
                    expected: "supported fit classes and nominal size",
                },
            )?;
        self.store_tolerance(id, tolerance)
    }
}

/// Resolve ISO fit bounds from a length in document working units.
/// Example: `resolve_fit_tolerance(0.8, 1., "H7", "")?`.
pub fn resolve_fit_tolerance(
    nominal: f64,
    scale: f64,
    hole: &str,
    shaft: &str,
) -> Result<Tolerance, crate::fits::FitError> {
    let nominal_mm = nominal * 10. * scale;
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
