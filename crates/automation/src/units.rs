// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::{
    quantity::{Dimension, Quantity},
    units::{UnitsOfMeasure, category_name, convert_value, parse_category, unit_category},
};

pub(super) fn document_units(units: &UnitsOfMeasure) -> DocumentUnitsInfo {
    let display = units.display_options();
    DocumentUnitsInfo {
        length_unit: units.preferred_name(Dimension::Length).into(),
        angle_unit: units.preferred_name(Dimension::Angle).into(),
        mass_unit: units.preferred_name(Dimension::Mass).into(),
        time_unit: units.preferred_name(Dimension::Time).into(),
        length_display_precision: display.length_precision,
        angle_display_precision: display.angle_precision,
        length_display_format: display_name(display.length_format).into(),
        working_scale_cm: units.working_scale(),
    }
}

fn display_name(format: ParameterDisplayFormat) -> &'static str {
    match format {
        DISPLAY_FORMAT_FRACTIONAL => "fractional",
        DISPLAY_FORMAT_ARCHITECTURAL => "architectural",
        _ => "decimal", // UnitsOfMeasure refuses other formats at construction/update.
    }
}

pub(super) fn convert(request: ConvertUnitsArgs) -> Result<ConvertUnitsResult, Error> {
    convert_value(request.value, &request.from, &request.to)
        .map(|value| ConvertUnitsResult { value })
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn category(request: UnitStringArgs) -> Result<UnitsTypeResult, Error> {
    unit_category(&request.unit_string)
        .map(|dimension| UnitsTypeResult {
            units_type: category_name(dimension).into(),
        })
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn preferred(
    units: &UnitsOfMeasure,
    request: UnitsTypeArgs,
) -> Result<StringResult, Error> {
    let dimension = require_category(&request.units_type)?;
    Ok(StringResult {
        value: units.preferred_name(dimension).into(),
    })
}

pub(super) fn formatted(
    units: &UnitsOfMeasure,
    request: StringFromValueArgs,
    precise: bool,
) -> Result<StringResult, Error> {
    let quantity = Quantity::new(request.value, require_category(&request.units_type)?);
    let formatted = if precise {
        units.format_precise(quantity)
    } else {
        units.format_display(quantity)
    };
    formatted
        .map(|value| StringResult { value })
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn require_category(name: &str) -> Result<Dimension, Error> {
    parse_category(name).ok_or_else(|| {
        Error(format!(
            "unit category {name:?}: expected a supported quantity category"
        ))
    })
}
