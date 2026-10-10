// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    dependency::DependencyError,
    expression::ExpressionError,
    parameter::{ParameterFailure, kind_name},
    quantity::{Dimension, Quantity},
};

#[test]
fn canonical_parameter_kind_names_cover_every_supported_category() {
    for (kind, name) in [
        (USER_PARAM, "user"),
        (MODEL_PARAM, "model"),
        (TABLE_PARAM, "table"),
        (REFERENCE_PARAM, "reference"),
        (DERIVED_PARAM, "derived"),
    ] {
        assert_eq!(kind_name(kind), Some(name));
    }
    assert_eq!(kind_name(ParameterKind(255)), None);
}

#[test]
fn parameter_failure_messages_identify_the_fault_and_expected_state() {
    let failures: [(ParameterFailure<u64>, &str, &str); 7] = [
        (
            ParameterFailure::Unresolved(vec!["width".into()]),
            "width",
            "resolved",
        ),
        (
            ParameterFailure::ExternalLink {
                source_document: "source.opd".into(),
                source_parameter: "width".into(),
            },
            "source.opd",
            "available",
        ),
        (
            ParameterFailure::Cycle(DependencyError::Cycle { id: 1, driver: 2 }),
            "1",
            "acyclic",
        ),
        (
            ParameterFailure::Evaluation(ExpressionError::Reference {
                name: "width".into(),
                expected: "a numeric value",
            }),
            "width",
            "numeric",
        ),
        (ParameterFailure::BlockedBy(vec![3]), "3", "healthy"),
        (
            ParameterFailure::NonFinite(Quantity::new(f64::INFINITY, Dimension::Length)),
            "inf",
            "finite",
        ),
        (ParameterFailure::NonNumeric(vec![4]), "4", "numeric"),
    ];
    for (failure, offending, expected) in failures {
        let message = failure.to_string();
        assert!(message.contains(offending), "{message}");
        assert!(message.contains(expected), "{message}");
    }
}

#[test]
fn dimension_display_names_cover_supported_modes_and_reject_unknown_ids() {
    use oblikovati_model::parameter::dimension_display_name;
    for (mode, name) in [
        (DIMENSION_DISPLAY_VALUE, "value"),
        (DIMENSION_DISPLAY_NAME, "name"),
        (DIMENSION_DISPLAY_EXPRESSION, "expression"),
        (DIMENSION_DISPLAY_TOLERANCE, "tolerance"),
        (DIMENSION_DISPLAY_PRECISE_VALUE, "preciseValue"),
    ] {
        assert_eq!(dimension_display_name(mode), Some(name));
    }
    assert_eq!(dimension_display_name(DimensionDisplayType(255)), None);
}

#[test]
fn member_vocabulary_preserves_display_selection_and_action_spellings() {
    use oblikovati_model::parameter::vocabulary::*;
    for (mode, name) in [
        (DISPLAY_FORMAT_DECIMAL, "decimal"),
        (DISPLAY_FORMAT_FRACTIONAL, "fractional"),
        (DISPLAY_FORMAT_ARCHITECTURAL, "architectural"),
    ] {
        assert_eq!(display_name(mode), Some(name));
    }
    for (mode, name) in [
        (MODEL_VALUE_NOMINAL, "nominal"),
        (MODEL_VALUE_LOWER, "lower"),
        (MODEL_VALUE_UPPER, "upper"),
        (MODEL_VALUE_MEDIAN, "median"),
    ] {
        assert_eq!(selection_name(mode), Some(name));
    }
    assert_eq!(display_name(ParameterDisplayFormat(255)), None);
    assert_eq!(selection_name(ModelValueType(255)), None);
    assert_eq!(action_names(ACTION_NONE), Vec::<String>::new());
    assert_eq!(
        action_names(ActionType(
            ACTION_EDIT.0 | ACTION_RENAME.0 | ACTION_DELETE.0
        )),
        ["edit", "rename", "delete"]
    );
}

#[test]
fn inspection_does_not_make_failed_parameters_usable_for_feature_evaluation() {
    use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_MEDIAN)
        .unwrap();
    let expected = parameters.model_value(1).unwrap();
    parameters.set_expression(1, "missing").unwrap();
    assert_eq!(parameters.inspection_value(1).unwrap(), Some(expected));
    assert!(parameters.model_value(1).is_err());
    assert!(parameters.inspection_value(99).is_err());
    parameters.insert_text(2, "material", "steel").unwrap();
    assert_eq!(parameters.inspection_value(2).unwrap(), None);
}

#[test]
fn dimension_display_parser_accepts_only_canonical_names() {
    use oblikovati_model::parameter::parse_dimension_display;
    for (name, mode) in [
        ("value", DIMENSION_DISPLAY_VALUE),
        ("name", DIMENSION_DISPLAY_NAME),
        ("expression", DIMENSION_DISPLAY_EXPRESSION),
        ("tolerance", DIMENSION_DISPLAY_TOLERANCE),
        ("preciseValue", DIMENSION_DISPLAY_PRECISE_VALUE),
    ] {
        assert_eq!(parse_dimension_display(name), Some(mode));
    }
    for name in ["", "Value", "invalid"] {
        assert_eq!(parse_dimension_display(name), None);
    }
}
