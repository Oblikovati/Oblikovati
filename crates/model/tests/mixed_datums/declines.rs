// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn an_axis_cannot_be_used_as_the_plane_driver_of_an_offset() {
    let mut snapshot = mixed_snapshot();
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Feature(FeatureId(2)),
        distance: WorkPlaneDistance::Constant(2.0),
    };
    snapshot
        .work_geometry
        .replace(FeatureId(3), definition, definition.reads())
        .unwrap();
    let evaluation = evaluate(&snapshot);
    assert!(matches!(
        evaluation.health[2].status,
        FeatureEvaluationStatus::Failed(WorkGeometryError::Kind { .. })
    ));
    assert!(evaluation.axis(FeatureId(2)).is_ok());
    assert!(evaluation.plane(FeatureId(3)).is_err());
}
#[test]
fn direct_axis_plane_query_and_invalid_saved_axis_keep_typed_causes() {
    let snapshot = mixed_snapshot();
    let definition = *snapshot
        .work_geometry
        .get(FeatureId(2))
        .unwrap()
        .definition();
    let error = definition
        .evaluate(&snapshot.parameters, &evaluate(&snapshot))
        .unwrap_err();
    assert!(matches!(error, WorkGeometryError::AxisInsteadOfPlane(_)));
    assert!(error.to_string().contains("expected a plane"));
    assert!(error.source().is_none());
    assert_invalid_saved_axis(definition);
}
fn assert_invalid_saved_axis(definition: WorkGeometryDefinition) {
    let mut recipe = definition.recipe();
    let oblikovati_model::feature::work_plane::WorkPlaneDefinitionRecipe::Axis(ref mut axis) =
        recipe.definition
    else {
        panic!("expected axis recipe");
    };
    axis.direction = [0.0; 3];
    let error = WorkGeometryDefinition::from_recipe(recipe).unwrap_err();
    assert!(matches!(error, WorkGeometryError::Axis(_)));
    assert!(error.source().is_some());
    assert!(error.to_string().contains("expected"));
}
#[test]
fn origin_plane_drivers_still_resolve_in_a_mixed_program() {
    let mut snapshot = mixed_snapshot();
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::YZ),
        distance: WorkPlaneDistance::Constant(4.0),
    };
    snapshot
        .work_geometry
        .replace(FeatureId(3), definition, definition.reads())
        .unwrap();
    assert_eq!(
        evaluate(&snapshot)
            .plane(FeatureId(3))
            .unwrap()
            .recipe()
            .origin,
        [4.0, 0.0, 0.0]
    );
}
