// SPDX-License-Identifier: GPL-2.0-only

use crate::{detail, parameters, settings_edit::assign};
use oblikovati_api::{Error, types::*, wire::*};
use oblikovati_model::parameter::{CustomPropertyFormat, ParameterCollection, vocabulary::*};

pub(super) fn apply(
    parameters: &mut ParameterCollection<u64>,
    request: ParameterUpdateArgs,
) -> Result<ParameterDetail, Error> {
    let id = parameters::resolve(parameters, &request.name)?;
    update_annotations(parameters, id, &request)?;
    update_enums(parameters, id, &request)?;
    if let Some(format) = request.custom_property_format {
        parameters
            .set_custom_property_format(id, property_format(*format)?)
            .map_err(|error| Error(error.to_string()))?;
    }
    detail::read(parameters, ParameterNameArgs { name: request.name })
}

fn update_annotations(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    request: &ParameterUpdateArgs,
) -> Result<(), Error> {
    let mut annotations = parameters::state(parameters, id)?.annotations.clone();
    assign(&mut annotations.comment, request.comment.clone());
    assign(&mut annotations.is_key, request.is_key.clone());
    assign(&mut annotations.visible, request.visible.clone());
    assign(&mut annotations.precision, request.precision.clone());
    assign(
        &mut annotations.exposed_as_property,
        request.exposed_as_property.clone(),
    );
    if let Some(name) = &request.display_format {
        annotations.display_format = [
            DISPLAY_FORMAT_DECIMAL,
            DISPLAY_FORMAT_FRACTIONAL,
            DISPLAY_FORMAT_ARCHITECTURAL,
        ]
        .into_iter()
        .find(|value| display_name(*value) == Some(name.as_str()))
        .ok_or_else(|| {
            Error(format!(
                "display format {name:?}: expected decimal, fractional, or architectural"
            ))
        })?;
    }
    parameters
        .set_annotations(id, annotations)
        .map_err(|error| Error(error.to_string()))
}

fn update_enums(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    request: &ParameterUpdateArgs,
) -> Result<(), Error> {
    if let Some(name) = &request.model_value_type {
        parameters
            .set_model_value_type(id, parse_selection(name)?)
            .map_err(|error| Error(error.to_string()))?;
    }
    if let Some(names) = &request.disabled_action_types {
        parameters
            .set_disabled_action_types(id, parse_actions(names.as_deref().unwrap_or_default())?)
            .map_err(|error| Error(error.to_string()))?;
    }
    Ok(())
}

pub(super) fn parse_selection(name: &str) -> Result<ModelValueType, Error> {
    [
        MODEL_VALUE_NOMINAL,
        MODEL_VALUE_LOWER,
        MODEL_VALUE_UPPER,
        MODEL_VALUE_MEDIAN,
    ]
    .into_iter()
    .find(|value| selection_name(*value) == Some(name))
    .ok_or_else(|| {
        Error(format!(
            "modelValueType {name:?}: expected nominal, lower, upper, or median"
        ))
    })
}

fn parse_actions(names: &[String]) -> Result<ActionType, Error> {
    names.iter().try_fold(ActionType(0), |mask, name| {
        let flag = [ACTION_EDIT, ACTION_RENAME, ACTION_DELETE]
            .into_iter()
            .find(|flag| action_names(*flag).first() == Some(name))
            .ok_or_else(|| {
                Error(format!(
                    "disabled action {name:?}: expected edit, rename, or delete"
                ))
            })?;
        Ok(ActionType(mask.0 | flag.0))
    })
}

fn property_format(request: CustomPropertyFormatInfo) -> Result<CustomPropertyFormat, Error> {
    let property_type = match request.property_type.as_str() {
        "text" => CUSTOM_PROPERTY_TEXT,
        "number" => CUSTOM_PROPERTY_NUMBER,
        name => {
            return Err(Error(format!(
                "custom property type {name:?}: expected text or number"
            )));
        }
    };
    let precision = PRECISION_NAMES.iter().find(|(name, _)| *name == request.precision)
        .map(|(_, precision)| *precision)
        .ok_or_else(|| Error(format!("custom property precision {:?}: expected a canonical precision from {PRECISION_NAMES:?}", request.precision)))?;
    Ok(CustomPropertyFormat {
        property_type,
        precision,
        units: request.units,
        show_leading_zeros: request.show_leading_zeros,
        show_trailing_zeros: request.show_trailing_zeros,
        show_units_string: request.show_units_string,
    })
}
