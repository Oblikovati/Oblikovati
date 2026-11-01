// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use fixture::*;
use oblikovati_model::{
    feature::{FeatureReadKey, sketch_revolve::SketchRevolveDefinition, work_geometry::DatumAngle},
    sketch::{PlanarConstraint, SketchConstraintId, SketchEntityId},
};

fn definition() -> SketchRevolveDefinition {
    SketchRevolveDefinition::new(SKETCH, selected_sketch_boundary(), DatumAngle::parameter(3))
        .unwrap()
}

fn document_with_axis() -> oblikovati_model::history::DocumentSnapshot {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "angle", oblikovati_api::types::USER_PARAM, "90 deg")
        .unwrap();
    let mut geometry = document.sketches.get(SKETCH).unwrap().geometry().clone();
    geometry.add_point(SketchEntityId(10), [0., -1.]).unwrap();
    geometry.add_point(SketchEntityId(11), [0., 4.]).unwrap();
    geometry
        .add_line(
            SketchEntityId(200),
            [SketchEntityId(10), SketchEntityId(11)],
        )
        .unwrap();
    geometry
        .set_line_centerline(SketchEntityId(200), true)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(20),
            PlanarConstraint::Fix {
                point: SketchEntityId(10),
                position: [0., -1.],
            },
        )
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(21),
            PlanarConstraint::Fix {
                point: SketchEntityId(11),
                position: [0., 4.],
            },
        )
        .unwrap();
    document
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    solve_selected_sketches(&mut document);
    document
}

#[test]
fn recipe_round_trip_keeps_only_authored_revolve_inputs() {
    let original = definition();
    let encoded = serde_json::to_string(&original.recipe()).unwrap();
    let restored =
        SketchRevolveDefinition::from_recipe(&serde_json::from_str(&encoded).unwrap()).unwrap();
    assert!(restored.same_recipe(&original));
    assert!(!encoded.contains("axis"));
}

#[test]
fn resolve_reads_current_angle_and_sole_centerline() {
    let mut document = document_with_axis();
    let revolve = definition();
    let resolved = revolve.resolve(&document).unwrap();
    assert_eq!(resolved.axis, [[0., -1.], [0., 4.]]);
    assert!((resolved.angle - std::f64::consts::FRAC_PI_2).abs() < 1e-12);
    document.parameters.set_expression(3, "180 deg").unwrap();
    assert!((definition().resolve(&document).unwrap().angle - std::f64::consts::PI).abs() < 1e-12);
}

#[test]
fn resolve_declines_a_sketch_without_a_centerline() {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "angle", oblikovati_api::types::USER_PARAM, "90 deg")
        .unwrap();
    assert!(
        definition()
            .resolve(&document)
            .unwrap_err()
            .to_string()
            .contains("exactly one owned centerline")
    );
}

#[test]
fn reads_include_the_sketch_dimensions_and_angle_parameter() {
    let reads = definition().reads(&document_with_axis()).unwrap();
    assert_eq!(
        reads.keys().iter().copied().collect::<Vec<_>>(),
        vec![
            FeatureReadKey::Parameter(1),
            FeatureReadKey::Parameter(2),
            FeatureReadKey::Parameter(3),
            FeatureReadKey::Sketch(1),
        ]
    );
}
