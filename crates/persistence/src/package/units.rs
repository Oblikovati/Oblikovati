// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_api::types::*;
use oblikovati_model::{
    quantity::Dimension,
    units::{AngleFormat, UnitsOfMeasure, category_name},
};
use std::collections::BTreeMap;
mod capture;
pub(super) use capture::capture_units;

const CATEGORIES: [Dimension; 6] = [
    Dimension::Length,
    Dimension::Angle,
    Dimension::Area,
    Dimension::Volume,
    Dimension::Mass,
    Dimension::Time,
];

impl DocumentPackage {
    /// Decode stored units before any expression or geometry is reconstructed.
    /// Example: `let units = package.document_units()?;`.
    pub fn document_units(&self) -> Result<UnitsOfMeasure, PackageCodecError> {
        let section = self
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .and_then(|model| model.get("units"));
        let records: BTreeMap<String, String> = match section {
            None | Some(Value::Null) => return Ok(UnitsOfMeasure::default()),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        let mut units = UnitsOfMeasure::default();
        for (key, value) in records {
            apply_unit(&mut units, &key, &value)?;
        }
        Ok(units)
    }
}

fn apply_unit(units: &mut UnitsOfMeasure, key: &str, value: &str) -> Result<(), PackageCodecError> {
    if let Some(dimension) = CATEGORIES
        .iter()
        .find(|dimension| category_name(**dimension) == key)
    {
        return units.set_preferred(*dimension, value).map_err(unit_error);
    }
    if key == "workingScale" {
        let scale = value
            .parse::<f64>()
            .map_err(|_| invalid_unit(key, value, "a finite positive working scale"))?;
        *units = units
            .clone()
            .with_working_scale(scale)
            .map_err(unit_error)?;
        return Ok(());
    }
    apply_display(units, key, value)
}

fn apply_display(
    units: &mut UnitsOfMeasure,
    key: &str,
    value: &str,
) -> Result<(), PackageCodecError> {
    let mut display = units.display_options();
    match key {
        "lengthPrecision" => display.length_precision = precision(key, value)?,
        "anglePrecision" => display.angle_precision = precision(key, value)?,
        "lengthFormat" => display.length_format = length_format(value)?,
        "angleFormat" => display.angle_format = angle_format(value)?,
        _ => {
            return Err(invalid_unit(
                key,
                value,
                "a supported quantity category or unit display/scale setting",
            ));
        }
    }
    units.set_display_options(display).map_err(unit_error)
}

fn precision(key: &str, value: &str) -> Result<i64, PackageCodecError> {
    value
        .parse()
        .map_err(|_| invalid_unit(key, value, "a nonnegative signed 64-bit display precision"))
}

fn length_format(value: &str) -> Result<ParameterDisplayFormat, PackageCodecError> {
    match value {
        "decimal" => Ok(DISPLAY_FORMAT_DECIMAL),
        "fractional" => Ok(DISPLAY_FORMAT_FRACTIONAL),
        "architectural" => Ok(DISPLAY_FORMAT_ARCHITECTURAL),
        _ => Err(invalid_unit(
            "lengthFormat",
            value,
            "decimal, fractional, or architectural",
        )),
    }
}

fn angle_format(value: &str) -> Result<AngleFormat, PackageCodecError> {
    match value {
        "decimal" => Ok(AngleFormat::Decimal),
        "dms" => Ok(AngleFormat::DegreesMinutesSeconds),
        _ => Err(invalid_unit("angleFormat", value, "decimal or dms")),
    }
}

fn invalid_unit(key: &str, value: &str, expected: &str) -> PackageCodecError {
    PackageCodecError(format!(
        "unit setting {key:?}={value:?}: expected {expected}"
    ))
}

fn unit_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("document units: {error}"))
}

pub(super) fn require_preserved_scale(
    package: &DocumentPackage,
    units: &UnitsOfMeasure,
) -> Result<(), PackageCodecError> {
    let stored = package.document_units()?;
    let existing = package
        .root
        .get("model")
        .and_then(Value::as_mapping)
        .is_some_and(|model| !model.is_empty());
    if existing && stored.working_scale() != units.working_scale() {
        return Err(PackageCodecError(format!(
            "working scale {}: expected stored scale {} while retaining existing model sections",
            units.working_scale(),
            stored.working_scale()
        )));
    }
    Ok(())
}
