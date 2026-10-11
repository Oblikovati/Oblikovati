// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchAxisReference, SketchOrientationDefinition},
    feature::{
        AuthoredFeature, FeatureEvaluationStatus, FeatureId, FeatureSequence,
        work_axis::WorkAxisLine,
        work_geometry::{WorkGeometryDefinition, WorkGeometryError},
        work_plane::{WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference, WorkPlaneResolver},
    },
    history::DocumentSnapshot,
    sketch::{Sketch2D, SketchFrameAxis, SketchPlane},
};
use std::error::Error;

fn add_axis(snapshot: &mut DocumentSnapshot, id: u64, direction: [f64; 3]) {
    let line = WorkAxisLine::new([0.0; 3], direction).unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_axis(FeatureId(id), "Spindle", line))
        .unwrap();
}
fn mixed_snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    let plane = WorkGeometryDefinition::Fixed(SketchPlane::xy());
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(FeatureId(1), "Base", plane))
        .unwrap();
    add_axis(&mut snapshot, 2, [1.0, 0.0, 0.0]);
    let offset = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Feature(FeatureId(1)),
        distance: WorkPlaneDistance::Constant(2.0),
    };
    snapshot
        .work_geometry
        .append(AuthoredFeature::work_plane(FeatureId(3), "Offset", offset))
        .unwrap();
    snapshot
}
fn orient_sketch(snapshot: &mut DocumentSnapshot, id: u64, axis: u64) {
    let id = DocumentSketchId(id);
    snapshot
        .sketches
        .add_on_work_plane(id, "Sketch", FeatureId(3), Sketch2D::default())
        .unwrap();
    let definition = SketchOrientationDefinition {
        reference: SketchAxisReference::Feature(FeatureId(axis)),
        sketch_axis: SketchFrameAxis::Y,
        reverse: false,
        origin: None,
    };
    snapshot
        .sketches
        .set_orientation(id, Some(definition))
        .unwrap();
}
fn evaluate(
    snapshot: &DocumentSnapshot,
) -> oblikovati_model::feature::work_geometry::WorkGeometryEvaluation {
    snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap()
}
#[test]
fn one_program_produces_current_planes_and_axes_with_ordered_health() {
    let snapshot = mixed_snapshot();
    let evaluation = evaluate(&snapshot);
    assert_ordered_health(&evaluation);
    assert_eq!(
        evaluation.axis(FeatureId(2)).unwrap().direction(),
        [1.0, 0.0, 0.0]
    );
    assert_eq!(
        evaluation.plane(FeatureId(3)).unwrap().recipe().origin,
        [0.0, 0.0, 2.0]
    );
}
fn assert_ordered_health(
    evaluation: &oblikovati_model::feature::work_geometry::WorkGeometryEvaluation,
) {
    assert_eq!(
        evaluation
            .health
            .iter()
            .map(|record| record.feature)
            .collect::<Vec<_>>(),
        [FeatureId(1), FeatureId(2), FeatureId(3)]
    );
    assert!(
        evaluation
            .health
            .iter()
            .all(|record| matches!(record.status, FeatureEvaluationStatus::Healthy))
    );
}
#[test]
fn owned_orientation_changes_when_its_axis_source_is_replaced() {
    let mut snapshot = mixed_snapshot();
    orient_sketch(&mut snapshot, 1, 2);
    let first = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(first.recipe().y_axis, [1.0, 0.0, 0.0]);
    let axis = WorkAxisLine::new([0.0; 3], [0.0, 1.0, 0.0]).unwrap();
    let definition = WorkGeometryDefinition::Axis(axis);
    snapshot
        .work_geometry
        .replace(FeatureId(2), definition, definition.reads())
        .unwrap();
    let next = snapshot.resolve_sketch_plane(DocumentSketchId(1)).unwrap();
    assert_eq!(next.recipe().y_axis, [0.0, 1.0, 0.0]);
    assert_eq!(next.recipe().origin, [0.0, 0.0, 2.0]);
}
#[test]
fn kind_mismatches_and_unavailable_datums_are_named_declines() {
    let snapshot = mixed_snapshot();
    let evaluation = evaluate(&snapshot);
    for (expected, actual, error) in [
        ("plane", "axis", evaluation.plane(FeatureId(2)).unwrap_err()),
        ("axis", "plane", evaluation.axis(FeatureId(1)).unwrap_err()),
    ] {
        assert!(matches!(error, WorkGeometryError::Kind { .. }));
        assert!(error.to_string().contains(expected));
        assert!(error.to_string().contains(actual));
        assert!(error.source().is_none());
    }
    for error in [
        evaluation.axis(FeatureId(99)).unwrap_err(),
        evaluation.plane(FeatureId(99)).unwrap_err(),
    ] {
        assert!(error.to_string().contains("99"));
    }
}
fn set_axis_suppression(snapshot: &mut DocumentSnapshot, suppressed: bool) {
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(2), suppressed)
        .unwrap();
}
#[test]
fn suppression_and_cutoff_never_return_a_stale_axis_or_host_plane() {
    let mut snapshot = mixed_snapshot();
    orient_sketch(&mut snapshot, 1, 2);
    set_axis_suppression(&mut snapshot, true);
    assert!(snapshot.resolve_sketch_plane(DocumentSketchId(1)).is_err());
    assert!(matches!(
        evaluate(&snapshot).health[1].status,
        FeatureEvaluationStatus::Suppressed
    ));
    set_axis_suppression(&mut snapshot, false);
    snapshot.work_geometry.set_end_of_part(Some(2)).unwrap();
    assert!(snapshot.resolve_sketch_plane(DocumentSketchId(1)).is_err());
    snapshot.work_geometry.set_end_of_part(None).unwrap();
    assert!(snapshot.resolve_sketch_plane(DocumentSketchId(1)).is_ok());
}
#[test]
fn mixed_recipe_requires_version_three_and_compares_exact_axis_sources() {
    let snapshot = mixed_snapshot();
    let mut recipe = snapshot.work_geometry.work_geometry_recipe().unwrap();
    assert_eq!(recipe.version, 3);
    let restored = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap();
    assert!(snapshot.work_geometry.same_work_geometry_recipe(&restored));
    for version in [1, 2] {
        recipe.version = version;
        let error = FeatureSequence::from_work_geometry_recipe(&recipe).unwrap_err();
        assert!(error.to_string().contains("expected program version 3"));
        assert!(error.source().is_none());
    }
}
#[test]
fn construction_axis_and_plane_are_removed_with_the_last_orientated_sketch() {
    let mut snapshot = mixed_snapshot();
    for id in [2, 3] {
        snapshot
            .work_geometry
            .set_work_plane_construction(FeatureId(id), true)
            .unwrap();
    }
    orient_sketch(&mut snapshot, 1, 2);
    orient_sketch(&mut snapshot, 2, 2);
    let first = snapshot.delete_sketch(DocumentSketchId(1)).unwrap();
    assert!(first.construction_datums.is_empty());
    let report = snapshot.delete_sketch(DocumentSketchId(2)).unwrap();
    assert_eq!(report.construction_datums, [FeatureId(3), FeatureId(2)]);
    assert_eq!(snapshot.work_geometry.features()[0].id(), FeatureId(1));
}

#[path = "mixed_datums/declines.rs"]
mod declines;
