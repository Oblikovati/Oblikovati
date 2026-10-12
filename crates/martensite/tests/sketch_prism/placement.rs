// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId,
    work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
};
fn linked_document() -> oblikovati_model::history::DocumentSnapshot {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "elevation", oblikovati_api::types::USER_PARAM, "4 cm")
        .unwrap();
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(1),
            "Elevation",
            elevation_definition(),
        ))
        .unwrap();
    document
        .sketches
        .attach_work_plane(SKETCH, FeatureId(1))
        .unwrap();
    document
}
#[test]
fn linked_plane_changes_and_failures_are_resolved_on_every_kernel_call() {
    let mut document = linked_document();
    let first =
        retain_selected_sketch_prism(&document, SKETCH, &selected_sketch_boundary(), [0., 0., 2.])
            .unwrap();
    assert_eq!(first.report().properties.centroid, [2., 1.5, 5.]);
    let local = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    document.parameters.set_expression(3, "7 cm").unwrap();
    let moved =
        retain_selected_sketch_prism(&document, SKETCH, &selected_sketch_boundary(), [0., 0., 2.])
            .unwrap();
    assert_eq!(moved.report().properties.centroid, [2., 1.5, 8.]);
    assert_eq!(
        document.sketches.get(SKETCH).unwrap().geometry().recipe(),
        local
    );
    assert_eq!(first.report().properties.centroid, [2., 1.5, 5.]);
}

fn elevation_definition() -> WorkPlaneDefinition {
    WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Parameter(3),
    }
}

#[test]
fn suppressed_host_plane_has_no_last_good_kernel_input() {
    let mut document = linked_document();
    document
        .work_geometry
        .set_suppressed(FeatureId(1), true)
        .unwrap();
    assert!(
        retain_selected_sketch_prism(&document, SKETCH, &selected_sketch_boundary(), [0., 0., 2.])
            .is_err()
    );
}
