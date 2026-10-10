// SPDX-License-Identifier: GPL-2.0-only

use super::{UnitError, registry};
use crate::quantity::Dimension;

/// Resolve a registered unit's category; for example `unit_category("deg")` gives `Angle`.
pub fn unit_category(name: &str) -> Result<Dimension, UnitError> {
    registry::lookup(name).map(|unit| unit.dimension)
}

/// Convert physical units without document preferences; for example `convert_value(1.0, "in", "cm")`.
pub fn convert_value(value: f64, from: &str, to: &str) -> Result<f64, UnitError> {
    let reject = |expected| conversion_error(value, from, to, expected);
    let source = registry::lookup(from)?;
    let target = registry::lookup(to)?;
    if source.dimension != target.dimension {
        return Err(reject("units in the same dimension category"));
    }
    let converted = value * source.factor / target.factor;
    if !value.is_finite() || !converted.is_finite() {
        return Err(reject("finite input and a finite converted value"));
    }
    Ok(converted)
}

/// Resolve a diagnostic category name; for example `parse_category("length")` gives `Some(Length)`.
pub fn parse_category(name: &str) -> Option<Dimension> {
    CATEGORY_NAMES
        .iter()
        .find(|(spelling, _)| *spelling == name)
        .map(|(_, dimension)| *dimension)
}

/// Return the stable quantity category spelling used by document recipes.
/// Example: `assert_eq!(category_name(Dimension::Length), "length");`.
pub fn category_name(dimension: Dimension) -> &'static str {
    CATEGORY_NAMES
        .iter()
        .find(|(_, category)| *category == dimension)
        .expect("every dimension has a category name")
        .0
}

const CATEGORY_NAMES: [(&str, Dimension); 9] = [
    ("unitless", Dimension::Unitless),
    ("length", Dimension::Length),
    ("angle", Dimension::Angle),
    ("area", Dimension::Area),
    ("volume", Dimension::Volume),
    ("mass", Dimension::Mass),
    ("time", Dimension::Time),
    ("boolean", Dimension::Boolean),
    ("text", Dimension::Text),
];

fn conversion_error(value: f64, from: &str, to: &str, expected: &'static str) -> UnitError {
    UnitError::Conversion {
        value,
        from: from.into(),
        to: to.into(),
        expected,
    }
}
