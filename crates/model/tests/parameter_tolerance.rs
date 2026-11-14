// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn parameter(value: f64, dimension: Dimension) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(1, "dimension", USER_PARAM, Quantity::new(value, dimension))
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn model_selections_match_go_values_and_do_not_change_expression_nominals() {
    let mut parameters = parameter(10.0, Dimension::Length);
    parameters
        .insert_expression(2, "twice", USER_PARAM, "2 * dimension")
        .unwrap();
    parameters.drain_changed();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    for (selection, expected) in [
        (MODEL_VALUE_NOMINAL, 10.0_f64),
        (MODEL_VALUE_UPPER, 10.2),
        (MODEL_VALUE_LOWER, 9.9),
        (MODEL_VALUE_MEDIAN, 10.05),
    ] {
        parameters.set_model_value_type(1, selection).unwrap();
        assert_eq!(
            parameters.model_value(1).unwrap().value.to_bits(),
            expected.to_bits()
        );
        assert_eq!(
            parameters.get(1).unwrap().current_value().unwrap().value,
            10.0
        );
        assert_eq!(
            parameters.get(2).unwrap().current_value().unwrap().value,
            20.0
        );
    }
    assert_eq!(parameters.drain_changed(), [1]);
}

#[test]
fn nominal_selection_preserves_signed_zero_bit_for_bit() {
    let parameters = parameter(-0.0, Dimension::Length);
    assert_eq!(
        parameters.model_value(1).unwrap().value.to_bits(),
        (-0.0_f64).to_bits()
    );
}

#[test]
fn symmetric_limits_and_bandless_operations_store_canonical_tolerances() {
    let mut parameters = parameter(10.0, Dimension::Length);
    parameters.set_symmetric(1, 0.05).unwrap();
    assert_eq!(
        parameters.get(1).unwrap().tolerance,
        Tolerance {
            type_: TOLERANCE_SYMMETRIC,
            upper: 0.05,
            lower: -0.05,
            ..Tolerance::default()
        }
    );
    parameters.set_limits(1, 10.3, 9.8).unwrap();
    assert_eq!(
        parameters.get(1).unwrap().tolerance,
        Tolerance {
            type_: TOLERANCE_LIMITS_STACKED,
            upper: 10.3 - 10.0,
            lower: 9.8 - 10.0,
            ..Tolerance::default()
        }
    );
    for kind in [
        TOLERANCE_BASIC,
        TOLERANCE_REFERENCE,
        TOLERANCE_MIN,
        TOLERANCE_MAX,
    ] {
        parameters.set_bandless(1, kind).unwrap();
        assert_eq!(
            parameters.get(1).unwrap().tolerance,
            Tolerance {
                type_: kind,
                ..Tolerance::default()
            }
        );
    }
    parameters.set_bandless(1, TOLERANCE_DEFAULT).unwrap();
    assert_eq!(parameters.get(1).unwrap().tolerance, Tolerance::default());
}

#[test]
fn malformed_bands_and_unknown_selections_are_atomic() {
    let mut parameters = parameter(10.0, Dimension::Length);
    let before = parameters.get(1).unwrap().clone();
    for (upper, lower) in [
        (-0.1, 0.1),
        (f64::NAN, 0.0),
        (1.0, f64::NEG_INFINITY),
        (f64::INFINITY, 0.0),
    ] {
        let error = parameters.set_deviation(1, upper, lower).unwrap_err();
        assert!(error.to_string().contains("expected finite"));
        assert_eq!(parameters.get(1), Some(&before));
    }
    assert!(parameters.set_symmetric(1, -0.1).is_err());
    assert!(parameters.set_limits(1, 9.0, 11.0).is_err());
    assert!(parameters.set_bandless(1, TOLERANCE_SYMMETRIC).is_err());
    assert!(
        parameters
            .set_model_value_type(1, ModelValueType(7))
            .is_err()
    );
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn nonnumeric_and_failed_parameters_cannot_supply_model_values() {
    for dimension in [Dimension::Boolean, Dimension::Text] {
        let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
        if dimension == Dimension::Text {
            parameters.insert_text(1, "literal", "text").unwrap();
        } else {
            parameters.insert_boolean(1, "literal", false).unwrap();
        }
        assert!(parameters.model_value(1).is_err());
        assert!(parameters.set_symmetric(1, 0.1).is_err());
        assert!(
            parameters
                .set_model_value_type(1, MODEL_VALUE_UPPER)
                .is_err()
        );
    }
    let mut parameters = parameter(1.0, Dimension::Length);
    parameters.set_expression(1, "missing").unwrap();
    assert!(parameters.model_value(1).is_err());
    assert!(parameters.set_tolerance_default(1).is_err());
    assert!(parameters.model_value(999).is_err());
}

#[test]
fn overflow_is_reported_when_the_selected_model_value_is_read() {
    let mut parameters = parameter(f64::MAX, Dimension::Length);
    parameters.set_deviation(1, f64::MAX, 0.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    assert!(
        parameters
            .model_value(1)
            .unwrap_err()
            .to_string()
            .contains("finite selected model value")
    );
    parameters
        .set_model_value_type(1, MODEL_VALUE_NOMINAL)
        .unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, f64::MAX);
}

#[test]
fn tolerance_edits_are_noops_when_unchanged_and_bands_follow_nominal_edits() {
    let mut parameters = parameter(10.0, Dimension::Length);
    parameters
        .set_model_value_type(1, MODEL_VALUE_NOMINAL)
        .unwrap();
    parameters.set_tolerance_default(1).unwrap();
    assert!(parameters.drain_changed().is_empty());
    parameters.set_limits(1, 11.0, 9.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters
        .set_value(1, Quantity::new(20.0, Dimension::Length))
        .unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 21.0);
    parameters.drain_changed();
    parameters.set_limits(1, 21.0, 19.0).unwrap();
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn saved_tolerance_validation_is_atomic_and_does_not_resolve_fit_labels() {
    let mut parameters = parameter(10.0, Dimension::Length);
    let saved = Tolerance {
        type_: TOLERANCE_LIMITS_FITS_SHOW_SIZE,
        upper: 0.5,
        lower: -0.25,
        hole_tolerance: "external fit class".into(),
        shaft_tolerance: "retained label".into(),
    };
    parameters.set_tolerance(1, saved.clone()).unwrap();
    parameters.drain_changed();
    parameters.set_tolerance(1, saved.clone()).unwrap();
    assert!(parameters.drain_changed().is_empty());
    let invalid = Tolerance {
        type_: ToleranceType(-1),
        ..saved.clone()
    };
    assert!(parameters.set_tolerance(1, invalid).is_err());
    assert!(
        parameters
            .set_tolerance(
                1,
                Tolerance {
                    upper: f64::NAN,
                    ..saved.clone()
                }
            )
            .is_err()
    );
    assert!(parameters.set_tolerance(99, saved.clone()).is_err());
    assert_eq!(parameters.get(1).unwrap().tolerance, saved);
    assert!(parameters.drain_changed().is_empty());
}
