// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, parameter_error};
use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{ParameterAnnotations, ParameterCollection},
    quantity::Quantity,
    units::parse_category,
};
use serde::Deserialize;

#[derive(Default, Deserialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub(super) struct ParameterRecipe {
    name: String,
    kind: String,
    value_type: String,
    expression: String,
    text: String,
    #[serde(rename = "bool")]
    boolean: bool,
    value: f64,
    unit: String,
    comment: String,
    key: bool,
    export: bool,
    precision: i64,
    hidden: bool,
    display_format: String,
    renamed: bool,
    disabled_actions: Vec<String>,
    expression_list: Vec<String>,
    allow_custom_value: bool,
    sorted_value_list: bool,
}

impl ParameterRecipe {
    pub(super) fn restore(
        &self,
        id: u64,
        parameters: &mut ParameterCollection<u64>,
    ) -> Result<(), PackageCodecError> {
        self.validate_value_fields()?;
        self.restore_value(id, parameters)?;
        self.restore_annotations(id, parameters)?;
        parameters
            .set_renamed(id, self.renamed)
            .map_err(parameter_error)?;
        parameters
            .set_disabled_action_types(id, action_mask(&self.disabled_actions)?)
            .map_err(parameter_error)?;
        if !self.expression_list.is_empty() {
            parameters
                .set_expression_list(id, self.expression_list.clone(), self.allow_custom_value)
                .map_err(parameter_error)?;
            parameters
                .set_custom_order(id, !self.sorted_value_list)
                .map_err(parameter_error)?;
        }
        Ok(())
    }

    fn validate_value_fields(&self) -> Result<(), PackageCodecError> {
        let literals_clear = self.text.is_empty() && !self.boolean;
        let measured_clear = self.value == 0.0 && self.unit.is_empty();
        let valid = match self.value_type.as_str() {
            "text" => !self.boolean && measured_clear && self.expression.is_empty(),
            "boolean" => self.text.is_empty() && measured_clear && self.expression.is_empty(),
            "" if matches!(self.kind.as_str(), "reference" | "derived") => {
                literals_clear && self.expression.is_empty()
            }
            "" => literals_clear && measured_clear,
            _ => false,
        };
        if !valid
            || (self.expression_list.is_empty()
                && (self.allow_custom_value || self.sorted_value_list))
        {
            return Err(PackageCodecError(format!(
                "parameter {:?}: expected value fields consistent with kind {:?}, flavor {:?}, and populated choices for list flags",
                self.name, self.kind, self.value_type
            )));
        }
        Ok(())
    }

    fn restore_value(
        &self,
        id: u64,
        parameters: &mut ParameterCollection<u64>,
    ) -> Result<(), PackageCodecError> {
        let kind = parameter_kind(&self.kind)?;
        let result = match self.value_type.as_str() {
            "text" if kind == USER_PARAM => parameters.insert_text(id, &self.name, &self.text),
            "boolean" if kind == USER_PARAM => {
                parameters.insert_boolean(id, &self.name, self.boolean)
            }
            "" if matches!(kind, USER_PARAM | MODEL_PARAM | TABLE_PARAM) => {
                parameters.insert_expression(id, &self.name, kind, &self.expression)
            }
            "" => parameters.insert_value(id, &self.name, kind, self.measured_value()?),
            value_type => {
                return Err(PackageCodecError(format!(
                    "parameter {:?} kind {:?}, valueType {value_type:?}: expected numeric or a text/boolean user parameter",
                    self.name, self.kind
                )));
            }
        };
        result.map_err(parameter_error)
    }

    fn measured_value(&self) -> Result<Quantity, PackageCodecError> {
        let dimension = parse_category(&self.unit).ok_or_else(|| {
            PackageCodecError(format!(
                "parameter {:?} unit {:?}: expected a known measured quantity category",
                self.name, self.unit
            ))
        })?;
        Ok(Quantity::new(self.value, dimension))
    }

    fn restore_annotations(
        &self,
        id: u64,
        parameters: &mut ParameterCollection<u64>,
    ) -> Result<(), PackageCodecError> {
        let annotations = ParameterAnnotations {
            comment: self.comment.clone(),
            is_key: self.key,
            exposed_as_property: self.export,
            precision: self.precision,
            visible: !self.hidden,
            display_format: display_format(&self.display_format)?,
        };
        parameters
            .set_annotations(id, annotations)
            .map_err(parameter_error)
    }
}

fn parameter_kind(name: &str) -> Result<ParameterKind, PackageCodecError> {
    match name {
        "user" => Ok(USER_PARAM),
        "model" => Ok(MODEL_PARAM),
        "table" => Ok(TABLE_PARAM),
        "reference" => Ok(REFERENCE_PARAM),
        "derived" => Ok(DERIVED_PARAM),
        _ => Err(PackageCodecError(format!(
            "parameter kind {name:?}: expected user, model, table, reference, or derived"
        ))),
    }
}

fn display_format(name: &str) -> Result<ParameterDisplayFormat, PackageCodecError> {
    match name {
        "" | "decimal" => Ok(DISPLAY_FORMAT_DECIMAL),
        "fractional" => Ok(DISPLAY_FORMAT_FRACTIONAL),
        "architectural" => Ok(DISPLAY_FORMAT_ARCHITECTURAL),
        _ => Err(PackageCodecError(format!(
            "display format {name:?}: expected decimal, fractional, or architectural"
        ))),
    }
}

fn action_mask(names: &[String]) -> Result<ActionType, PackageCodecError> {
    let mut mask = ACTION_NONE;
    for name in names {
        mask.0 |= match name.as_str() {
            "edit" => ACTION_EDIT.0,
            "rename" => ACTION_RENAME.0,
            "delete" => ACTION_DELETE.0,
            _ => {
                return Err(PackageCodecError(format!(
                    "disabled action {name:?}: expected edit, rename, or delete"
                )));
            }
        };
    }
    Ok(mask)
}
