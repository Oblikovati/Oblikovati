// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchAxisReference, SketchOrientationDefinition},
    feature::{
        AuthoredFeature, FeatureEvaluationStatus, FeatureId, FeatureReadKey, FeatureSequence,
        work_axis::WorkAxisLine,
        work_geometry::{
            DatumLength, DatumLengthError, WorkAxisPoints, WorkGeometryDefinition,
            WorkGeometryError, WorkGeometryEvaluation, WorkPointReference,
        },
        work_plane::WorkPlaneResolver,
    },
    history::DocumentSnapshot,
    quantity::{Dimension, Quantity},
    sketch::{Sketch2D, SketchFrameAxis, SketchPlane},
};
use std::error::Error;

fn point_snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_value(
            1,
            "height",
            USER_PARAM,
            Quantity::new(4.0, Dimension::Length),
        )
        .unwrap();
    insert(&mut snapshot, 1, driven_point());
    let endpoint = WorkPointReference::Feature(FeatureId(1));
    let axis = through(WorkPointReference::Origin, endpoint);
    insert(&mut snapshot, 2, axis);
    let plane = WorkGeometryDefinition::Fixed(SketchPlane::xy());
    insert(&mut snapshot, 3, plane);
    snapshot
}
fn driven_point() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point([
        DatumLength::Constant(2.0),
        DatumLength::Parameter(1),
        DatumLength::Constant(0.0),
    ])
}
fn between(first: u64, second: u64) -> WorkGeometryDefinition {
    through(
        WorkPointReference::Feature(FeatureId(first)),
        WorkPointReference::Feature(FeatureId(second)),
    )
}
fn insert(snapshot: &mut DocumentSnapshot, id: u64, definition: WorkGeometryDefinition) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn through(first: WorkPointReference, second: WorkPointReference) -> WorkGeometryDefinition {
    WorkGeometryDefinition::AxisThroughPoints(WorkAxisPoints { first, second })
}
fn evaluate(snapshot: &DocumentSnapshot) -> WorkGeometryEvaluation {
    snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap()
}
fn point(position: [f64; 3]) -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point(position.map(DatumLength::Constant))
}
fn replace(snapshot: &mut DocumentSnapshot, id: u64, definition: WorkGeometryDefinition) {
    snapshot
        .work_geometry
        .replace(FeatureId(id), definition.clone(), definition.reads())
        .unwrap();
}
fn assert_axis_direction(evaluation: &WorkGeometryEvaluation, height: f64) {
    let axis = evaluation.axis(FeatureId(2)).unwrap();
    let length = 2.0f64.hypot(height);
    for (actual, expected) in axis
        .direction()
        .into_iter()
        .zip([2.0 / length, height / length, 0.0])
    {
        assert!((actual - expected).abs() < f64::EPSILON * 2.0);
    }
    assert_eq!(axis.origin(), [0.0; 3]);
}
#[test]
fn current_parameters_drive_points_axes_and_owned_sketch_orientation() {
    let mut snapshot = point_snapshot();
    add_oriented_sketch(&mut snapshot);
    assert_eq!(
        evaluate(&snapshot).point(FeatureId(1)).unwrap(),
        [2.0, 4.0, 0.0]
    );
    assert_axis_direction(&evaluate(&snapshot), 4.0);
    snapshot.parameters.set_expression(1, "8 cm").unwrap();
    let evaluation = evaluate(&snapshot);
    assert_axis_direction(&evaluation, 8.0);
    let plane = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(
        plane.recipe().y_axis,
        evaluation.axis(FeatureId(2)).unwrap().direction()
    );
}
fn add_oriented_sketch(snapshot: &mut DocumentSnapshot) {
    snapshot
        .sketches
        .add_on_work_plane(
            DocumentSketchId(1),
            "Oriented",
            FeatureId(3),
            Sketch2D::default(),
        )
        .unwrap();
    snapshot
        .sketches
        .set_orientation(DocumentSketchId(1), Some(axis_orientation()))
        .unwrap();
}
fn axis_orientation() -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference: SketchAxisReference::Feature(FeatureId(2)),
        sketch_axis: SketchFrameAxis::Y,
        reverse: false,
        origin: None,
    }
}

#[test]
fn point_and_axis_reads_are_exact_and_stable_under_kind_changes() {
    let snapshot = point_snapshot();
    assert_read_keys(&snapshot, 1, &[FeatureReadKey::Parameter(1)]);
    assert_read_keys(&snapshot, 2, &[FeatureReadKey::Feature(FeatureId(1))]);
    for error in kind_errors(&evaluate(&snapshot)) {
        assert!(matches!(error, WorkGeometryError::Kind(_)));
        assert!(error.to_string().contains("expected an available current"));
        assert!(error.source().is_none());
    }
}
fn assert_read_keys(snapshot: &DocumentSnapshot, id: u64, expected: &[FeatureReadKey]) {
    let keys = snapshot
        .work_geometry
        .get(FeatureId(id))
        .unwrap()
        .reads()
        .keys();
    assert_eq!(keys.iter().copied().collect::<Vec<_>>(), expected);
}
fn kind_errors(evaluation: &WorkGeometryEvaluation) -> [WorkGeometryError; 6] {
    [
        evaluation.plane(FeatureId(1)).unwrap_err(),
        evaluation.axis(FeatureId(1)).unwrap_err(),
        evaluation.point(FeatureId(2)).unwrap_err(),
        evaluation.point(FeatureId(3)).unwrap_err(),
        evaluation.plane(FeatureId(2)).unwrap_err(),
        evaluation.axis(FeatureId(3)).unwrap_err(),
    ]
}
#[test]
fn two_feature_points_retain_directed_order_and_replay_dependents() {
    let mut snapshot = point_snapshot();
    insert(&mut snapshot, 4, point([2.0, 4.0, 5.0]));
    insert(&mut snapshot, 5, between(1, 4));
    assert_eq!(
        evaluate(&snapshot).axis(FeatureId(5)).unwrap().direction(),
        [0.0, 0.0, 1.0]
    );
    replace(&mut snapshot, 5, between(4, 1));
    assert_eq!(
        evaluate(&snapshot).axis(FeatureId(5)).unwrap().direction(),
        [0.0, 0.0, -1.0]
    );
    assert!(snapshot.work_geometry.remove(FeatureId(4)).is_err());
}
#[test]
fn point_suppression_and_cutoff_cannot_return_a_stale_line() {
    let mut snapshot = point_snapshot();
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    let evaluation = evaluate(&snapshot);
    assert_suppressed_point_cascade(&evaluation);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(1), false)
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(1)).unwrap();
    assert!(evaluate(&snapshot).axis(FeatureId(2)).is_err());
    assert!(evaluate(&snapshot).point(FeatureId(1)).is_ok());
}
fn assert_suppressed_point_cascade(evaluation: &WorkGeometryEvaluation) {
    assert!(matches!(
        evaluation.health[1].status,
        FeatureEvaluationStatus::Failed(WorkGeometryError::Unavailable {
            feature: FeatureId(1),
            expected: "point"
        })
    ));
    assert!(evaluation.axis(FeatureId(2)).is_err());
    assert!(evaluation.plane(FeatureId(3)).is_ok());
}
#[test]
fn point_recipe_and_directed_references_require_program_version_four() {
    let snapshot = point_snapshot();
    let mut recipe = snapshot.work_geometry.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 4);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(snapshot.work_geometry.same_work_geometry_recipe(&restored));
    for version in [1, 2, 3] {
        recipe.version = version;
        let error = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap_err();
        assert!(error.to_string().contains("expected program version 4"));
        assert!(error.source().is_none());
    }
}
#[test]
fn last_sketch_removal_cleans_the_axis_and_its_construction_point_chain() {
    let mut snapshot = point_snapshot();
    for id in [1, 2, 3] {
        snapshot
            .work_geometry
            .set_work_plane_construction(FeatureId(id), true)
            .unwrap();
    }
    add_oriented_sketch(&mut snapshot);
    let report = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert_eq!(
        report.construction_datums,
        [FeatureId(3), FeatureId(2), FeatureId(1)]
    );
    assert!(snapshot.work_geometry.features().is_empty());
}

#[path = "point_datums/declines.rs"]
mod declines;
#[path = "point_datums/numerical.rs"]
mod numerical;
