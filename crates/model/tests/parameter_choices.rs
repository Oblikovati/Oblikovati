// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{REFERENCE_PARAM, USER_PARAM};
use oblikovati_api::wire::ExpressionListInfo;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn numeric_choices(custom: bool) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "10 mm")
        .unwrap();
    parameters
        .set_expression_list(1, vec!["10 mm".into(), "20 mm".into()], custom)
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn setting_and_clearing_choices_preserves_current_values() {
    let mut parameters = numeric_choices(false);
    let before = parameters.get(1).unwrap().current_value();
    let choices = &parameters.get(1).unwrap().choices;
    assert_eq!(choices.expressions.as_ref().unwrap(), &["10 mm", "20 mm"]);
    assert!(choices.custom_order && !choices.allow_custom_values);
    assert!(parameters.drain_changed().is_empty());
    parameters.clear_expression_list(1).unwrap();
    assert_eq!(
        parameters.get(1).unwrap().choices,
        ExpressionListInfo::default()
    );
    assert_eq!(parameters.get(1).unwrap().current_value(), before);
    parameters.set_custom_order(1, true).unwrap();
    assert!(!parameters.get(1).unwrap().choices.custom_order);
    parameters.set_expression_list(1, Vec::new(), true).unwrap();
    assert_eq!(
        parameters.get(1).unwrap().choices,
        ExpressionListInfo::default()
    );
}

#[test]
fn numeric_selection_recomputes_dependents_and_rejects_unlisted_source() {
    let mut parameters = numeric_choices(false);
    parameters
        .insert_expression(2, "twice", USER_PARAM, "width * 2")
        .unwrap();
    parameters.drain_changed();
    parameters.select_value(1, "20 mm").unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 2.0);
    assert_eq!(parameters.model_value(2).unwrap().value, 4.0);
    assert_eq!(parameters.drain_changed(), [1, 2]);
    let before = parameters.get(1).unwrap().clone();
    assert!(
        parameters
            .select_value(1, "99 mm")
            .unwrap_err()
            .to_string()
            .contains("99 mm")
    );
    assert!(parameters.select_value(1, "20mm").is_err());
    assert_eq!(parameters.get(1), Some(&before));
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn custom_values_replace_current_source_without_growing_the_fixed_list() {
    let mut parameters = numeric_choices(true);
    for (source, expected) in [("35 mm", 3.5), ("40 mm", 4.0)] {
        parameters.select_value(1, source).unwrap();
        assert_eq!(parameters.model_value(1).unwrap().value, expected);
        assert_eq!(parameters.get(1).unwrap().source, source);
        assert_eq!(
            parameters
                .get(1)
                .unwrap()
                .choices
                .expressions
                .as_ref()
                .unwrap()
                .len(),
            2
        );
    }
    parameters.clear_expression_list(1).unwrap();
    parameters.select_value(1, "50 mm").unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 5.0);
}

#[test]
fn lexical_sort_preserves_duplicates_and_reenabling_order_keeps_current_order() {
    let mut parameters = numeric_choices(false);
    let original = vec!["鋼".into(), "z".into(), "a".into(), "a".into(), "é".into()];
    parameters
        .set_expression_list(1, original.clone(), false)
        .unwrap();
    assert_eq!(
        parameters.get(1).unwrap().choices.expressions.as_ref(),
        Some(&original)
    );
    parameters.set_custom_order(1, false).unwrap();
    assert_eq!(
        parameters
            .get(1)
            .unwrap()
            .choices
            .expressions
            .as_ref()
            .unwrap(),
        &["a", "a", "z", "é", "鋼"]
    );
    parameters.set_custom_order(1, true).unwrap();
    assert_eq!(
        parameters
            .get(1)
            .unwrap()
            .choices
            .expressions
            .as_ref()
            .unwrap()[0],
        "a"
    );
    assert!(parameters.get(1).unwrap().choices.custom_order);
    assert_eq!(parameters.model_value(1).unwrap().value, 1.0);
}

#[test]
fn text_and_boolean_selections_use_typed_setters() {
    let mut parameters = numeric_choices(false);
    parameters.insert_text(2, "material", "steel").unwrap();
    parameters.insert_boolean(3, "vented", true).unwrap();
    parameters
        .set_expression_list(2, vec!["wood".into(), "width * 2".into()], false)
        .unwrap();
    parameters
        .set_expression_list(3, vec!["true".into(), "false".into()], false)
        .unwrap();
    parameters.select_value(2, "width * 2").unwrap();
    parameters.select_value(3, "false").unwrap();
    assert_eq!(parameters.text_value(2).unwrap(), "width * 2");
    assert_eq!(parameters.get(2).unwrap().source, "\"width * 2\"");
    assert!(!parameters.boolean_value(3).unwrap());
    parameters
        .set_expression_list(3, vec!["TRUE".into()], true)
        .unwrap();
    assert!(parameters.select_value(3, "TRUE").is_err());
    assert!(!parameters.boolean_value(3).unwrap());
}

#[test]
fn all_choice_edits_reject_missing_and_read_only_parameters() {
    let mut parameters = numeric_choices(false);
    parameters
        .insert_value(
            2,
            "measured",
            REFERENCE_PARAM,
            Quantity::new(1.0, Dimension::Length),
        )
        .unwrap();
    parameters.drain_changed();
    let before = parameters.get(2).unwrap().clone();
    for id in [2, 99] {
        assert!(
            parameters
                .set_expression_list(id, vec!["1 cm".into()], false)
                .is_err()
        );
        assert!(parameters.clear_expression_list(id).is_err());
        assert!(parameters.set_custom_order(id, false).is_err());
        assert!(parameters.select_value(id, "1 cm").is_err());
    }
    assert_eq!(parameters.get(2), Some(&before));
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn listed_formulas_bind_live_and_malformed_selection_is_atomic() {
    let mut parameters = numeric_choices(false);
    parameters
        .insert_expression(2, "base", USER_PARAM, "2 cm")
        .unwrap();
    parameters
        .set_expression_list(1, vec!["base * 3".into(), "1 +".into()], false)
        .unwrap();
    parameters.select_value(1, "base * 3").unwrap();
    parameters.set_expression(2, "4 cm").unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 12.0);
    assert_eq!(parameters.drivers(1).unwrap(), [2]);
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    assert!(parameters.select_value(1, "1 +").is_err());
    assert_eq!(parameters.get(1), Some(&before));
    assert!(parameters.drain_changed().is_empty());
}
