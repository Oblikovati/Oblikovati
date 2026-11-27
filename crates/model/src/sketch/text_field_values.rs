// SPDX-License-Identifier: GPL-2.0-only
//! Bind authored field names to document identities before evaluating their values.
use super::{
    SketchError, SketchText2D, SketchTextFieldProjection, resolve_sketch_text_display,
    text_fields::project_sketch_text_fields,
};
use crate::parameter::ParameterCollection;
use oblikovati_api::types::{
    SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER, SketchTextField,
    SketchTextFraction, SketchTextStyle,
};
use std::borrow::Cow;

const STANDARD_FIELD_DECIMALS: u8 = 3;

/// Label of the linked file-name property inside the Sketch Text editor.
pub const SKETCH_TEXT_FILE_NAME_LABEL: &str = "FILENAME";

/// A document-local parameter identity and its authored field offset.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub struct SketchTextParameterBinding {
    pub at: u32,
    pub parameter_id: u64,
}

/// Bind a new source once, for example after inserting a field for `d0`.
/// The recipe retains the returned identity when that parameter is renamed.
pub fn bind_sketch_text_parameters(
    authored: &SketchTextStyle,
    parameters: &ParameterCollection<u64>,
) -> Result<Vec<SketchTextParameterBinding>, SketchError> {
    bind_text_parameter_changes(authored, None, parameters)
}

pub(crate) fn bind_text_parameter_changes(
    authored: &SketchTextStyle,
    existing: Option<&SketchText2D>,
    parameters: &ParameterCollection<u64>,
) -> Result<Vec<SketchTextParameterBinding>, SketchError> {
    let fields = authored.linked_fields.as_deref().unwrap_or_default();
    let fractions = authored.stacked_fractions.as_deref().unwrap_or_default();
    SketchTextFraction::validate(Some(&authored.content), fractions)
        .and_then(|()| SketchTextField::validate(Some(&authored.content), fields, fractions))
        .map_err(|error| SketchError::Definition(error.to_string()))?;
    fields
        .iter()
        .filter(|field| field.kind == SKETCH_TEXT_FIELD_MODEL_PARAMETER)
        .map(|field| {
            match existing.and_then(|text| retained_parameter_binding(text, field, parameters)) {
                Some(binding) => Ok(binding),
                None => bind_text_parameter(field, parameters),
            }
        })
        .collect()
}

fn retained_parameter_binding(
    existing: &SketchText2D,
    field: &SketchTextField,
    parameters: &ParameterCollection<u64>,
) -> Option<SketchTextParameterBinding> {
    let fields = existing.style.linked_fields.as_deref()?;
    let previous = fields.get(
        fields
            .binary_search_by_key(&field.at, |field| field.at)
            .ok()?,
    )?;
    let index = existing
        .parameter_bindings
        .binary_search_by_key(&field.at, |link| link.at)
        .ok()?;
    let binding = existing.parameter_bindings[index];
    let same_source = field.parameter == previous.parameter
        || parameters
            .get(binding.parameter_id)
            .is_some_and(|source| source.name == field.parameter);
    same_source.then_some(binding)
}

impl SketchText2D {
    /// Read the editable wire style using the current names of bound parameters.
    /// For example, a field authored as `d0` reads as `width` after a rename.
    pub fn current_text_style(&self, parameters: &ParameterCollection<u64>) -> SketchTextStyle {
        let mut style = self.style.clone();
        let fields = style
            .linked_fields
            .iter_mut()
            .flatten()
            .filter(|field| field.kind == SKETCH_TEXT_FIELD_MODEL_PARAMETER);
        for (field, link) in fields.zip(&self.parameter_bindings) {
            if let Some(parameter) = parameters.get(link.parameter_id) {
                field.parameter.clone_from(&parameter.name);
            }
        }
        style
    }
}

fn bind_text_parameter(
    field: &SketchTextField,
    parameters: &ParameterCollection<u64>,
) -> Result<SketchTextParameterBinding, SketchError> {
    let parameter_id = parameters.find(&field.parameter).ok_or_else(|| {
        SketchError::Definition(format!(
            "text field at {} parameter {:?}: expected a current document parameter",
            field.at, field.parameter
        ))
    })?;
    parameters
        .model_value(parameter_id)
        .map_err(field_value_error)?;
    Ok(SketchTextParameterBinding {
        at: field.at,
        parameter_id,
    })
}

/// Derive text for the current document state, for example with file name `"Part6.opd"`.
/// Document code supplies existing bindings so a reused parameter name cannot redirect a field.
pub fn resolve_document_sketch_text<'a>(
    authored: &'a SketchTextStyle,
    bindings: &[SketchTextParameterBinding],
    parameters: &ParameterCollection<u64>,
    file_name: &str,
) -> Result<Cow<'a, SketchTextStyle>, SketchError> {
    resolve_sketch_text_display(authored, |field| {
        if field.kind == SKETCH_TEXT_FIELD_FILE_NAME {
            return Ok(file_name.into());
        }
        evaluate_bound_text_parameter(field, bindings, parameters)
    })
}

pub(crate) fn resolve_document_sketch_text_editor(
    authored: &SketchTextStyle,
    existing: Option<&SketchText2D>,
    parameters: &ParameterCollection<u64>,
) -> Result<SketchTextFieldProjection, SketchError> {
    let bindings = bind_text_parameter_changes(authored, existing, parameters)?;
    project_sketch_text_fields(authored, |field| {
        if field.kind == SKETCH_TEXT_FIELD_FILE_NAME {
            return Ok(SKETCH_TEXT_FILE_NAME_LABEL.into());
        }
        evaluate_bound_text_parameter(field, &bindings, parameters)
    })
}

fn evaluate_bound_text_parameter(
    field: &SketchTextField,
    bindings: &[SketchTextParameterBinding],
    parameters: &ParameterCollection<u64>,
) -> Result<String, SketchError> {
    let index = bindings
        .binary_search_by_key(&field.at, |binding| binding.at)
        .map_err(|_| {
            SketchError::Definition(format!(
                "text field at {} parameter {:?}: expected an authored parameter identity",
                field.at, field.parameter
            ))
        })?;
    let amount = parameters
        .model_value(bindings[index].parameter_id)
        .map_err(field_value_error)?;
    let digits = field
        .precision
        .as_deref()
        .copied()
        .unwrap_or(STANDARD_FIELD_DECIMALS);
    parameters
        .units()
        .format_fixed_value(amount, digits)
        .map_err(field_value_error)
}

fn field_value_error(error: impl std::fmt::Display) -> SketchError {
    SketchError::Definition(format!("linked sketch text: {error}"))
}

#[cfg(test)]
#[path = "text_field_values_tests.rs"]
mod tests;
