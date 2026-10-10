// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{REFERENCE_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterFailure, ParameterHealth, ParameterValue},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn literals() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "material", "steel").unwrap();
    parameters.insert_boolean(2, "vented", true).unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn text_and_boolean_values_match_go_examples_without_numeric_stand_ins() {
    let parameters = literals();
    assert_eq!(parameters.text_value(1).unwrap(), "steel");
    assert!(parameters.boolean_value(2).unwrap());
    assert_eq!(parameters.get(1).unwrap().source, "\"steel\"");
    assert_eq!(parameters.get(2).unwrap().source, "true");
    for id in [1, 2] {
        assert_eq!(parameters.get(id).unwrap().kind, USER_PARAM);
        assert_eq!(parameters.get(id).unwrap().health, ParameterHealth::Healthy);
        assert!(parameters.get(id).unwrap().current_value().is_none());
        assert!(parameters.drivers(id).unwrap().is_empty());
    }
}

#[test]
fn literal_edits_are_typed_and_identical_values_are_no_ops() {
    let mut parameters = literals();
    parameters.set_text(1, "b c").unwrap();
    parameters.set_boolean(2, false).unwrap();
    assert_eq!(parameters.get(1).unwrap().source, "\"b c\"");
    assert_eq!(parameters.get(2).unwrap().source, "false");
    assert_eq!(parameters.drain_changed(), [1, 2]);
    parameters.set_text(1, "b c").unwrap();
    parameters.set_boolean(2, false).unwrap();
    assert!(parameters.drain_changed().is_empty());
    assert_eq!(
        parameters.get(2).unwrap().current(),
        Some(&ParameterValue::Boolean(false))
    );
}

#[test]
fn text_source_is_a_lossless_quoted_literal_even_with_formula_tokens() {
    let mut parameters = literals();
    for text in [
        "",
        "\"\\\n\t\r\0",
        "鋼 – aço 🛠",
        "vented + material",
        "true",
        "\u{2028}\u{7f}",
    ] {
        parameters.set_text(1, text).unwrap();
        assert_eq!(parameters.text_value(1).unwrap(), text);
        let source = &parameters.get(1).unwrap().source;
        assert_eq!(serde_json::from_str::<String>(source).unwrap(), text);
        assert!(parameters.drivers(1).unwrap().is_empty());
    }
    parameters.set_text(1, "\0\n\"\\").unwrap();
    assert_eq!(parameters.get(1).unwrap().source, "\"\\u0000\\n\\\"\\\\\"");
}

#[test]
fn numeric_edits_cannot_change_a_literal_parameter_kind() {
    let mut parameters = literals();
    for id in [1, 2] {
        let before = parameters.get(id).unwrap().clone();
        assert!(parameters.set_expression(id, "1 mm").is_err());
        assert!(
            parameters
                .set_value(id, Quantity::new(1.0, Dimension::Length))
                .is_err()
        );
        assert!(
            parameters
                .update_measured(id, Quantity::new(1.0, Dimension::Length))
                .is_err()
        );
        assert_eq!(parameters.get(id), Some(&before));
    }
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn wrong_literal_edits_and_reads_report_the_expected_kind() {
    let mut parameters = literals();
    assert!(
        parameters
            .set_text(2, "false")
            .unwrap_err()
            .to_string()
            .contains("text parameter")
    );
    assert!(
        parameters
            .set_boolean(1, false)
            .unwrap_err()
            .to_string()
            .contains("Boolean parameter")
    );
    assert!(parameters.text_value(2).is_err());
    assert!(parameters.boolean_value(1).is_err());
    for id in [1, 2] {
        assert!(parameters.model_value(id).is_err());
        assert!(parameters.set_deviation(id, 0.1, -0.1).is_err());
    }
    assert!(parameters.drain_changed().is_empty());
    assert_eq!(parameters.text_value(1).unwrap(), "steel");
    assert!(parameters.boolean_value(2).unwrap());
}

#[test]
fn literal_insertions_and_missing_id_edits_are_atomic() {
    let mut parameters = literals();
    assert!(parameters.insert_text(1, "other", "x").is_err());
    assert!(parameters.insert_boolean(3, "material", true).is_err());
    assert!(parameters.insert_text(3, "bad name", "x").is_err());
    assert!(parameters.set_text(99, "x").is_err());
    assert!(parameters.set_boolean(99, true).is_err());
    assert!(parameters.text_value(99).is_err());
    assert!(parameters.boolean_value(99).is_err());
    assert_eq!(parameters.ids(), &[1, 2]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn numeric_parameters_cannot_be_replaced_with_literal_values() {
    let mut parameters = literals();
    parameters
        .insert_expression(3, "width", USER_PARAM, "1 mm")
        .unwrap();
    parameters
        .insert_value(
            4,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(1.0, Dimension::Length),
        )
        .unwrap();
    parameters.drain_changed();
    for id in [3, 4] {
        let before = parameters.get(id).unwrap().clone();
        assert!(parameters.set_text(id, "x").is_err());
        assert!(parameters.set_boolean(id, false).is_err());
        assert_eq!(parameters.get(id), Some(&before));
    }
    reject_literal_quantities(&mut parameters);
    assert!(parameters.drain_changed().is_empty());
}

fn reject_literal_quantities(parameters: &mut ParameterCollection<u64>) {
    for dimension in [Dimension::Text, Dimension::Boolean] {
        let quantity = Quantity::new(1.0, dimension);
        assert!(
            parameters
                .insert_value(5, "fake", USER_PARAM, quantity)
                .is_err()
        );
        assert!(parameters.set_value(3, quantity).is_err());
        assert!(parameters.update_measured(4, quantity).is_err());
    }
    assert!(parameters.find("fake").is_none());
}

#[test]
fn numeric_references_to_literals_fail_explicitly_and_recover_after_replacement() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(3, "width", USER_PARAM, "material * 2")
        .unwrap();
    parameters.insert_text(1, "material", "steel").unwrap();
    assert_eq!(
        parameters.get(3).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::NonNumeric(vec![1]))
    );
    parameters.set_text(1, "wood").unwrap();
    assert!(parameters.get(3).unwrap().current_value().is_none());
    assert!(parameters.remove(1).is_err());
    parameters.remove_for_owner(1).unwrap();
    assert_eq!(
        parameters.get(3).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::Unresolved(vec!["material".into()]))
    );
    parameters
        .insert_expression(4, "material", USER_PARAM, "3 cm")
        .unwrap();
    assert_eq!(parameters.model_value(3).unwrap().value, 6.0);
}

#[test]
fn literal_rename_preserves_quoted_contents_and_updates_bound_numeric_sources() {
    let mut parameters = literals();
    parameters.set_text(1, "vented material").unwrap();
    parameters
        .insert_expression(3, "bad", USER_PARAM, "material + material")
        .unwrap();
    parameters.rename(1, "label").unwrap();
    parameters.rename(2, "enabled").unwrap();
    assert_eq!(parameters.text_value(1).unwrap(), "vented material");
    assert_eq!(parameters.get(1).unwrap().source, "\"vented material\"");
    assert_eq!(parameters.get(3).unwrap().source, "label + label");
    assert_eq!(parameters.drivers(3).unwrap(), [1]);
    assert!(parameters.boolean_value(2).unwrap());
    parameters.remove(2).unwrap();
    assert!(parameters.find("enabled").is_none());
}

#[test]
fn typed_read_scopes_keep_identity_and_failed_read_footprints() {
    let parameters = literals();
    let (text, reads) = parameters.track(|scope| {
        assert!(scope.boolean_value(2).unwrap());
        let (text, inner) = scope.track(|inner| inner.text_value(1).unwrap());
        assert_eq!(inner, [1]);
        assert!(scope.text_value(2).is_err());
        assert!(scope.boolean_value(99).is_err());
        text
    });
    assert_eq!(text, "steel");
    assert_eq!(reads, [1, 2]);
}
