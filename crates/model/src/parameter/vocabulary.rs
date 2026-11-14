// SPDX-License-Identifier: GPL-2.0-only

//! Canonical parameter vocabulary shared by storage and automation.
use oblikovati_api::types::*;

pub const PRECISION_NAMES: &[(&str, CustomPropertyPrecision)] = &[
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

pub const TOLERANCE_NAMES: [(&str, ToleranceType); 14] = [
    ("default", TOLERANCE_DEFAULT),
    ("override", TOLERANCE_OVERRIDE),
    ("symmetric", TOLERANCE_SYMMETRIC),
    ("deviation", TOLERANCE_DEVIATION),
    ("limitsStacked", TOLERANCE_LIMITS_STACKED),
    ("limitLinear", TOLERANCE_LIMIT_LINEAR),
    ("max", TOLERANCE_MAX),
    ("min", TOLERANCE_MIN),
    ("limitsFitsStacked", TOLERANCE_LIMITS_FITS_STACKED),
    ("limitsFitsLinear", TOLERANCE_LIMITS_FITS_LINEAR),
    ("limitsFitsShowSize", TOLERANCE_LIMITS_FITS_SHOW_SIZE),
    (
        "limitsFitsShowTolerance",
        TOLERANCE_LIMITS_FITS_SHOW_TOLERANCE,
    ),
    ("basic", TOLERANCE_BASIC),
    ("reference", TOLERANCE_REFERENCE),
];

/// Read a validated display format name; example: `display_name(DISPLAY_FORMAT_DECIMAL)`.
pub fn display_name(value: ParameterDisplayFormat) -> Option<&'static str> {
    match value {
        DISPLAY_FORMAT_DECIMAL => Some("decimal"),
        DISPLAY_FORMAT_FRACTIONAL => Some("fractional"),
        DISPLAY_FORMAT_ARCHITECTURAL => Some("architectural"),
        _ => None,
    }
}
/// Read a model selection name; example: `selection_name(MODEL_VALUE_NOMINAL)`.
pub fn selection_name(value: ModelValueType) -> Option<&'static str> {
    match value {
        MODEL_VALUE_NOMINAL => Some("nominal"),
        MODEL_VALUE_UPPER => Some("upper"),
        MODEL_VALUE_LOWER => Some("lower"),
        MODEL_VALUE_MEDIAN => Some("median"),
        _ => None,
    }
}
/// Read disabled actions in canonical order; example: `action_names(ACTION_EDIT)`.
pub fn action_names(value: ActionType) -> Vec<String> {
    [
        ("edit", ACTION_EDIT),
        ("rename", ACTION_RENAME),
        ("delete", ACTION_DELETE),
    ]
    .into_iter()
    .filter(|(_, flag)| value.0 & flag.0 != 0)
    .map(|(name, _)| name.into())
    .collect()
}
