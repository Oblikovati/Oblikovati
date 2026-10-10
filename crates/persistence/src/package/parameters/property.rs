// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use oblikovati_api::types::*;
use oblikovati_model::parameter::CustomPropertyFormat;
use serde::Deserialize;

#[derive(Default, Deserialize)]
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

const PRECISION_NAMES: &[(&str, CustomPropertyPrecision)] = &[
    ("zeroDecimalPlace", PRECISION_ZERO_DECIMAL_PLACE),
    ("oneDecimalPlace", PRECISION_ONE_DECIMAL_PLACE),
    ("twoDecimalPlaces", PRECISION_TWO_DECIMAL_PLACES),
    ("threeDecimalPlaces", PRECISION_THREE_DECIMAL_PLACES),
    ("fourDecimalPlaces", PRECISION_FOUR_DECIMAL_PLACES),
    ("fiveDecimalPlaces", PRECISION_FIVE_DECIMAL_PLACES),
    ("sixDecimalPlaces", PRECISION_SIX_DECIMAL_PLACES),
    ("sevenDecimalPlaces", PRECISION_SEVEN_DECIMAL_PLACES),
    ("eightDecimalPlaces", PRECISION_EIGHT_DECIMAL_PLACES),
    ("zeroFractional", PRECISION_ZERO_FRACTIONAL),
    ("halfFractional", PRECISION_HALF_FRACTIONAL),
    ("quarterFractional", PRECISION_QUARTER_FRACTIONAL),
    ("eighthsFractional", PRECISION_EIGHTHS_FRACTIONAL),
    ("sixteenthsFractional", PRECISION_SIXTEENTHS_FRACTIONAL),
    (
        "thirtySecondsFractional",
        PRECISION_THIRTY_SECONDS_FRACTIONAL,
    ),
    ("sixtyFourthsFractional", PRECISION_SIXTY_FOURTHS_FRACTIONAL),
    (
        "oneTwentyEighthsFractional",
        PRECISION_ONE_TWENTY_EIGHTHS_FRACTIONAL,
    ),
    ("degrees", PRECISION_DEGREES_ANGLE),
    ("minutes", PRECISION_MINUTES_ANGLE),
    ("seconds", PRECISION_SECONDS_ANGLE),
    (
        "secondsOneDecimalPlace",
        PRECISION_SECONDS_ONE_DECIMAL_ANGLE,
    ),
    (
        "secondsTwoDecimalPlaces",
        PRECISION_SECONDS_TWO_DECIMAL_ANGLE,
    ),
    (
        "secondsThreeDecimalPlaces",
        PRECISION_SECONDS_THREE_DECIMAL_ANGLE,
    ),
    (
        "secondsFourDecimalPlaces",
        PRECISION_SECONDS_FOUR_DECIMAL_ANGLE,
    ),
];
