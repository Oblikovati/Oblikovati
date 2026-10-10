// SPDX-License-Identifier: GPL-2.0-only

use crate::units::document_units;
use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::{
    parameter::ParameterCollection, quantity::Dimension, units::UnitsOfMeasure,
};

pub(super) fn apply(
    parameters: &mut ParameterCollection<u64>,
    request: SetDocumentUnitsArgs,
) -> Result<DocumentUnitsInfo, Error> {
    let mut units = parameters.units().clone();
    apply_names(&mut units, &request)?;
    apply_display(&mut units, &request)?;
    parameters
        .set_unit_preferences(units)
        .map_err(|error| Error(error.to_string()))?;
    Ok(document_units(parameters.units()))
}

fn apply_names(units: &mut UnitsOfMeasure, request: &SetDocumentUnitsArgs) -> Result<(), Error> {
    for (dimension, name) in [
        (Dimension::Length, &request.length_unit),
        (Dimension::Angle, &request.angle_unit),
        (Dimension::Mass, &request.mass_unit),
        (Dimension::Time, &request.time_unit),
    ] {
        if let Some(name) = name {
            units
                .set_preferred(dimension, name)
                .map_err(|error| Error(error.to_string()))?;
        }
    }
    Ok(())
}

fn apply_display(units: &mut UnitsOfMeasure, request: &SetDocumentUnitsArgs) -> Result<(), Error> {
    let mut display = units.display_options();
    display.length_precision = precision(
        request.length_display_precision.as_deref(),
        display.length_precision,
    )?;
    display.angle_precision = precision(
        request.angle_display_precision.as_deref(),
        display.angle_precision,
    )?;
    if let Some(name) = &request.length_display_format {
        display.length_format = match name.as_str() {
            "decimal" => DISPLAY_FORMAT_DECIMAL,
            "fractional" => DISPLAY_FORMAT_FRACTIONAL,
            "architectural" => DISPLAY_FORMAT_ARCHITECTURAL,
            _ => {
                return Err(Error(format!(
                    "length display format {name:?}: expected decimal, fractional, or architectural"
                )));
            }
        };
    }
    units
        .set_display_options(display)
        .map_err(|error| Error(error.to_string()))
}

fn precision(requested: Option<&i64>, current: u8) -> Result<u8, Error> {
    requested.map_or(Ok(current), |value| {
        u8::try_from(*value).map_err(|_| {
            Error(format!(
                "display precision {value}: expected 0..255 supported by Rust document storage"
            ))
        })
    })
}
