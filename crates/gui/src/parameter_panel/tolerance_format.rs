// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiParameterEngineering;
use oblikovati_api::types::*;
pub(super) use oblikovati_model::parameter::is_fit_tolerance as is_fit;
use oblikovati_model::{
    parameter::{resolve_fit_tolerance, tolerance_offset, vocabulary::TOLERANCE_NAMES},
    quantity::{Dimension, Quantity},
};

pub(super) const FORMATS: [(ToleranceType, &str); 12] = [
    (TOLERANCE_DEFAULT, "Default"),
    (TOLERANCE_BASIC, "Basic"),
    (TOLERANCE_SYMMETRIC, "Symmetric"),
    (TOLERANCE_DEVIATION, "Deviation"),
    (TOLERANCE_LIMITS_STACKED, "Limits-Stacked"),
    (TOLERANCE_LIMIT_LINEAR, "Limits-Linear"),
    (TOLERANCE_MAX, "MAX"),
    (TOLERANCE_MIN, "MIN"),
    (TOLERANCE_LIMITS_FITS_STACKED, "Limits/Fits-Stacked"),
    (TOLERANCE_LIMITS_FITS_LINEAR, "Limits/Fits-Linear"),
    (
        TOLERANCE_LIMITS_FITS_SHOW_SIZE,
        "Limits/Fits-Show Size Limits",
    ),
    (
        TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE,
        "Limits/Fits-Show Tolerance",
    ),
];

pub(super) fn seed_kind(name: &str) -> Result<ToleranceType, String> {
    if name.is_empty() {
        return Ok(TOLERANCE_DEFAULT);
    }
    TOLERANCE_NAMES
        .iter()
        .find(|(wire, _)| *wire == name)
        .map(|(_, kind)| *kind)
        .ok_or_else(|| format!("tolerance format {name:?}: expected a canonical format"))
}

pub(super) fn kind_label(kind: ToleranceType) -> Option<&'static str> {
    match kind {
        TOLERANCE_REFERENCE => Some("Reference"),
        TOLERANCE_OVERRIDE => Some("Override"),
        ToleranceType(0) => Some("Default"),
        kind => FORMATS
            .iter()
            .find(|(value, _)| *value == kind)
            .map(|(_, label)| *label),
    }
}

pub(super) fn bounds_editable(kind: ToleranceType) -> bool {
    matches!(
        kind,
        TOLERANCE_SYMMETRIC
            | TOLERANCE_DEVIATION
            | TOLERANCE_LIMITS_STACKED
            | TOLERANCE_LIMIT_LINEAR
            | TOLERANCE_OVERRIDE
    )
}

pub(super) fn validate_band(band: &Tolerance) -> Result<(), String> {
    let fit = is_fit(band.type_);
    let annotations = !band.hole_tolerance.is_empty() || !band.shaft_tolerance.is_empty();
    let bounds_valid = if fit || bounds_editable(band.type_) {
        band.upper.is_finite() && band.lower.is_finite() && band.upper >= band.lower
    } else {
        band.upper == 0. && band.lower == 0.
    };
    let symmetric = band.type_ != TOLERANCE_SYMMETRIC || band.lower == -band.upper;
    if kind_label(band.type_).is_some()
        && fit == annotations
        && bounds_valid
        && symmetric
        && !band.hole_tolerance.contains('\0')
        && !band.shaft_tolerance.contains('\0')
    {
        return Ok(());
    }
    Err(format!(
        "GUI tolerance {band:?}: expected valid bounds and annotations for its format"
    ))
}

pub(super) fn fit_band(
    seed: Option<&GuiParameterEngineering>,
    kind: ToleranceType,
    hole: &str,
    shaft: &str,
) -> Result<Tolerance, String> {
    let seed = seed
        .filter(|seed| seed.nominal.dimension == Dimension::Length)
        .ok_or("tolerance fit: expected a length parameter")?;
    let mut band = resolve_fit_tolerance(
        seed.nominal.value,
        seed.units.working_scale(),
        hole.trim(),
        shaft.trim(),
    )
    .map_err(|cause| cause.to_string())?;
    band.type_ = kind;
    Ok(band)
}

pub(super) fn model_label(
    seed: &GuiParameterEngineering,
    band: &Tolerance,
    selection: ModelValueType,
    precision: i64,
) -> Option<String> {
    let digits = if precision == 0 {
        seed.default_precision
    } else {
        u8::try_from(precision).ok()?
    };
    if digits > 8 {
        return None;
    }
    let nominal = seed.nominal;
    let value = Quantity::new(
        nominal.value + tolerance_offset(band, selection),
        nominal.dimension,
    );
    let formatted = seed.units.format_fixed_value(value, digits).ok()?;
    let unit = seed.units.preferred_name(value.dimension);
    Some(if unit.is_empty() {
        formatted
    } else {
        format!("{formatted} {unit}")
    })
}
