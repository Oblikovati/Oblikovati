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

#[test]
fn numeric_extrusion_replays_its_plane_and_depth_parameters_independently() {
    let mut document = depth_document();
    document
        .parameters
        .insert_expression(4, "start", oblikovati_api::types::USER_PARAM, "3 mm")
        .unwrap();
    let base = WorkPlaneReference::Origin(WorkPlaneOrigin::XY);
    let datum = WorkPlaneDefinition::Offset {
        base,
        distance: WorkPlaneDistance::Parameter(4),
    };
    let id = FeatureId(9);
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(id, "Height", datum))
        .unwrap();
    let definition = driven_definition()
        .with_start_plane(WorkPlaneReference::Feature(id))
        .unwrap();
    let recipe = definition.recipe();
    let definition = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    assert_eq!(recipe.version, 4);
    assert_offset_span(&definition, &document, [0.3, 2.]);
    document.parameters.set_expression(4, "6 mm").unwrap();
    assert_offset_span(&definition, &document, [0.6, 2.]);
    document.parameters.set_expression(3, "8 mm").unwrap();
    assert_offset_span(&definition, &document, [0.6, 0.8]);
    assert_eq!(definition.recipe(), recipe);
}

fn assert_offset_span(
    definition: &SketchPrismDefinition,
    snapshot: &DocumentSnapshot,
    [offset, depth]: [f64; 2],
) {
    let input = definition.resolve(snapshot).unwrap();
    assert_eq!(input.sketch.plane.recipe().origin, [0.; 3]);
    assert!((input.sweep_start.recipe().origin[2] - offset).abs() < 1e-12);
    assert!((input.displacement[2] - depth).abs() < 1e-12);
    assert!(
        definition
            .reads(snapshot)
            .unwrap()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(9)))
    );
}
