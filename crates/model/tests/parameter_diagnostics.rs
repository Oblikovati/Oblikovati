// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn unresolved() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "missing + 1 cm")
        .unwrap();
    parameters
        .insert_expression(2, "twice", USER_PARAM, "width * 2")
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn diagnostic_restore_keeps_failure_and_readers_blocked_until_real_recovery() {
    let mut parameters = unresolved();
    let health = parameters.get(1).unwrap().health.clone();
    let blocked = parameters.get(2).unwrap().health.clone();
    let band = Tolerance {
        type_: TOLERANCE_DEVIATION,
        upper: 0.2,
        lower: -0.1,
        ..Default::default()
    };
    parameters
        .restore_numeric_diagnostics(
            1,
            Some(Quantity::new(4.0, Dimension::Length)),
            band,
            MODEL_VALUE_UPPER,
        )
        .unwrap();
    assert_eq!(parameters.get(1).unwrap().health, health);
    assert_eq!(parameters.get(2).unwrap().health, blocked);
    assert!(parameters.model_value(1).is_err());
    assert_eq!(parameters.inspection_value(1).unwrap().unwrap().value, 4.2);
    assert!(parameters.get(2).unwrap().current_value().is_none());
    parameters
        .insert_expression(3, "missing", USER_PARAM, "6 cm")
        .unwrap();
    assert_eq!(parameters.model_value(1).unwrap().value, 7.2);
    assert_eq!(
        parameters.get(2).unwrap().current_value().unwrap().value,
        14.0
    );
}

#[test]
fn invalid_diagnostics_are_atomic_and_healthy_values_cannot_be_overwritten() {
    let mut parameters = unresolved();
    let before = parameters.get(1).unwrap().clone();
    for quantity in [
        Quantity::new(f64::NAN, Dimension::Length),
        Quantity::new(f64::INFINITY, Dimension::Length),
        Quantity::new(1.0, Dimension::Text),
    ] {
        assert!(
            parameters
                .restore_numeric_diagnostics(
                    1,
                    Some(quantity),
                    Tolerance::default(),
                    MODEL_VALUE_NOMINAL
                )
                .is_err()
        );
        assert_eq!(parameters.get(1).unwrap(), &before);
    }
    assert!(
        parameters
            .restore_numeric_diagnostics(1, None, Tolerance::default(), ModelValueType(999))
            .is_err()
    );
    assert!(
        parameters
            .restore_numeric_diagnostics(999, None, Tolerance::default(), MODEL_VALUE_NOMINAL)
            .is_err()
    );
    parameters.set_expression(1, "3 cm").unwrap();
    assert!(
        parameters
            .restore_numeric_diagnostics(
                1,
                Some(Quantity::new(9.0, Dimension::Length)),
                Tolerance::default(),
                MODEL_VALUE_NOMINAL
            )
            .is_err()
    );
    assert_eq!(parameters.model_value(1).unwrap().value, 3.0);
}

#[test]
fn restored_bands_and_selected_values_are_validated_before_adoption() {
    let mut parameters = unresolved();
    for band in [
        Tolerance {
            type_: ToleranceType(999),
            ..Default::default()
        },
        Tolerance {
            type_: TOLERANCE_DEVIATION,
            upper: f64::NAN,
            ..Default::default()
        },
        Tolerance {
            type_: TOLERANCE_DEVIATION,
            upper: f64::MAX,
            ..Default::default()
        },
    ] {
        let before = parameters.get(1).unwrap().clone();
        assert!(
            parameters
                .restore_numeric_diagnostics(
                    1,
                    Some(Quantity::new(f64::MAX, Dimension::Length)),
                    band,
                    MODEL_VALUE_UPPER
                )
                .is_err()
        );
        assert_eq!(parameters.get(1).unwrap(), &before);
    }
    parameters
        .restore_numeric_diagnostics(1, None, Tolerance::default(), MODEL_VALUE_NOMINAL)
        .unwrap();
    assert!(parameters.get(1).unwrap().last_good_value.is_none());
    assert!(parameters.get(1).unwrap().current_value().is_none());
}

#[test]
fn diagnostic_signed_zero_is_preserved_without_repairing_health() {
    let mut parameters = unresolved();
    parameters
        .restore_numeric_diagnostics(
            1,
            Some(Quantity::new(-0.0, Dimension::Length)),
            Tolerance::default(),
            MODEL_VALUE_NOMINAL,
        )
        .unwrap();
    let value = parameters.inspection_value(1).unwrap().unwrap();
    assert_eq!(value.value.to_bits(), (-0.0_f64).to_bits());
    assert!(parameters.get(1).unwrap().current_value().is_none());
}
