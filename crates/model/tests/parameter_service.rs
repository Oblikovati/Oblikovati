// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_VALUE_UPPER, USER_PARAM};
use oblikovati_model::{
    expression::ExpressionError,
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

#[test]
fn service_reads_live_nominals_without_edits_or_model_tolerance_offsets() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(1, 1.0, -1.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    assert_eq!(
        parameters.evaluate_expression("width * 2").unwrap(),
        Quantity::new(8.0, Dimension::Length)
    );
    assert_eq!(parameters.get(1), Some(&before));
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "7 cm").unwrap();
    assert_eq!(
        parameters.evaluate_expression("width / 2").unwrap().value,
        3.5
    );
}

#[test]
fn service_converts_micro_units_in_document_working_coordinates() {
    for scale in [1.0, 1e-4, 100.0] {
        let units = UnitsOfMeasure::default().with_working_scale(scale).unwrap();
        let parameters = ParameterCollection::<u64>::new(units);
        for (source, expected) in [
            ("1 µm", 1e-4),
            ("1 um", 1e-4),
            ("2500 nm", 2500.0 * 1e-7),
            ("1000 pm", 1000.0 * 1e-10),
        ] {
            assert_eq!(
                parameters.evaluate_expression(source).unwrap(),
                Quantity::new(expected / scale, Dimension::Length)
            );
        }
    }
}

#[test]
fn service_reports_unusable_references_and_does_not_create_parameters() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "material", "steel").unwrap();
    parameters.insert_boolean(2, "vented", true).unwrap();
    parameters
        .insert_expression(3, "sick", USER_PARAM, "missing")
        .unwrap();
    parameters.drain_changed();
    for source in ["material", "vented", "sick", "missing"] {
        assert!(matches!(
            parameters.evaluate_expression(source),
            Err(ExpressionError::Reference { .. })
        ));
    }
    for source in ["1 +", "1 cm + 1 rad", "1 / 0"] {
        assert!(parameters.evaluate_expression(source).is_err());
    }
    assert_eq!(parameters.ids(), &[1, 2, 3]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn nonfinite_service_results_are_explicit_and_keep_authored_source() {
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    for source in ["1e308 * 2", "sqrt(-1)", "exp(1000)"] {
        let error = parameters.evaluate_expression(source).unwrap_err();
        assert!(matches!(error, ExpressionError::NonFinite { .. }));
        let message = error.to_string();
        assert!(message.contains(source));
        assert!(message.contains("finite numeric result"));
        assert_eq!(
            parameters.qualify_authored(source, Dimension::Length),
            source
        );
    }
}
