// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn missing_zero_and_wrong_kind_sources_name_the_current_decline() {
    let snapshot = snapshot();
    for id in [0, 2, 4, 99] {
        let reference = WorkPointReference::Feature(FeatureId(id));
        let error = snapshot
            .project_datum_point_into_sketch(SKETCH, reference)
            .unwrap_err();
        assert!(
            matches!(error, DocumentSketchPlacementError::Point { reference: actual, .. } if actual == reference)
        );
        assert!(error.to_string().contains(&format!("FeatureId({id})")));
        assert!(error.source().is_some());
    }
}
#[test]
fn failed_point_driver_declines_without_affecting_independent_origin_projection() {
    let mut snapshot = snapshot();
    project(&snapshot, SOURCE);
    snapshot.parameters.set_expression(1, "missing").unwrap();
    assert_source_decline(&snapshot);
    assert_eq!(
        project(&snapshot, WorkPointReference::Origin).local_position,
        [-10.0, -20.0]
    );
    snapshot.parameters.set_expression(1, "6 cm").unwrap();
    assert_eq!(project(&snapshot, SOURCE).local_position, [-8.0, -14.0]);
}
#[test]
fn suppressed_or_rolled_back_sources_do_not_return_last_good_coordinates() {
    let mut snapshot = snapshot();
    project(&snapshot, SOURCE);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    assert_source_decline(&snapshot);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), false)
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(0)).unwrap();
    assert_placement_decline(&snapshot);
}
fn assert_source_decline(snapshot: &DocumentSnapshot) {
    let result = snapshot.project_datum_point_into_sketch(SKETCH, SOURCE);
    assert!(matches!(
        result,
        Err(DocumentSketchPlacementError::Point { .. })
    ));
}
#[test]
fn host_and_orientation_failures_remain_placement_errors() {
    let mut snapshot = snapshot();
    orient(&mut snapshot, false);
    let axis = WorkAxisLine::new([0.0; 3], [0.0, 0.0, 1.0]).unwrap();
    replace(&mut snapshot, 4, WorkGeometryDefinition::Axis(axis));
    assert_placement_decline(&snapshot);
    snapshot.sketches.set_orientation(SKETCH, None).unwrap();
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    assert_placement_decline(&snapshot);
}
fn assert_placement_decline(snapshot: &DocumentSnapshot) {
    let error = snapshot
        .project_datum_point_into_sketch(SKETCH, SOURCE)
        .unwrap_err();
    assert!(matches!(
        error,
        DocumentSketchPlacementError::Plane { sketch: SKETCH, .. }
    ));
    assert!(error.source().is_some());
}
#[test]
fn absent_sketch_is_an_explicit_error() {
    let snapshot = document_snapshot::empty_snapshot();
    assert!(matches!(
        snapshot.project_datum_point_into_sketch(SKETCH, SOURCE),
        Err(DocumentSketchPlacementError::Sketch(_))
    ));
}
#[test]
fn incomplete_read_footprint_is_an_explicit_replay_error() {
    let mut snapshot = document_snapshot::empty_snapshot();
    let definition = WorkGeometryDefinition::Point([DatumLength::Parameter(1); 3]);
    let entry = AuthoredFeature::new(
        FeatureId(1),
        "Undeclared",
        definition,
        FeatureReadFootprint::default(),
    );
    snapshot.work_geometry.append(entry).unwrap();
    snapshot
        .sketches
        .add(SKETCH, "Fixed", SketchPlane::xy(), Sketch2D::default())
        .unwrap();
    let error = snapshot
        .project_datum_point_into_sketch(SKETCH, SOURCE)
        .unwrap_err();
    assert!(matches!(error, DocumentSketchPlacementError::Replay(_)));
    assert!(error.source().is_some());
}
#[test]
fn projection_overflow_is_refused_without_mutation() {
    let mut snapshot = snapshot();
    replace(
        &mut snapshot,
        1,
        WorkGeometryDefinition::Point([DatumLength::Constant(f64::MAX); 3]),
    );
    replace(
        &mut snapshot,
        2,
        WorkGeometryDefinition::Fixed(frame([-f64::MAX; 3])),
    );
    let before = snapshot.sketches.clone();
    let error = snapshot
        .project_datum_point_into_sketch(SKETCH, SOURCE)
        .unwrap_err();
    assert!(matches!(error, DocumentSketchPlacementError::Sketch(_)));
    assert!(error.to_string().contains("expected 3 finite"));
    assert!(before.same_evaluated_state(&snapshot.sketches));
}
#[test]
fn origin_on_fixed_sketch_has_no_datum_dependency() {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(SKETCH, "Fixed", SketchPlane::yz(), Sketch2D::default())
        .unwrap();
    let result = project(&snapshot, WorkPointReference::Origin);
    assert_eq!(result.local_position, [0.0; 2]);
    assert_eq!(
        result.reads,
        FeatureReadFootprint::new([FeatureReadKey::Sketch(7)], false)
    );
}
#[test]
fn source_kind_failure_keeps_the_typed_geometry_cause() {
    let snapshot = snapshot();
    let error = snapshot
        .project_datum_point_into_sketch(SKETCH, WorkPointReference::Feature(FeatureId(2)))
        .unwrap_err();
    let cause = error
        .source()
        .unwrap()
        .downcast_ref::<WorkGeometryError>()
        .unwrap();
    assert!(matches!(cause, WorkGeometryError::Kind(_)));
    assert!(
        error
            .to_string()
            .contains("expected an available current point")
    );
}
