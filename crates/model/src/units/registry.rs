// SPDX-License-Identifier: GPL-2.0-only

use super::UnitError;
use crate::quantity::Dimension::{self, *};

#[derive(Clone, Copy, Debug, PartialEq)]
pub(super) struct NamedUnit {
    pub name: &'static str,
    pub dimension: Dimension,
    pub factor: f64,
}

// Factors retain the Go baseline's centimetre/radian/kg/second arithmetic order.
const NAMED_UNITS: [(&str, Dimension, f64); 26] = [
    ("pm", Length, 1e-10),
    ("nm", Length, 1e-7),
    ("um", Length, 1e-4),
    ("µm", Length, 1e-4),
    ("mm", Length, 0.1),
    ("cm", Length, 1.0),
    ("m", Length, 100.0),
    ("km", Length, 100000.0),
    ("in", Length, 2.54),
    ("ft", Length, 30.48),
    ("rad", Angle, 1.0),
    ("deg", Angle, std::f64::consts::PI / 180.0),
    ("mm^2", Area, 0.01),
    ("cm^2", Area, 1.0),
    ("m^2", Area, 10000.0),
    ("mm^3", Volume, 0.001),
    ("cm^3", Volume, 1.0),
    ("m^3", Volume, 1e6),
    ("l", Volume, 1000.0),
    ("kg", Mass, 1.0),
    ("g", Mass, 0.001),
    ("lb", Mass, 0.45359237),
    ("s", Time, 1.0),
    ("ms", Time, 0.001),
    ("min", Time, 60.0),
    ("hr", Time, 3600.0),
];

pub(super) fn lookup(name: &str) -> Result<NamedUnit, UnitError> {
    NAMED_UNITS
        .iter()
        .find(|(spelling, _, _)| *spelling == name)
        .map(|&(name, dimension, factor)| NamedUnit {
            name,
            dimension,
            factor,
        })
        .ok_or_else(|| UnitError::UnknownUnit(name.to_owned()))
}

pub(super) fn require_category(name: &str, dimension: Dimension) -> Result<NamedUnit, UnitError> {
    let unit = lookup(name)?;
    if unit.dimension != dimension {
        return Err(UnitError::WrongCategory {
            name: name.to_owned(),
            expected: dimension,
        });
    }
    Ok(unit)
}
