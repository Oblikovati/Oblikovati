// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use oblikovati_api::types::*;
use oblikovati_model::parameter::CustomPropertyFormat;
use oblikovati_model::parameter::vocabulary::PRECISION_NAMES;
use serde::{Deserialize, Serialize};

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub(super) struct PropertyRecipe {
    property_type: String,
    units: String,
    precision: String,
    show_leading_zeros: bool,
    show_trailing_zeros: bool,
    show_units_string: bool,
}

impl PropertyRecipe {
    pub(super) fn restore(&self) -> Result<CustomPropertyFormat, PackageCodecError> {
        Ok(CustomPropertyFormat {
            property_type: property_type(&self.property_type)?,
            units: self.units.clone(),
            precision: precision(&self.precision)?,
            show_leading_zeros: self.show_leading_zeros,
            show_trailing_zeros: self.show_trailing_zeros,
            show_units_string: self.show_units_string,
        })
    }
}

fn property_type(name: &str) -> Result<CustomPropertyType, PackageCodecError> {
    match name {
        "" | "text" => Ok(CUSTOM_PROPERTY_TEXT),
        "number" => Ok(CUSTOM_PROPERTY_NUMBER),
        _ => Err(PackageCodecError(format!(
            "custom property type {name:?}: expected text or number"
        ))),
    }
}

fn precision(name: &str) -> Result<CustomPropertyPrecision, PackageCodecError> {
    if name.is_empty() {
        return Ok(PRECISION_THREE_DECIMAL_PLACES);
    }
    PRECISION_NAMES.iter().find(|(spelling, _)| *spelling == name).map(|(_, value)| *value)
        .ok_or_else(|| PackageCodecError(format!("custom property precision {name:?}: expected a defined decimal, fractional, or angle precision spelling")))
}

impl PropertyRecipe {
    pub(super) fn capture(format: &CustomPropertyFormat) -> Result<Self, PackageCodecError> {
        let precision = PRECISION_NAMES
            .iter()
            .find(|(_, value)| *value == format.precision)
            .ok_or_else(|| {
                PackageCodecError(format!(
                    "property precision {:?}: expected a known API precision",
                    format.precision
                ))
            })?
            .0;
        let property_type = match format.property_type {
            CUSTOM_PROPERTY_TEXT => "text",
            CUSTOM_PROPERTY_NUMBER => "number",
            found => {
                return Err(PackageCodecError(format!(
                    "property type {found:?}: expected text or number"
                )));
            }
        };
        Ok(Self {
            property_type: property_type.into(),
            units: format.units.clone(),
            precision: precision.into(),
            show_leading_zeros: format.show_leading_zeros,
            show_trailing_zeros: format.show_trailing_zeros,
            show_units_string: format.show_units_string,
        })
    }
}
