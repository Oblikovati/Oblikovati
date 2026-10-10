// SPDX-License-Identifier: GPL-2.0-only

use crate::parameters;
use oblikovati_api::{
    Error,
    types::{MODEL_PARAM, USER_PARAM},
    wire::*,
};
use oblikovati_model::{
    history::RecipeReservations, parameter::ParameterCollection, quantity::Dimension,
};

pub(super) fn add(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut RecipeReservations<'_>,
    request: ParameterSetArgs,
) -> Result<ParameterInfo, Error> {
    let id = reservations
        .allocate()
        .map_err(|error| Error(error.to_string()))?;
    let result = match request.value_type.trim().to_lowercase().as_str() {
        "" | "numeric" => {
            require_expression(&request)?;
            let kind = if request.kind.trim().eq_ignore_ascii_case("model") {
                MODEL_PARAM
            } else {
                USER_PARAM
            };
            parameters.insert_expression(id, &request.name, kind, &request.expression)
        }
        "text" => parameters.insert_text(id, &request.name, &request.expression),
        "boolean" | "bool" => {
            parameters.insert_boolean(id, &request.name, boolean_literal(&request.expression)?)
        }
        _ => {
            return Err(Error(format!(
                "valueType {:?}: expected numeric, text, or boolean",
                request.value_type
            )));
        }
    };
    result.map_err(|error| Error(error.to_string()))?;
    parameters::information(parameters, id)
}

pub(super) fn set(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterSetArgs,
) -> Result<ParameterInfo, Error> {
    require_expression(&request)?;
    let id = parameters::resolve(parameters, &request.name)?;
    let dimension = match &parameters::state(parameters, id)?.last_good_value {
        Some(oblikovati_model::parameter::ParameterValue::Numeric(quantity)) => quantity.dimension,
        _ => Dimension::Unitless,
    };
    parameters
        .set_authored_expression(id, &request.expression, dimension)
        .map_err(|error| Error(error.to_string()))?;
    parameters::information(parameters, id)
}

fn require_expression(request: &ParameterSetArgs) -> Result<(), Error> {
    if request.expression.is_empty() {
        return Err(Error(format!(
            "parameter {:?} expression {:?}: expected a nonempty numeric expression",
            request.name, request.expression
        )));
    }
    Ok(())
}

fn boolean_literal(source: &str) -> Result<bool, Error> {
    // Preserve strconv.ParseBool spellings, including its numeric and single-letter forms.
    match source.trim() {
        "1" | "t" | "T" | "TRUE" | "true" | "True" => Ok(true),
        "0" | "f" | "F" | "FALSE" | "false" | "False" => Ok(false),
        _ => Err(Error(format!(
            "boolean expression {source:?}: expected a Go-compatible true or false literal"
        ))),
    }
}
