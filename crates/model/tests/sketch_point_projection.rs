// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchAxisReference, SketchOrientationDefinition},
    feature::{
        AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_axis::WorkAxisLine,
        work_geometry::{
            DatumLength, WorkGeometryDefinition, WorkGeometryError, WorkPointReference,
        },
    },
    history::{DocumentSketchPlacementError, DocumentSnapshot, ResolvedSketchPointProjection},
    sketch::{Sketch2D, SketchFrameAxis, SketchPlane},
};
use std::error::Error;
const SKETCH: DocumentSketchId = DocumentSketchId(7);
const SOURCE: WorkPointReference = WorkPointReference::Feature(FeatureId(1));

fn snapshot() -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "height", USER_PARAM, "4 cm")
        .unwrap();
    insert(&mut snapshot, 1, driven_point());
    insert(
        &mut snapshot,
        2,
        WorkGeometryDefinition::Fixed(frame([10.0, 20.0, 30.0])),
    );
    insert_axis(&mut snapshot, [0.0, 1.0, 0.0]);
    snapshot
        .sketches
        .add_on_work_plane(SKETCH, "Projected", FeatureId(2), Sketch2D::default())
        .unwrap();
    snapshot
}
fn driven_point() -> WorkGeometryDefinition {
    WorkGeometryDefinition::Point([
        DatumLength::Constant(2.0),
        DatumLength::Parameter(1),
        DatumLength::Constant(9.0),
    ])
}
fn frame(origin: [f64; 3]) -> SketchPlane {
    SketchPlane::new(origin, [1.0, 0.0, 0.0], [0.0, 1.0, 0.0], Default::default()).unwrap()
}
fn insert(snapshot: &mut DocumentSnapshot, id: u64, definition: WorkGeometryDefinition) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(id), "Datum", definition))
        .unwrap();
}
fn insert_axis(snapshot: &mut DocumentSnapshot, direction: [f64; 3]) {
    let line = WorkAxisLine::new([0.0; 3], direction).unwrap();
    insert(snapshot, 4, WorkGeometryDefinition::Axis(line));
}
fn replace(snapshot: &mut DocumentSnapshot, id: u64, definition: WorkGeometryDefinition) {
    snapshot
        .work_geometry
        .replace(FeatureId(id), definition.clone(), definition.reads())
        .unwrap();
}
fn project(
    snapshot: &DocumentSnapshot,
    reference: WorkPointReference,
) -> ResolvedSketchPointProjection {
    snapshot
        .project_datum_point_into_sketch(SKETCH, reference)
        .unwrap()
}
fn orient(snapshot: &mut DocumentSnapshot, reverse: bool) {
    snapshot
        .sketches
        .set_orientation(
            SKETCH,
            Some(SketchOrientationDefinition {
                reference: SketchAxisReference::Feature(FeatureId(4)),
                sketch_axis: SketchFrameAxis::X,
                reverse,
                origin: None,
            }),
        )
        .unwrap();
}

#[test]
fn projection_reads_current_point_and_host_without_a_solve_or_mutation() {
    let mut snapshot = snapshot();
    let before = snapshot.sketches.clone();
    assert_eq!(project(&snapshot, SOURCE).local_position, [-8.0, -16.0]);
    snapshot.parameters.set_expression(1, "8 cm").unwrap();
    let result = project(&snapshot, SOURCE);
    assert_eq!(result.model_position, [2.0, 8.0, 9.0]);
    assert_eq!(result.local_position, [-8.0, -12.0]);
    assert!(before.same_evaluated_state(&snapshot.sketches));
    replace(
        &mut snapshot,
        2,
        WorkGeometryDefinition::Fixed(frame([1.0, 3.0, 5.0])),
    );
    assert_eq!(project(&snapshot, SOURCE).local_position, [1.0, 5.0]);
}
#[test]
fn automatic_origin_source_projects_the_origin_in_the_current_frame() {
    let mut snapshot = snapshot();
    assert_eq!(
        project(&snapshot, WorkPointReference::Origin).local_position,
        [-10.0, -20.0]
    );
    orient(&mut snapshot, false);
    let result = project(&snapshot, WorkPointReference::Origin);
    assert_eq!(result.model_position, [0.0; 3]);
    assert_eq!(result.local_position, [-20.0, 10.0]);
    orient(&mut snapshot, true);
    assert_eq!(
        project(&snapshot, WorkPointReference::Origin).local_position,
        [20.0, -10.0]
    );
}
#[test]
fn exact_footprint_includes_sketch_host_axis_and_point_identity() {
    let mut snapshot = snapshot();
    orient(&mut snapshot, false);
    let expected = FeatureReadFootprint::new(
        [
            FeatureReadKey::Sketch(7),
            FeatureReadKey::Feature(FeatureId(1)),
            FeatureReadKey::Feature(FeatureId(2)),
            FeatureReadKey::Feature(FeatureId(4)),
        ],
        false,
    );
    assert_eq!(project(&snapshot, SOURCE).reads, expected);
    let result = project(&snapshot, WorkPointReference::Origin);
    assert_eq!(result.reads.keys().len(), 3);
    assert!(!result.reads.has_unattributed_reads());
}
#[test]
fn axis_edits_drive_projection_without_changing_the_stored_frame() {
    let mut snapshot = snapshot();
    orient(&mut snapshot, false);
    assert_eq!(project(&snapshot, SOURCE).local_position, [-16.0, 8.0]);
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    replace(&mut snapshot, 4, WorkGeometryDefinition::Axis(axis));
    assert_eq!(project(&snapshot, SOURCE).local_position, [-8.0, -16.0]);
}

#[path = "sketch_point_projection/generated.rs"]
mod generated;
#[path = "sketch_point_projection/guards.rs"]
mod guards;
