// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn coincident_points_fail_locally_and_leave_independent_planes_available() {
    let mut snapshot = point_snapshot();
    replace(&mut snapshot, 1, point([0.0; 3]));
    let evaluation = evaluate(&snapshot);
    assert!(matches!(
        evaluation.health[1].status,
        FeatureEvaluationStatus::Failed(WorkGeometryError::Axis(_))
    ));
    assert!(evaluation.axis(FeatureId(2)).is_err());
    assert_eq!(evaluation.point(FeatureId(1)).unwrap(), [0.0; 3]);
    assert!(evaluation.plane(FeatureId(3)).is_ok());
}
#[test]
fn failed_coordinates_block_their_axis_and_retain_component_diagnostics() {
    let mut snapshot = point_snapshot();
    replace(&mut snapshot, 1, missing_coordinate());
    let evaluation = evaluate(&snapshot);
    let FeatureEvaluationStatus::Failed(WorkGeometryError::Coordinate(cause)) =
        &evaluation.health[0].status
    else {
        panic!("expected named coordinate failure");
    };
    assert_eq!(cause.component, 1);
    assert!(cause.to_string().contains("99"));
    assert!(cause.source().unwrap().source().is_some());
    assert!(matches!(
        evaluation.health[1].status,
        FeatureEvaluationStatus::Blocked { .. }
    ));
}
fn missing_coordinate() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point([
        DatumLength::Constant(0.0),
        DatumLength::Parameter(99),
        DatumLength::Constant(0.0),
    ])
}
#[test]
fn non_point_drivers_and_unknown_point_ids_are_named_declines() {
    let mut snapshot = point_snapshot();
    let driver = WorkPointReference::Feature(FeatureId(3));
    let axis = through(WorkPointReference::Origin, driver);
    insert(&mut snapshot, 4, axis);
    let evaluation = evaluate(&snapshot);
    assert!(matches!(
        evaluation.health[3].status,
        FeatureEvaluationStatus::Failed(WorkGeometryError::Kind(_))
    ));
    let error = evaluation.point(FeatureId(99)).unwrap_err();
    assert!(error.to_string().contains("99"));
    assert!(
        error
            .to_string()
            .contains("expected an available current point")
    );
    assert!(error.source().is_none());
}
#[test]
fn nonfinite_point_recipes_are_refused_before_adoption() {
    let definition = point([0.0, f64::INFINITY, 0.0]);
    let error = WorkGeometryDefinition::from_recipe(definition.recipe()).unwrap_err();
    assert!(matches!(error, WorkGeometryError::Coordinate(_)));
    assert!(error.to_string().contains("component 1"));
    assert!(error.to_string().contains("finite Length"));
    assert!(error.source().is_some());
    assert!(
        definition
            .evaluate(&point_snapshot().parameters, &evaluate(&point_snapshot()))
            .unwrap_err()
            .to_string()
            .contains("expected a plane frame")
    );
}
#[test]
fn coordinate_scalars_share_length_checks_and_exact_source_comparison() {
    let mut snapshot = point_snapshot();
    snapshot
        .parameters
        .insert_value(2, "angle", USER_PARAM, Quantity::new(0.5, Dimension::Angle))
        .unwrap();
    let error = DatumLength::Parameter(2)
        .resolve(&snapshot.parameters)
        .unwrap_err();
    assert!(matches!(error, DatumLengthError::Quantity { .. }));
    assert!(error.to_string().contains("finite Length"));
    assert!(error.source().is_none());
    assert!(DatumLength::Constant(f64::NAN).validate_source().is_err());
    assert!(!point([0.0; 3]).same_recipe(point([-0.0, 0.0, 0.0])));
    assert!(!DatumLength::Constant(1.0).same_recipe(DatumLength::Parameter(1)));
    assert!(DatumLength::Parameter(1).same_recipe(DatumLength::Parameter(1)));
    assert!(!DatumLength::Parameter(1).same_recipe(DatumLength::Parameter(2)));
}
