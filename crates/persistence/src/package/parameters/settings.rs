// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, Value, codec_error, parameter_error};
use oblikovati_api::types::{
    DIMENSION_DISPLAY_EXPRESSION, DIMENSION_DISPLAY_NAME, DIMENSION_DISPLAY_PRECISE_VALUE,
    DIMENSION_DISPLAY_TOLERANCE, DIMENSION_DISPLAY_VALUE, DimensionDisplayType,
};
use oblikovati_model::parameter::{CollectionSettings, ParameterCollection};
use serde::{Deserialize, Serialize};

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
struct SettingsRecipe {
    linear_standard_tolerance: String,
    angular_standard_tolerance: String,
    use_standard_tolerances: bool,
    export_standard_tolerances: bool,
    linear_dimension_precision: i64,
    angular_dimension_precision: i64,
    dimension_display_type: String,
    display_parameter_as_expression: bool,
}

pub(super) fn restore_settings(
    model: Option<&Value>,
    parameters: &mut ParameterCollection<u64>,
) -> Result<(), PackageCodecError> {
    let section = model
        .and_then(Value::as_mapping)
        .and_then(|model| model.get("parameterSettings"));
    let record: SettingsRecipe = match section {
        None | Some(Value::Null) => return Ok(()),
        Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
    };
    parameters
        .set_settings(record.into_settings()?)
        .map_err(parameter_error)
}

impl SettingsRecipe {
    fn into_settings(self) -> Result<CollectionSettings, PackageCodecError> {
        Ok(CollectionSettings {
            linear_standard_tolerance: self.linear_standard_tolerance,
            angular_standard_tolerance: self.angular_standard_tolerance,
            use_standard_tolerances: self.use_standard_tolerances,
            export_standard_tolerances: self.export_standard_tolerances,
            linear_dimension_precision: self.linear_dimension_precision,
            angular_dimension_precision: self.angular_dimension_precision,
            dimension_display_type: display_mode(&self.dimension_display_type)?,
            display_parameter_as_expression: self.display_parameter_as_expression,
        })
    }
}

fn display_mode(name: &str) -> Result<DimensionDisplayType, PackageCodecError> {
    match name {
        "value" => Ok(DIMENSION_DISPLAY_VALUE),
        "name" => Ok(DIMENSION_DISPLAY_NAME),
        "expression" => Ok(DIMENSION_DISPLAY_EXPRESSION),
        "tolerance" => Ok(DIMENSION_DISPLAY_TOLERANCE),
        "preciseValue" => Ok(DIMENSION_DISPLAY_PRECISE_VALUE),
        _ => Err(PackageCodecError(format!(
            "dimension display mode {name:?}: expected value, name, expression, tolerance, or preciseValue"
        ))),
    }
}

pub(super) fn capture_settings(settings: &CollectionSettings) -> Result<Value, PackageCodecError> {
    let mode = match settings.dimension_display_type {
        DIMENSION_DISPLAY_VALUE => "value",
        DIMENSION_DISPLAY_NAME => "name",
        DIMENSION_DISPLAY_EXPRESSION => "expression",
        DIMENSION_DISPLAY_TOLERANCE => "tolerance",
        DIMENSION_DISPLAY_PRECISE_VALUE => "preciseValue",
        found => {
            return Err(PackageCodecError(format!(
                "dimension display {found:?}: expected a known API display mode"
            )));
        }
    };
    serde_yaml_ng::to_value(SettingsRecipe {
        linear_standard_tolerance: settings.linear_standard_tolerance.clone(),
        angular_standard_tolerance: settings.angular_standard_tolerance.clone(),
        use_standard_tolerances: settings.use_standard_tolerances,
        export_standard_tolerances: settings.export_standard_tolerances,
        linear_dimension_precision: settings.linear_dimension_precision,
        angular_dimension_precision: settings.angular_dimension_precision,
        dimension_display_type: mode.into(),
        display_parameter_as_expression: settings.display_parameter_as_expression,
    })
    .map_err(codec_error)
}
