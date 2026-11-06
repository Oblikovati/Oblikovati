// SPDX-License-Identifier: GPL-2.0-only

use crate::{detail, parameters};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::parameter::ParameterCollection;

pub(super) fn replace(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterExpressionListArgs,
) -> Result<ParameterDetail, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    let choices = request.expressions.unwrap_or_default();
    parameters
        .set_expression_list(id, choices, request.allow_custom_values)
        .map_err(|error| Error(error.to_string()))?;
    parameters
        .set_custom_order(id, request.custom_order)
        .map_err(|error| Error(error.to_string()))?;
    detail::read(parameters, ParameterNameArgs { name: request.name })
}

pub(super) fn sweep(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterSweepArgs,
) -> Result<ParameterSweepResult, Error> {
    let selection = crate::parameter_update::parse_selection(&request.model_value_type)?;
    let count = parameters
        .set_all_model_value_type(selection)
        .map_err(|error| Error(error.to_string()))?;
    let affected = i64::try_from(count).map_err(|_| {
        Error(format!(
            "affected count {count}: expected a signed 64-bit count"
        ))
    })?;
    Ok(ParameterSweepResult { affected })
}
