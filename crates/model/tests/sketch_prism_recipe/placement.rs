// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId,
    work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
};
#[test]
fn linked_frame_reads_and_resolution_follow_current_placement() {
    let mut document = linked_depth_document();
    let definition = driven_definition();
    assert!(
        definition
            .reads(&document)
            .unwrap()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(9)))
    );
    assert_resolved_origin(&definition, &document, [0., 0., 2.]);
    document.parameters.set_expression(3, "5 cm").unwrap();
    assert_resolved_origin(&definition, &document, [0., 0., 5.]);
    assert_eq!(definition.resolve(&document).unwrap().displacement[2], 5.);
}

fn linked_depth_document() -> DocumentSnapshot {
    let mut document = depth_document();
    let datum = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Parameter(3),
    };
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(FeatureId(9), "Height", datum))
        .unwrap();
    document
        .sketches
        .attach_work_plane(SKETCH, FeatureId(9))
        .unwrap();
    document
}

fn assert_resolved_origin(
    definition: &SketchPrismDefinition,
    document: &DocumentSnapshot,
    expected: [f64; 3],
) {
    assert_eq!(
        definition
            .resolve(document)
            .unwrap()
            .sketch
            .plane
            .recipe()
            .origin,
        expected
    );
}
