// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{
    DIMENSION_DISPLAY_EXPRESSION, DIMENSION_DISPLAY_NAME, DIMENSION_DISPLAY_PRECISE_VALUE,
    DIMENSION_DISPLAY_TOLERANCE, DIMENSION_DISPLAY_VALUE, DimensionDisplayType,
};

/// Document presentation/defaulting preferences; this is not a wire or storage encoding.
#[derive(Clone, Debug, PartialEq)]
pub struct CollectionSettings {
    pub linear_standard_tolerance: String,
    pub angular_standard_tolerance: String,
    pub use_standard_tolerances: bool,
    pub export_standard_tolerances: bool,
    pub linear_dimension_precision: i64,
    pub angular_dimension_precision: i64,
    pub dimension_display_type: DimensionDisplayType,
    pub display_parameter_as_expression: bool,
}

impl Default for CollectionSettings {
    fn default() -> Self {
        Self {
            linear_standard_tolerance: String::new(),
            angular_standard_tolerance: String::new(),
            use_standard_tolerances: false,
            export_standard_tolerances: false,
            linear_dimension_precision: 3,
            angular_dimension_precision: 2,
            dimension_display_type: DIMENSION_DISPLAY_VALUE,
            display_parameter_as_expression: false,
        }
    }
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Read document settings; for example `parameters.settings().linear_dimension_precision`.
    pub fn settings(&self) -> &CollectionSettings {
        &self.settings
    }

    /// Replace validated settings atomically; for example `parameters.set_settings(settings)`.
    /// Standard-tolerance strings remain metadata; this edit does not apply tolerance bands.
    pub fn set_settings(&mut self, settings: CollectionSettings) -> Result<(), ParameterError<Id>> {
        validate_precision(settings.linear_dimension_precision)?;
        validate_precision(settings.angular_dimension_precision)?;
        validate_display_mode(settings.dimension_display_type)?;
        self.settings = settings;
        Ok(())
    }
}

fn validate_precision<Id>(precision: i64) -> Result<(), ParameterError<Id>> {
    // #3582: move the Go router's 0–8 rule to the collection so every caller shares it.
    if !(0..=8).contains(&precision) {
        return Err(ParameterError::Settings {
            found: precision.to_string(),
            expected: "linear and angular precision in 0..=8",
        });
    }
    Ok(())
}

fn validate_display_mode<Id>(mode: DimensionDisplayType) -> Result<(), ParameterError<Id>> {
    if !matches!(
        mode,
        DIMENSION_DISPLAY_VALUE
            | DIMENSION_DISPLAY_NAME
            | DIMENSION_DISPLAY_EXPRESSION
            | DIMENSION_DISPLAY_TOLERANCE
            | DIMENSION_DISPLAY_PRECISE_VALUE
    ) {
        return Err(ParameterError::Settings {
            found: format!("{:?}", mode),
            expected: "Value, Name, Expression, Tolerance, or PreciseValue dimension display",
        });
    }
    Ok(())
}
