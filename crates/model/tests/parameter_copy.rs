// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{ParameterAnnotations, ParameterCollection},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "depth", MODEL_PARAM, "5 mm")
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn annotations_default_to_visible_decimal_and_reject_invalid_edits_atomically() {
    let mut parameters = parameters();
    let defaults = ParameterAnnotations::default();
    assert_eq!(parameters.get(1).unwrap().annotations, defaults);
    assert!(defaults.visible);
    assert_eq!(defaults.display_format, DISPLAY_FORMAT_DECIMAL);
    for invalid in [
        ParameterAnnotations {
            precision: -1,
            ..defaults.clone()
        },
        ParameterAnnotations {
            display_format: ParameterDisplayFormat(0),
            ..defaults.clone()
        },
    ] {
        assert!(parameters.set_annotations(1, invalid).is_err());
        assert_eq!(parameters.get(1).unwrap().annotations, defaults);
    }
    assert!(parameters.set_annotations(99, defaults).is_err());
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn numeric_copy_carries_selected_metadata_and_resets_presentation_as_in_go() {
    let mut parameters = parameters();
    parameters
        .set_annotations(
            1,
            ParameterAnnotations {
                comment: "depth label".into(),
                is_key: true,
                exposed_as_property: true,
                visible: false,
                precision: 4,
                display_format: DISPLAY_FORMAT_FRACTIONAL,
            },
        )
        .unwrap();
    parameters.set_deviation(1, 0.02, -0.01).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters
        .set_expression_list(1, vec!["20 mm".into(), "10 mm".into()], true)
        .unwrap();
    parameters.set_custom_order(1, false).unwrap();
    parameters.copy_to_user(1, 2).unwrap();
    check_copied_metadata(&parameters);
    parameters.set_expression(2, "9 mm").unwrap();
    parameters.clear_expression_list(2).unwrap();
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        0.5
    );
    assert!(parameters.get(1).unwrap().choices.expressions.is_some());
}

fn check_copied_metadata(parameters: &ParameterCollection<u64>) {
    let source = parameters.get(1).unwrap();
    let copy = parameters.get(2).unwrap();
    assert_eq!(copy.name, "depth_copy");
    assert_eq!(copy.kind, USER_PARAM);
    assert_eq!(copy.annotations.comment, "depth label");
    assert!(
        copy.annotations.is_key && copy.annotations.exposed_as_property && copy.annotations.visible
    );
    assert_eq!(copy.annotations.precision, 4);
    assert_eq!(copy.annotations.display_format, DISPLAY_FORMAT_DECIMAL);
    assert_eq!(copy.tolerance, source.tolerance);
    assert_eq!(copy.model_value_type, MODEL_VALUE_NOMINAL);
    assert!(copy.choices.custom_order && copy.choices.allow_custom_values);
    assert_eq!(copy.choices.expressions, source.choices.expressions);
}

#[test]
fn repeated_copy_names_skip_collisions_and_do_not_inherit_memberships_or_renamed_flags() {
    let mut parameters = parameters();
    parameters.rename(1, "thickness").unwrap();
    parameters.add_group("feature", "", "").unwrap();
    parameters.add_to_group(1, "feature").unwrap();
    for id in 2..5 {
        parameters.copy_to_user(1, id).unwrap();
    }
    assert_eq!(parameters.get(2).unwrap().name, "thickness_copy");
    assert_eq!(parameters.get(3).unwrap().name, "thickness_copy2");
    assert_eq!(parameters.get(4).unwrap().name, "thickness_copy3");
    assert!(!parameters.get(2).unwrap().renamed);
    assert!(parameters.groups_of(2).unwrap().is_empty());
    assert_eq!(parameters.group_members("feature").unwrap(), [1]);
}

#[test]
fn copied_formulas_keep_drivers_but_can_be_edited_independently() {
    let mut parameters = parameters();
    parameters
        .insert_expression(2, "double", USER_PARAM, "depth * 2")
        .unwrap();
    parameters.copy_to_user(2, 3).unwrap();
    parameters.set_expression(1, "10 mm").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 2.0);
    assert_eq!(parameters.model_value(3).unwrap().value, 2.0);
    assert_eq!(parameters.drivers(3).unwrap(), [1]);
    parameters.set_expression(3, "99 mm").unwrap();
    parameters.set_expression(1, "15 mm").unwrap();
    assert_eq!(parameters.model_value(2).unwrap().value, 3.0);
    assert_eq!(parameters.model_value(3).unwrap().value, 99.0 * 0.1);
    assert!(parameters.drivers(3).unwrap().is_empty());
}

#[test]
fn typed_literals_and_measured_constants_copy_without_reparsing_diagnostic_source() {
    let mut parameters = parameters();
    parameters
        .insert_text(2, "label", "steel \"plate\"")
        .unwrap();
    parameters.insert_boolean(3, "vented", true).unwrap();
    parameters
        .insert_value(
            4,
            "area",
            REFERENCE_PARAM,
            Quantity::new(4.0, Dimension::Area),
        )
        .unwrap();
    parameters
        .set_annotations(
            4,
            ParameterAnnotations {
                comment: "measurement".into(),
                ..ParameterAnnotations::default()
            },
        )
        .unwrap();
    for id in 2..5 {
        parameters.copy_to_user(id, id + 10).unwrap();
    }
    parameters.set_text(12, "wood").unwrap();
    parameters.set_boolean(13, false).unwrap();
    assert_eq!(parameters.text_value(2).unwrap(), "steel \"plate\"");
    assert!(parameters.boolean_value(3).unwrap());
    assert_eq!(
        parameters.model_value(14).unwrap(),
        Quantity::new(4.0, Dimension::Area)
    );
    assert_eq!(
        parameters.get(14).unwrap().annotations.comment,
        "measurement"
    );
    assert_eq!(parameters.get(14).unwrap().kind, USER_PARAM);
}

#[test]
fn rejected_copy_does_not_allocate_names_ids_or_change_existing_state() {
    let mut parameters = parameters();
    let before = parameters.get(1).unwrap().clone();
    assert!(parameters.copy_to_user(99, 2).is_err());
    assert!(parameters.copy_to_user(1, 1).is_err());
    assert_eq!(parameters.get(1), Some(&before));
    assert_eq!(parameters.ids(), &[1]);
    assert!(parameters.find("depth_copy").is_none());
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn a_copy_cannot_bind_its_own_generated_name_and_failed_sources_do_not_invent_values() {
    let mut parameters = parameters();
    parameters.set_expression(1, "depth_copy").unwrap();
    parameters.drain_changed();
    assert!(parameters.copy_to_user(1, 2).is_err());
    assert!(parameters.get(2).is_none());
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "unknown").unwrap();
    parameters.copy_to_user(1, 2).unwrap();
    assert!(parameters.get(2).unwrap().current_value().is_none());
    assert!(parameters.get(2).unwrap().last_good_value.is_none());
    assert_eq!(parameters.get(2).unwrap().source, "unknown");
}
