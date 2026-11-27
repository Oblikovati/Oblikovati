// SPDX-License-Identifier: GPL-2.0-only

use super::{ParameterCollection, ParameterError};
use oblikovati_api::types::{
    ACTION_DELETE, ACTION_EDIT, ACTION_RENAME, ActionType, CUSTOM_PROPERTY_NUMBER,
    CUSTOM_PROPERTY_TEXT, CustomPropertyPrecision, CustomPropertyType,
    PRECISION_SECONDS_FOUR_DECIMAL_ANGLE, PRECISION_THREE_DECIMAL_PLACES,
    PRECISION_ZERO_DECIMAL_PLACE,
};
use std::fmt::Debug;

/// Parameter-to-document-property formatting state; numeric evaluation is separate.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct CustomPropertyFormat {
    pub property_type: CustomPropertyType,
    pub units: String,
    pub precision: CustomPropertyPrecision,
    pub show_leading_zeros: bool,
    pub show_trailing_zeros: bool,
    pub show_units_string: bool,
}

impl Default for CustomPropertyFormat {
    fn default() -> Self {
        Self {
            property_type: CUSTOM_PROPERTY_TEXT,
            units: String::new(),
            precision: PRECISION_THREE_DECIMAL_PLACES,
            show_leading_zeros: true,
            show_trailing_zeros: false,
            show_units_string: true,
        }
    }
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Store export-format preferences without changing nominal values or dependencies.
    /// Example: `parameters.set_custom_property_format(id, CustomPropertyFormat::default())?;`.
    pub fn set_custom_property_format(
        &mut self,
        id: Id,
        format: CustomPropertyFormat,
    ) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        if !matches!(
            format.property_type,
            CUSTOM_PROPERTY_TEXT | CUSTOM_PROPERTY_NUMBER
        ) || !(PRECISION_ZERO_DECIMAL_PLACE.0..=PRECISION_SECONDS_FOUR_DECIMAL_ANGLE.0)
            .contains(&format.precision.0)
        {
            return Err(ParameterError::Value {
                id,
                found: format!("{format:?}"),
                expected: "Text or Number property type and a defined custom-property precision",
            });
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .custom_property = format;
        Ok(())
    }

    /// Store the host's restricted-action mask; host commands own its enforcement.
    /// Example: `parameters.set_disabled_action_types(id, oblikovati_api::types::ACTION_RENAME)?;`.
    pub fn set_disabled_action_types(
        &mut self,
        id: Id,
        mask: ActionType,
    ) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        let known = ACTION_EDIT.0 | ACTION_RENAME.0 | ACTION_DELETE.0;
        if mask.0 & !known != 0 {
            return Err(ParameterError::Value {
                id,
                found: format!("{mask:?}"),
                expected: "a combination of Edit, Rename, and Delete action bits",
            });
        }
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .disabled_actions = mask;
        Ok(())
    }

    /// Restore the persisted rename marker independently of the current display name.
    /// Example: `parameters.set_renamed(id, true)?;`.
    pub fn set_renamed(&mut self, id: Id, renamed: bool) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        self.records
            .get_mut(&id)
            .expect("validated parameter")
            .state
            .renamed = renamed;
        Ok(())
    }
}
