// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{CustomPropertyFormat, ParameterCollection},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "2 cm")
        .unwrap();
    parameters
        .insert_expression(2, "double", USER_PARAM, "width * 2")
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn custom_property_defaults_match_go_and_are_independent_of_annotations() {
    let parameters = parameters();
    let state = parameters.get(1).unwrap();
    assert_eq!(state.custom_property.property_type, CUSTOM_PROPERTY_TEXT);
    assert_eq!(
        state.custom_property.precision,
        PRECISION_THREE_DECIMAL_PLACES
    );
    assert!(state.custom_property.units.is_empty());
    assert!(state.custom_property.show_leading_zeros);
    assert!(!state.custom_property.show_trailing_zeros);
    assert!(state.custom_property.show_units_string);
    assert_eq!(state.disabled_actions, ACTION_NONE);
    assert!(!state.renamed);
}

#[test]
fn all_contract_precision_values_and_action_masks_are_accepted() {
    let mut parameters = parameters();
    let precisions = [
        PRECISION_ZERO_DECIMAL_PLACE,
        PRECISION_ONE_DECIMAL_PLACE,
        PRECISION_TWO_DECIMAL_PLACES,
        PRECISION_THREE_DECIMAL_PLACES,
        PRECISION_FOUR_DECIMAL_PLACES,
        PRECISION_FIVE_DECIMAL_PLACES,
        PRECISION_SIX_DECIMAL_PLACES,
        PRECISION_SEVEN_DECIMAL_PLACES,
        PRECISION_EIGHT_DECIMAL_PLACES,
        PRECISION_ZERO_FRACTIONAL,
        PRECISION_HALF_FRACTIONAL,
        PRECISION_QUARTER_FRACTIONAL,
        PRECISION_EIGHTHS_FRACTIONAL,
        PRECISION_SIXTEENTHS_FRACTIONAL,
        PRECISION_THIRTY_SECONDS_FRACTIONAL,
        PRECISION_SIXTY_FOURTHS_FRACTIONAL,
        PRECISION_ONE_TWENTY_EIGHTHS_FRACTIONAL,
        PRECISION_DEGREES_ANGLE,
        PRECISION_MINUTES_ANGLE,
        PRECISION_SECONDS_ANGLE,
        PRECISION_SECONDS_ONE_DECIMAL_ANGLE,
        PRECISION_SECONDS_TWO_DECIMAL_ANGLE,
        PRECISION_SECONDS_THREE_DECIMAL_ANGLE,
        PRECISION_SECONDS_FOUR_DECIMAL_ANGLE,
    ];
    for precision in precisions {
        let format = CustomPropertyFormat {
            precision,
            ..Default::default()
        };
        parameters
            .set_custom_property_format(1, format.clone())
            .unwrap();
        assert_eq!(parameters.get(1).unwrap().custom_property, format);
    }
    for mask in 0..=7 {
        parameters
            .set_disabled_action_types(1, ActionType(mask))
            .unwrap();
        assert_eq!(
            parameters.get(1).unwrap().disabled_actions,
            ActionType(mask)
        );
    }
}

#[test]
fn metadata_edits_preserve_nominal_values_and_dependency_state() {
    let mut parameters = parameters();
    let before = parameters.get(2).unwrap().clone();
    let format = CustomPropertyFormat {
        property_type: CUSTOM_PROPERTY_NUMBER,
        units: "mm".into(),
        show_leading_zeros: false,
        show_trailing_zeros: true,
        show_units_string: false,
        ..Default::default()
    };
    parameters
        .set_custom_property_format(1, format.clone())
        .unwrap();
    parameters
        .set_disabled_action_types(1, ACTION_EDIT)
        .unwrap();
    parameters.set_renamed(1, true).unwrap();
    assert!(parameters.get(1).unwrap().renamed);
    assert_eq!(parameters.get(1).unwrap().custom_property, format);
    assert_eq!(parameters.get(2), Some(&before));
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn invalid_metadata_is_refused_without_partial_mutation() {
    let mut parameters = parameters();
    let before = parameters.get(1).unwrap().clone();
    for precision in [
        CustomPropertyPrecision(-1),
        CustomPropertyPrecision(85504),
        CustomPropertyPrecision(85529),
    ] {
        assert!(
            parameters
                .set_custom_property_format(
                    1,
                    CustomPropertyFormat {
                        precision,
                        ..Default::default()
                    }
                )
                .is_err()
        );
    }
    let invalid = CustomPropertyFormat {
        property_type: CustomPropertyType(-1),
        ..Default::default()
    };
    assert!(parameters.set_custom_property_format(1, invalid).is_err());
    for mask in [-1, 8, i32::MAX] {
        assert!(
            parameters
                .set_disabled_action_types(1, ActionType(mask))
                .is_err()
        );
    }
    assert!(parameters.set_renamed(99, true).is_err());
    assert!(
        parameters
            .set_disabled_action_types(99, ACTION_NONE)
            .is_err()
    );
    assert!(
        parameters
            .set_custom_property_format(99, CustomPropertyFormat::default())
            .is_err()
    );
    assert_eq!(parameters.get(1), Some(&before));
}

#[test]
fn literals_and_read_only_measurements_retain_formatting_metadata() {
    let mut parameters = parameters();
    parameters.insert_text(3, "label", "hello").unwrap();
    parameters
        .insert_value(
            4,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(4.0, Dimension::Length),
        )
        .unwrap();
    for id in [3, 4] {
        parameters
            .set_custom_property_format(id, CustomPropertyFormat::default())
            .unwrap();
        parameters
            .set_disabled_action_types(id, ACTION_DELETE)
            .unwrap();
        parameters.set_renamed(id, true).unwrap();
        assert_eq!(parameters.get(id).unwrap().disabled_actions, ACTION_DELETE);
    }
}

#[test]
fn user_copy_resets_property_format_action_mask_and_rename_marker_like_go() {
    let mut parameters = parameters();
    let format = CustomPropertyFormat {
        units: "mm".into(),
        ..Default::default()
    };
    parameters.set_custom_property_format(1, format).unwrap();
    parameters
        .set_disabled_action_types(1, ACTION_RENAME)
        .unwrap();
    parameters.set_renamed(1, true).unwrap();
    parameters.copy_to_user(1, 3).unwrap();
    let copy = parameters.get(3).unwrap();
    assert_eq!(copy.custom_property, CustomPropertyFormat::default());
    assert_eq!(copy.disabled_actions, ACTION_NONE);
    assert!(!copy.renamed);
    assert_eq!(parameters.get(1).unwrap().custom_property.units, "mm");
}
