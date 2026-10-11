// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM};
use oblikovati_model::{
    feature::*,
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use std::error::Error;

mod feature_suppression {
    use super::*;
    mod replay;
}

fn parameters(value: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(1, "count", USER_PARAM, Quantity::new(value, dimension))
        .unwrap();
    parameters
}

fn condition(parameter: u64, threshold: f64, dimension: Dimension) -> FeatureSuppressionCondition {
    FeatureSuppressionCondition::new(
        parameter,
        FeatureSuppressionComparison::LessThan,
        Quantity::new(threshold, dimension),
    )
    .unwrap()
}

fn authored(id: u64, keys: Vec<FeatureReadKey>) -> AuthoredFeature<u64> {
    let reads = FeatureReadFootprint::new(keys, false);
    AuthoredFeature::new(FeatureId(id), "Feature", id, reads)
}

fn conditioned_sequence() -> FeatureSequence<u64> {
    let mut sequence = FeatureSequence::default();
    for id in 1..=3 {
        let keys = if id == 2 {
            vec![FeatureReadKey::Feature(FeatureId(1))]
        } else {
            vec![]
        };
        sequence.append(authored(id, keys)).unwrap();
    }
    sequence
        .set_suppression_condition(FeatureId(1), Some(condition(1, 2.0, Dimension::Unitless)))
        .unwrap();
    sequence
}

const COMPARISON_CASES: [(FeatureSuppressionComparison, [bool; 3]); 6] = [
    (FeatureSuppressionComparison::Equal, [false, true, false]),
    (FeatureSuppressionComparison::NotEqual, [true, false, true]),
    (FeatureSuppressionComparison::LessThan, [true, false, false]),
    (
        FeatureSuppressionComparison::GreaterThan,
        [false, false, true],
    ),
    (
        FeatureSuppressionComparison::LessOrEqual,
        [true, true, false],
    ),
    (
        FeatureSuppressionComparison::GreaterOrEqual,
        [false, true, true],
    ),
];

#[test]
fn all_six_comparisons_match_exact_go_boundaries() {
    for (comparison, expected) in COMPARISON_CASES {
        let threshold = Quantity::new(2.0, Dimension::Unitless);
        let condition = FeatureSuppressionCondition::new(1, comparison, threshold).unwrap();
        assert_eq!(condition.comparison(), comparison);
        for (index, value) in [1.0, 2.0, 3.0].into_iter().enumerate() {
            let result = condition
                .evaluate(&parameters(value, Dimension::Unitless))
                .unwrap();
            assert_eq!(result, expected[index]);
        }
    }
}

#[test]
fn condition_uses_selected_tolerance_value_and_survives_parameter_rename() {
    let mut parameters = parameters(1.0, Dimension::Length);
    let condition = condition(1, 2.0, Dimension::Length);
    assert!(condition.evaluate(&parameters).unwrap());
    parameters.set_symmetric(1, 2.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    assert!(!condition.evaluate(&parameters).unwrap());
    parameters.rename(1, "renamed").unwrap();
    assert_eq!(condition.parameter(), 1);
    assert_eq!(condition.threshold(), Quantity::new(2.0, Dimension::Length));
    assert!(!condition.evaluate(&parameters).unwrap());
}

#[test]
fn dimensions_must_match_without_hidden_unit_conversion() {
    let condition = condition(1, 2.0, Dimension::Length);
    let error = condition
        .evaluate(&parameters(1.0, Dimension::Angle))
        .unwrap_err();
    assert_eq!(
        error,
        FeatureSuppressionError::Dimension {
            parameter: 1,
            found: Dimension::Angle,
            expected: Dimension::Length
        }
    );
    assert!(
        error
            .to_string()
            .contains("parameter 1 dimension Angle: expected Length")
    );
    assert!(error.source().is_none());
}

const INVALID_THRESHOLD_INPUTS: [(f64, Dimension); 4] = [
    (f64::NAN, Dimension::Length),
    (f64::INFINITY, Dimension::Length),
    (1.0, Dimension::Boolean),
    (1.0, Dimension::Text),
];

#[test]
fn invalid_thresholds_are_refused_with_the_offending_quantity() {
    for (value, dimension) in INVALID_THRESHOLD_INPUTS {
        let threshold = Quantity::new(value, dimension);
        let error =
            FeatureSuppressionCondition::new(1, FeatureSuppressionComparison::Equal, threshold)
                .unwrap_err();
        assert!(matches!(
            error,
            FeatureSuppressionError::Threshold { parameter: 1, .. }
        ));
        let text = error.to_string();
        assert!(text.contains(&format!("threshold {threshold:?}")));
        assert!(text.contains("expected a finite numeric working-unit quantity"));
    }
}

#[test]
fn missing_and_failed_parameters_have_no_last_good_condition_fallback() {
    let mut parameters = parameters(1.0, Dimension::Unitless);
    let missing = condition(9, 2.0, Dimension::Unitless)
        .evaluate(&parameters)
        .unwrap_err();
    assert!(missing.to_string().contains('9'));
    assert!(missing.source().is_some());
    parameters.set_expression(1, "missing").unwrap();
    let failed = condition(1, 2.0, Dimension::Unitless)
        .evaluate(&parameters)
        .unwrap_err();
    assert!(matches!(failed, FeatureSuppressionError::Parameter(_)));
    assert!(failed.source().is_some());
}

#[test]
fn text_and_boolean_parameters_cannot_be_numeric_condition_inputs() {
    let selected = condition(1, 2.0, Dimension::Unitless);
    let mut text = ParameterCollection::new(UnitsOfMeasure::default());
    text.insert_text(1, "material", "steel").unwrap();
    assert!(matches!(
        selected.evaluate(&text),
        Err(FeatureSuppressionError::Parameter(_))
    ));
    let mut boolean = ParameterCollection::new(UnitsOfMeasure::default());
    boolean.insert_boolean(1, "enabled", true).unwrap();
    assert!(matches!(
        selected.evaluate(&boolean),
        Err(FeatureSuppressionError::Parameter(_))
    ));
}

#[test]
fn condition_edits_track_exact_recipe_bits() {
    let mut sequence = conditioned_sequence();
    for threshold in [0.0, -0.0] {
        let selected = condition(1, threshold, Dimension::Unitless);
        let changed = sequence
            .set_suppression_condition(FeatureId(1), Some(selected))
            .unwrap();
        assert!(changed.is_some());
    }
    let selected = condition(1, -0.0, Dimension::Unitless);
    assert_eq!(
        sequence
            .set_suppression_condition(FeatureId(1), Some(selected))
            .unwrap(),
        None
    );
}

#[test]
fn clearing_a_condition_retains_a_shared_definition_read() {
    let mut sequence = conditioned_sequence();
    let reads = FeatureReadFootprint::new([FeatureReadKey::Parameter(1)], false);
    sequence.replace(FeatureId(1), 99, reads).unwrap();
    sequence
        .set_suppression_condition(FeatureId(1), None)
        .unwrap();
    let feature = sequence.get(FeatureId(1)).unwrap();
    assert!(
        feature
            .reads()
            .keys()
            .contains(&FeatureReadKey::Parameter(1))
    );
    let repeated = sequence
        .set_suppression_condition(FeatureId(1), None)
        .unwrap();
    assert_eq!(repeated, None);
}

#[test]
fn replacing_a_condition_updates_only_its_own_read_key() {
    let mut sequence = conditioned_sequence();
    retarget_condition(&mut sequence);
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(1)]),
        None
    );
    let changed = sequence.replay_for_reads([FeatureReadKey::Parameter(9)]);
    assert_eq!(changed, Some(FeatureReplayRange { start: 0, end: 3 }));
}

fn retarget_condition(sequence: &mut FeatureSequence<u64>) {
    let selected = condition(9, 2.0, Dimension::Unitless);
    sequence
        .set_suppression_condition(FeatureId(1), Some(selected))
        .unwrap();
}

#[test]
fn definition_replacement_retains_the_condition_until_it_is_cleared() {
    let mut sequence = conditioned_sequence();
    retarget_condition(&mut sequence);
    sequence
        .replace(FeatureId(1), 99, FeatureReadFootprint::default())
        .unwrap();
    let feature = sequence.get(FeatureId(1)).unwrap();
    assert!(feature.suppression_condition().is_some());
    sequence
        .set_suppression_condition(FeatureId(1), None)
        .unwrap();
    assert_eq!(
        sequence.replay_for_reads([FeatureReadKey::Parameter(9)]),
        None
    );
    let unknown = sequence.set_suppression_condition(FeatureId(9), None);
    assert!(unknown.is_err());
}
