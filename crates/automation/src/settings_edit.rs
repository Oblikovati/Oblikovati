// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, wire::ParameterSettingsUpdateArgs};
use oblikovati_model::parameter::{ParameterCollection, parse_dimension_display};

pub(super) fn apply(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterSettingsUpdateArgs,
) -> Result<(), Error> {
    let mut settings = parameters.settings().clone();
    assign(
        &mut settings.linear_standard_tolerance,
        request.linear_standard_tolerance,
    );
    assign(
        &mut settings.angular_standard_tolerance,
        request.angular_standard_tolerance,
    );
    assign(
        &mut settings.use_standard_tolerances,
        request.use_standard_tolerances,
    );
    assign(
        &mut settings.export_standard_tolerances,
        request.export_standard_tolerances,
    );
    assign(
        &mut settings.display_parameter_as_expression,
        request.display_parameter_as_expression,
    );
    assign(
        &mut settings.linear_dimension_precision,
        request.linear_dimension_precision,
    );
    assign(
        &mut settings.angular_dimension_precision,
        request.angular_dimension_precision,
    );
    if let Some(name) = request.dimension_display_type {
        settings.dimension_display_type = parse_dimension_display(&name).ok_or_else(|| Error(format!(
            "dimension display {name:?}: expected value, name, expression, tolerance, or preciseValue")))?;
    }
    parameters
        .set_settings(settings)
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn assign<T>(target: &mut T, value: Option<Box<T>>) {
    if let Some(value) = value {
        *target = *value;
    }
}
