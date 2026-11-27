// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};

fn sweep_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name, source) in [
        (1, "plain", "10 mm"),
        (2, "a", "10 mm"),
        (3, "b", "20 mm"),
        (5, "reader", "a + b"),
    ] {
        parameters
            .insert_expression(id, name, USER_PARAM, source)
            .unwrap();
    }
    parameters.insert_text(4, "label", "lid").unwrap();
    parameters.set_deviation(2, 0.02, -0.01).unwrap();
    parameters.set_symmetric(3, 0.05).unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn all_selections_match_go_bands_and_leave_nominal_readers_unchanged() {
    let mut parameters = sweep_parameters();
    for (selection, a, b) in [
        (MODEL_VALUE_UPPER, 1.02, 2.05),
        (MODEL_VALUE_LOWER, 0.99, 1.95),
        (MODEL_VALUE_MEDIAN, 1.005, 2.0),
        (MODEL_VALUE_NOMINAL, 1.0, 2.0),
    ] {
        assert_eq!(parameters.set_all_model_value_type(selection).unwrap(), 2);
        assert_eq!(parameters.model_value(2).unwrap().value, a);
        assert_eq!(parameters.model_value(3).unwrap().value, b);
        assert_eq!(parameters.model_value(5).unwrap().value, 3.0);
        assert_eq!(parameters.drain_changed(), [2, 3]);
    }
    assert_eq!(
        parameters.get(1).unwrap().model_value_type,
        MODEL_VALUE_NOMINAL
    );
    assert_eq!(
        parameters.get(4).unwrap().model_value_type,
        MODEL_VALUE_NOMINAL
    );
}

#[test]
fn repeated_sweeps_report_eligible_count_without_dirtying_unchanged_values() {
    let mut parameters = sweep_parameters();
    assert_eq!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_NOMINAL)
            .unwrap(),
        2
    );
    assert!(parameters.drain_changed().is_empty());
    parameters
        .set_all_model_value_type(MODEL_VALUE_UPPER)
        .unwrap();
    parameters.drain_changed();
    assert_eq!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_UPPER)
            .unwrap(),
        2
    );
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn invalid_selections_are_rejected_even_for_empty_collections() {
    let mut empty = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    assert_eq!(
        empty.set_all_model_value_type(MODEL_VALUE_UPPER).unwrap(),
        0
    );
    for selection in [ModelValueType(0), ModelValueType(9)] {
        let error = empty
            .set_all_model_value_type(selection)
            .unwrap_err()
            .to_string();
        assert!(error.contains(&format!("{selection:?}")) && error.contains("expected"));
        let mut parameters = sweep_parameters();
        assert!(parameters.set_all_model_value_type(selection).is_err());
        assert!(parameters.drain_changed().is_empty());
        assert_eq!(
            parameters.get(2).unwrap().model_value_type,
            MODEL_VALUE_NOMINAL
        );
    }
}

#[test]
fn failed_member_refuses_whole_sweep_before_any_selection_changes() {
    let mut parameters = sweep_parameters();
    parameters.set_expression(3, "missing").unwrap();
    parameters.drain_changed();
    let before: Vec<_> = parameters
        .ids()
        .iter()
        .map(|id| parameters.get(*id).unwrap().clone())
        .collect();
    assert!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_UPPER)
            .is_err()
    );
    let after: Vec<_> = parameters
        .ids()
        .iter()
        .map(|id| parameters.get(*id).unwrap().clone())
        .collect();
    assert_eq!(after, before);
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(3, "20 mm").unwrap();
    assert_eq!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_UPPER)
            .unwrap(),
        2
    );
}

#[test]
fn zero_width_and_bandless_records_are_eligible_until_reset_to_default() {
    let mut parameters = sweep_parameters();
    parameters.set_symmetric(2, 0.0).unwrap();
    parameters.set_bandless(3, TOLERANCE_BASIC).unwrap();
    assert_eq!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_LOWER)
            .unwrap(),
        2
    );
    assert_eq!(parameters.model_value(2).unwrap().value, 1.0);
    parameters.set_tolerance_default(3).unwrap();
    assert_eq!(
        parameters
            .set_all_model_value_type(MODEL_VALUE_UPPER)
            .unwrap(),
        1
    );
    assert_eq!(
        parameters.get(3).unwrap().model_value_type,
        MODEL_VALUE_LOWER
    );
}
