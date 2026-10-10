// SPDX-License-Identifier: GPL-2.0-only

use super::property::PropertyRecipe;
use super::{PackageCodecError, parameter_error};
use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{ParameterAnnotations, ParameterCollection},
    quantity::Quantity,
    units::parse_category,
};
use serde::{Deserialize, Serialize};

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub(super) struct ParameterRecipe {
    pub(super) name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(super) last_good: Option<super::diagnostics::NumericDiagnostic>,
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
    custom_property: Option<PropertyRecipe>,
    pub(super) tolerance: Option<super::tolerance::ToleranceRecipe>,
    pub(super) model_value_type: String,
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
        if let Some(property) = &self.custom_property {
            parameters
                .set_custom_property_format(id, property.restore()?)
                .map_err(parameter_error)?;
        }
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
                match super::measured::parse_saved_constant(&self.expression)? {
                    Some(value) => parameters.insert_value(id, &self.name, kind, value),
                    None => parameters.insert_expression(id, &self.name, kind, &self.expression),
                }
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

impl ParameterRecipe {
    pub(super) fn capture(
        state: &oblikovati_model::parameter::ParameterState<u64>,
    ) -> Result<Self, PackageCodecError> {
        let mut record = Self {
            name: state.name.clone(),
            kind: super::export::kind_name(state.kind)?.into(),
            comment: state.annotations.comment.clone(),
            key: state.annotations.is_key,
            export: state.annotations.exposed_as_property,
            precision: state.annotations.precision,
            hidden: !state.annotations.visible,
            renamed: state.renamed,
            display_format: super::export::display_name(state.annotations.display_format)?.into(),
            custom_property: Some(PropertyRecipe::capture(&state.custom_property)?),
            ..Self::default()
        };
        record.last_good = super::diagnostics::NumericDiagnostic::capture(state)?;
        record.capture_value(state)?;
        record.capture_behavior(state)?;
        Ok(record)
    }

    fn capture_value(
        &mut self,
        state: &oblikovati_model::parameter::ParameterState<u64>,
    ) -> Result<(), PackageCodecError> {
        use oblikovati_model::parameter::ParameterValue;
        if state.current().is_none() && matches!(state.kind, USER_PARAM | MODEL_PARAM | TABLE_PARAM)
        {
            self.expression = state.source.clone();
            return Ok(());
        }
        match state.current().or(state.last_good_value.as_ref()) {
            Some(ParameterValue::Text(text)) => {
                self.value_type = "text".into();
                self.text = text.clone();
            }
            Some(ParameterValue::Boolean(value)) => {
                self.value_type = "boolean".into();
                self.boolean = *value;
            }
            Some(ParameterValue::Numeric(value))
                if matches!(state.kind, REFERENCE_PARAM | DERIVED_PARAM) =>
            {
                self.value = value.value;
                self.unit = oblikovati_model::units::category_name(value.dimension).into();
            }
            Some(ParameterValue::Numeric(_)) => self.expression = state.source.clone(),
            None => {
                return Err(PackageCodecError(format!(
                    "parameter {:?}: expected a healthy value for recipe export",
                    state.name
                )));
            }
        }
        Ok(())
    }

    fn capture_behavior(
        &mut self,
        state: &oblikovati_model::parameter::ParameterState<u64>,
    ) -> Result<(), PackageCodecError> {
        if state.tolerance != Tolerance::default() {
            self.tolerance = Some(super::tolerance::ToleranceRecipe::capture(
                &state.tolerance,
            )?);
        }
        self.model_value_type = super::export::selection_name(state.model_value_type)?.into();
        self.disabled_actions =
            oblikovati_model::parameter::vocabulary::action_names(state.disabled_actions);
        if let Some(choices) = &state.choices.expressions {
            self.expression_list = choices.clone();
            self.allow_custom_value = state.choices.allow_custom_values;
            self.sorted_value_list = !state.choices.custom_order;
        }
        Ok(())
    }
}
