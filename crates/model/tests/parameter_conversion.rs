// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn conversion_preserves_identity_metadata_and_edges() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "od", USER_PARAM, "10 cm")
        .unwrap();
    parameters
        .insert_expression(2, "half", USER_PARAM, "od / 2")
        .unwrap();
    parameters.set_symmetric(1, 0.02).unwrap();
    parameters.drain_changed();
    let mut expected = parameters.get(1).unwrap().clone();
    parameters.convert_kind(1, MODEL_PARAM).unwrap();
    expected.kind = MODEL_PARAM;
    assert_eq!(parameters.get(1), Some(&expected));
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "20 cm").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 10.0);
    parameters.convert_kind(1, USER_PARAM).unwrap();
    assert_eq!(parameters.get(1).unwrap().kind, USER_PARAM);
}

#[test]
fn reference_conversion_enforces_editability_and_can_be_reversed() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "label", "steel").unwrap();
    parameters
        .insert_expression(2, "width", TABLE_PARAM, "5 cm")
        .unwrap();
    for id in [1, 2] {
        parameters.convert_kind(id, REFERENCE_PARAM).unwrap();
    }
    assert!(parameters.set_text(1, "wood").is_err());
    assert!(parameters.set_expression(2, "6 cm").is_err());
    parameters
        .update_measured(2, Quantity::new(7.0, Dimension::Length))
        .unwrap();
    for id in [1, 2] {
        parameters.convert_kind(id, USER_PARAM).unwrap();
    }
    parameters.set_text(1, "wood").unwrap();
    parameters.set_expression(2, "6 cm").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 6.0);
}

#[test]
fn auto_dimensions_skip_taken_names_and_keep_their_builtin_guard_after_rename() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "d0", USER_PARAM, "1 cm")
        .unwrap();
    parameters.insert_auto_model(2, "3 cm").unwrap();
    parameters.insert_auto_model(3, "d1 * 2").unwrap();
    assert_eq!(parameters.get(2).unwrap().name, "d1");
    assert_eq!(parameters.get(3).unwrap().name, "d2");
    assert!(parameters.get(2).unwrap().built_in);
    parameters.rename(2, "depth").unwrap();
    assert!(parameters.get(2).unwrap().renamed);
    assert!(parameters.convert_kind(2, USER_PARAM).is_err());
    assert!(parameters.convert_kind(2, MODEL_PARAM).is_err());
    assert!(parameters.remove(2).is_err());
    parameters.copy_to_user(2, 4).unwrap();
    assert!(!parameters.get(4).unwrap().built_in);
    parameters.convert_kind(4, REFERENCE_PARAM).unwrap();
}

#[test]
fn invalid_conversions_leave_the_full_parameter_unchanged() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(
            1,
            "linked",
            DERIVED_PARAM,
            Quantity::new(1.0, Dimension::Length),
        )
        .unwrap();
    parameters
        .insert_expression(2, "user", USER_PARAM, "2 cm")
        .unwrap();
    parameters.drain_changed();
    let derived = parameters.get(1).unwrap().clone();
    let user = parameters.get(2).unwrap().clone();
    assert!(parameters.convert_kind(1, USER_PARAM).is_err());
    for target in [DERIVED_PARAM, TABLE_PARAM, ParameterKind(99)] {
        assert!(parameters.convert_kind(2, target).is_err());
        assert_eq!(parameters.get(2), Some(&user));
    }
    assert!(parameters.convert_kind(99, USER_PARAM).is_err());
    assert_eq!(parameters.get(1), Some(&derived));
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn malformed_auto_insertion_does_not_consume_a_name_or_identity() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    assert!(parameters.insert_auto_model(1, "1 +").is_err());
    assert!(parameters.insert_auto_model(1, "d0 + 1").is_err());
    assert!(parameters.ids().is_empty());
    assert!(parameters.drain_changed().is_empty());
    parameters.insert_auto_model(1, "3 cm").unwrap();
    assert_eq!(parameters.get(1).unwrap().name, "d0");
    assert!(parameters.insert_auto_model(1, "4 cm").is_err());
    assert_eq!(parameters.ids(), &[1]);
    parameters.remove_for_owner(1).unwrap();
    parameters.insert_auto_model(2, "4 cm").unwrap();
    assert_eq!(parameters.get(2).unwrap().name, "d0");
}
