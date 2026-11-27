// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};

fn parameters() -> ParameterCollection<u64> {
    let mut values = ParameterCollection::new(UnitsOfMeasure::default());
    values
        .insert_expression(1, "a", USER_PARAM, "1 mm")
        .unwrap();
    values.insert_expression(2, "b", USER_PARAM, "a").unwrap();
    values
}
#[test]
fn simultaneous_changes_do_not_create_a_transient_cycle() {
    let mut values = parameters();
    values
        .set_expressions(&[(1, "b".into()), (2, "2 mm".into())])
        .unwrap();
    assert_eq!(values.get(1).unwrap().source, "b");
    assert_eq!(values.get(2).unwrap().source, "2 mm");
    assert_eq!(
        values.get(1).unwrap().current_value(),
        values.get(2).unwrap().current_value()
    );
}
#[test]
fn invalid_or_duplicate_changes_leave_the_collection_unchanged() {
    let mut values = parameters();
    assert!(
        values
            .set_expressions(&[(1, "missing".into()), (2, "2 mm".into())])
            .is_err()
    );
    assert_eq!(values.get(1).unwrap().source, "1 mm");
    assert_eq!(values.get(2).unwrap().source, "a");
    assert!(
        values
            .set_expressions(&[(1, "2 mm".into()), (1, "3 mm".into())])
            .is_err()
    );
    assert_eq!(values.get(1).unwrap().source, "1 mm");
}

#[test]
fn detailed_batch_error_identifies_invalid_parameter_without_mutation() {
    let mut values = parameters();
    let error = values
        .set_expressions_detailed(&[(1, "2 mm".into()), (2, "(".into())])
        .unwrap_err();
    assert_eq!(error.parameter, 2);
    assert_eq!(values.get(1).unwrap().source, "1 mm");
    assert_eq!(values.get(2).unwrap().source, "a");
}

#[test]
fn detailed_batch_reports_the_cross_parameter_cycle() {
    use oblikovati_model::{dependency::DependencyError, parameter::ParameterError};
    let mut values = parameters();
    let error = values
        .set_expressions_detailed(&[(1, "b".into()), (2, "a".into())])
        .unwrap_err();
    assert!(matches!(
        error.cause,
        ParameterError::Dependency(DependencyError::Cycle {
            id: 1 | 2,
            driver: 1 | 2
        })
    ));
    assert_eq!(values.get(1).unwrap().source, "1 mm");
    assert_eq!(values.get(2).unwrap().source, "a");
}
