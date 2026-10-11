// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{SketchAxisReference, SketchOrientationDefinition},
    feature::work_axis::WorkAxisLine,
    sketch::SketchFrameAxis,
};
fn append_plane(snapshot: &mut DocumentSnapshot, plane: SketchPlane) {
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Host",
            WorkGeometryDefinition::Fixed(plane),
        ))
        .unwrap();
    snapshot
        .sketches
        .attach_work_plane(SKETCH, FeatureId(2))
        .unwrap();
}
fn assign_axis(snapshot: &mut DocumentSnapshot) {
    let axis = WorkAxisLine::new([0.0; 3], [0.0, 1.0, 0.0]).unwrap();
    let definition = WorkGeometryDefinition::Axis(axis);
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(3), "Axis", definition))
        .unwrap();
    let orientation = SketchOrientationDefinition {
        reference: SketchAxisReference::Feature(FeatureId(3)),
        sketch_axis: SketchFrameAxis::X,
        reverse: false,
        origin: None,
    };
    snapshot
        .sketches
        .set_orientation(SKETCH, Some(orientation))
        .unwrap();
}
#[test]
fn current_owned_host_axis_and_source_share_live_binding_resolution_and_reads() {
    let mut snapshot = snapshot();
    append_plane(&mut snapshot, SketchPlane::xy());
    assign_axis(&mut snapshot);
    recompute(&mut snapshot);
    assert_ready(&snapshot, [4.0, -2.0]);
    let axis = WorkAxisLine::new([0.0; 3], [1.0, 0.0, 0.0]).unwrap();
    let definition = WorkGeometryDefinition::Axis(axis);
    snapshot
        .work_geometry
        .replace(FeatureId(3), definition, definition.reads())
        .unwrap();
    assert_projection_changed(&snapshot);
    recompute(&mut snapshot);
    assert_ready(&snapshot, [2.0, 4.0]);
    let reads = snapshot.sketches.get(SKETCH).unwrap().read_footprint();
    assert_eq!(reads.keys().len(), 3);
}
#[test]
fn reversing_the_host_normal_invalidates_projected_local_coordinates() {
    let mut snapshot = snapshot();
    append_plane(&mut snapshot, SketchPlane::xy());
    recompute(&mut snapshot);
    snapshot
        .work_geometry
        .flip_plane_normal(FeatureId(2))
        .unwrap();
    assert_projection_changed(&snapshot);
    recompute(&mut snapshot);
    assert_ready(&snapshot, [2.0, -4.0]);
}
#[test]
fn host_loss_is_a_named_projection_failure_without_a_last_good_frame() {
    let mut snapshot = snapshot();
    append_plane(&mut snapshot, SketchPlane::xy());
    recompute(&mut snapshot);
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    let result = snapshot.solve_document_sketches(SolveOptions::default(), || false);
    let error = result.report.reports[0].result.as_ref().unwrap_err();
    assert!(matches!(
        error,
        SketchError::ProjectionSource { point: ANCHOR, .. }
    ));
    assert!(error.to_string().contains("FeatureId(2)"));
}
