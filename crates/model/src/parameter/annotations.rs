// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{
    DISPLAY_FORMAT_ARCHITECTURAL, DISPLAY_FORMAT_DECIMAL, DISPLAY_FORMAT_FRACTIONAL,
    ParameterDisplayFormat,
};

/// Model-owned presentation preferences, independent of numeric evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ParameterAnnotations {
    pub comment: String,
    pub is_key: bool,
    pub exposed_as_property: bool,
    pub visible: bool,
    pub precision: i64,
    pub display_format: ParameterDisplayFormat,
}

impl Default for ParameterAnnotations {
    fn default() -> Self {
        Self {
            comment: String::new(),
            is_key: false,
            exposed_as_property: false,
            visible: true,
            precision: 0,
            display_format: DISPLAY_FORMAT_DECIMAL,
        }
    }
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Replace annotations without recomputing values; for example `parameters.set_annotations(id, annotations)`.
    pub fn set_annotations(
        &mut self,
        id: Id,
        annotations: ParameterAnnotations,
    ) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        if annotations.precision < 0
            || !matches!(
                annotations.display_format,
                DISPLAY_FORMAT_DECIMAL | DISPLAY_FORMAT_FRACTIONAL | DISPLAY_FORMAT_ARCHITECTURAL
            )
        {
            return Err(ParameterError::Value {
                id,
                found: format!("{annotations:?}"),
                expected: "nonnegative precision and a Decimal, Fractional, or Architectural display format",
            });
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .annotations = annotations;
        Ok(())
    }
}
