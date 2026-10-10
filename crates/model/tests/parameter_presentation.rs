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
