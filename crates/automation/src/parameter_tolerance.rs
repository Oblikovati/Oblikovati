// SPDX-License-Identifier: GPL-2.0-only

use crate::{detail, parameters};
use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::{parameter::ParameterCollection, quantity::Dimension};

pub(super) fn apply(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterToleranceArgs,
) -> Result<ParameterDetail, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    match request.mode.as_str() {
        "symmetric" | "deviation" | "limits" => apply_band(parameters, id, &request)?,
        _ => apply_annotation(parameters, id, &request)?,
    }
    detail::read(parameters, ParameterNameArgs { name: request.name })
}

fn apply_band(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    request: &ParameterToleranceArgs,
) -> Result<(), Error> {
    let dimension = parameters
        .model_value(id)
        .map_err(|error| Error(error.to_string()))?
        .dimension;
    let upper = operand(parameters, dimension, &request.upper, "upper")?;
    if request.mode == "symmetric" {
        return parameters
            .set_symmetric(id, upper)
            .map_err(|error| Error(error.to_string()));
    }
    let lower = operand(parameters, dimension, &request.lower, "lower")?;
    let result = if request.mode == "limits" {
        parameters.set_limits(id, upper, lower)
    } else {
        parameters.set_deviation(id, upper, lower)
    };
    result.map_err(|error| Error(error.to_string()))
}

fn apply_annotation(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    request: &ParameterToleranceArgs,
) -> Result<(), Error> {
    let result = match request.mode.as_str() {
        "default" => parameters.set_tolerance_default(id),
        "min" => parameters.set_bandless(id, TOLERANCE_MIN),
        "max" => parameters.set_bandless(id, TOLERANCE_MAX),
        "basic" => parameters.set_bandless(id, TOLERANCE_BASIC),
        "reference" => parameters.set_bandless(id, TOLERANCE_REFERENCE),
        "fits" => parameters.set_fits(id, &request.hole, &request.shaft),
        mode => {
            return Err(Error(format!(
                "tolerance mode {mode:?}: expected default, deviation, symmetric, limits, min, max, fits, basic, or reference"
            )));
        }
    };
    result.map_err(|error| Error(error.to_string()))
}

fn operand(
    parameters: &ParameterCollection<u64>,
    dimension: Dimension,
    source: &str,
    field: &str,
) -> Result<f64, Error> {
    // Match the Go resolver: only a same-dimension expression bypasses literal parsing.
    if let Ok(quantity) = parameters.evaluate_expression(source)
        && quantity.dimension == dimension
    {
        return Ok(quantity.value);
    }
    let quantity = parameters
        .units()
        .parse(source, dimension)
        .map_err(|error| {
            Error(format!(
                "tolerance {field} {source:?}: expected a {dimension:?} quantity: {error}"
            ))
        })?;
    if quantity.dimension != dimension {
        return Err(Error(format!(
            "tolerance {field} {source:?}: expected {dimension:?}, found {:?}",
            quantity.dimension
        )));
    }
    Ok(quantity.value)
}
