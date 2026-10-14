// SPDX-License-Identifier: GPL-2.0-only

use crate::parameters;
use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterState, vocabulary::*},
    quantity::Dimension,
    units::category_name,
};

pub(super) fn read(
    parameters: &ParameterCollection<u64>,
    request: ParameterNameArgs,
) -> Result<ParameterDetail, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    parameter_detail(parameters, id)
}

/// Build the canonical detail without JSON loss for an application-owned parameter identity.
/// Example: `parameter_detail(&snapshot.parameters, id)?` before publishing an internal GUI view.
pub fn parameter_detail(
    parameters: &ParameterCollection<u64>,
    id: u64,
) -> Result<ParameterDetail, Error> {
    let state = parameters::state(parameters, id)?;
    let readers = parameters.readers(id).map_err(|e| Error(e.to_string()))?;
    let drivers = parameters.drivers(id).map_err(|e| Error(e.to_string()))?;
    let mut detail = presentation(state)?;
    detail.parameter_info = parameters::information(parameters, id)?;
    detail.in_use = !readers.is_empty() || state.kind == MODEL_PARAM;
    detail.dependents = Some(parameters::names(parameters, &readers)?);
    detail.driven_by = Some(parameters::names(parameters, &drivers)?);
    numeric(parameters, id, state, &mut detail)?;
    Ok(detail)
}

fn presentation(state: &ParameterState<u64>) -> Result<ParameterDetail, Error> {
    Ok(ParameterDetail {
        comment: state.annotations.comment.clone(),
        is_key: state.annotations.is_key,
        visible: state.annotations.visible,
        precision: state.annotations.precision,
        exposed_as_property: state.annotations.exposed_as_property,
        built_in: state.built_in,
        renamed: state.renamed,
        display_format: display_name(state.annotations.display_format)
            .ok_or_else(|| {
                Error(format!(
                    "parameter display {:?}: expected a supported format",
                    state.annotations.display_format
                ))
            })?
            .into(),
        model_value_type: selection_name(state.model_value_type)
            .ok_or_else(|| {
                Error(format!(
                    "model selection {:?}: expected a supported selection",
                    state.model_value_type
                ))
            })?
            .into(),
        disabled_action_types: Some(action_names(state.disabled_actions)),
        custom_property_format: Some(std::boxed::Box::new(property_format(state)?)),
        expression_list: state
            .choices
            .expressions
            .as_ref()
            .filter(|v| !v.is_empty())
            .map(|_| std::boxed::Box::new(state.choices.clone())),
        ..Default::default()
    })
}

fn numeric(
    parameters: &ParameterCollection<u64>,
    id: u64,
    state: &ParameterState<u64>,
    detail: &mut ParameterDetail,
) -> Result<(), Error> {
    let quantity = parameters
        .inspection_value(id)
        .map_err(|e| Error(e.to_string()))?;
    let Some(quantity) = quantity else {
        detail.units = match state.last_good_value {
            Some(oblikovati_model::parameter::ParameterValue::Text(_)) => "text",
            _ => "boolean",
        }
        .into();
        return Ok(());
    };
    detail.units = unit_name(parameters, quantity.dimension);
    detail.model_value = quantity.value;
    detail.tolerance = Some(std::boxed::Box::new(tolerance_info(&state.tolerance)?));
    Ok(())
}

fn tolerance_info(tolerance: &Tolerance) -> Result<ToleranceInfo, Error> {
    // The stored zero band means the standard/default tolerance in the Go model.
    let kind = if tolerance.type_.0 == 0 {
        TOLERANCE_DEFAULT
    } else {
        tolerance.type_
    };
    let name = TOLERANCE_NAMES
        .iter()
        .find(|(_, candidate)| *candidate == kind)
        .ok_or_else(|| Error(format!("tolerance {kind:?}: expected a supported type")))?
        .0;
    Ok(ToleranceInfo {
        type_: name.into(),
        upper: tolerance.upper,
        lower: tolerance.lower,
        hole_tolerance: tolerance.hole_tolerance.clone(),
        shaft_tolerance: tolerance.shaft_tolerance.clone(),
    })
}

fn unit_name(parameters: &ParameterCollection<u64>, dimension: Dimension) -> String {
    let name = parameters.units().preferred_name(dimension);
    if name.is_empty() {
        category_name(dimension).into()
    } else {
        name.into()
    }
}

fn property_format(state: &ParameterState<u64>) -> Result<CustomPropertyFormatInfo, Error> {
    let property = &state.custom_property;
    let precision = PRECISION_NAMES
        .iter()
        .find(|(_, value)| *value == property.precision)
        .ok_or_else(|| {
            Error(format!(
                "property precision {:?}: expected a defined precision",
                property.precision
            ))
        })?
        .0;
    let property_type = match property.property_type {
        CUSTOM_PROPERTY_TEXT => "text",
        CUSTOM_PROPERTY_NUMBER => "number",
        value => {
            return Err(Error(format!(
                "property type {value:?}: expected text or number"
            )));
        }
    };
    Ok(CustomPropertyFormatInfo {
        property_type: property_type.into(),
        units: property.units.clone(),
        precision: precision.into(),
        show_leading_zeros: property.show_leading_zeros,
        show_trailing_zeros: property.show_trailing_zeros,
        show_units_string: property.show_units_string,
    })
}
