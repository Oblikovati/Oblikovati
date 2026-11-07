// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{DisplayTolerance, Resolution, ToleranceCap};
use oblikovati_martensite::retain_certified_sketch_prism;
use oblikovati_model::feature::{sketch_prism::SketchPrismDefinition, work_geometry::DatumLength};

fn resolution() -> Resolution {
    Resolution::new(
        1e-6,
        1e-7,
        1e-9,
        DisplayTolerance {
            chordal: 1e-3,
            angular: 0.1,
        },
        ToleranceCap::Refuse(1e-5),
    )
}
fn parameter_definition() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
#[test]
fn current_document_parameters_drive_the_certified_sketch_prism() {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    let definition = parameter_definition();
    let before = retain_certified_sketch_prism(&document, &definition, resolution()).unwrap();
    assert_analytic_scalar(before.body.report().properties.volume, 24.);
    document.parameters.set_expression(1, "6 cm").unwrap();
    document.parameters.set_expression(3, "3 cm").unwrap();
    let stale = retain_certified_sketch_prism(&document, &definition, resolution())
        .err()
        .unwrap();
    assert_eq!(stale.sketch(), Some(SKETCH));
    solve_selected_sketches(&mut document);
    let after = retain_certified_sketch_prism(&document, &definition, resolution()).unwrap();
    assert_analytic_scalar(after.body.report().properties.volume, 54.);
    assert_source_roles(after.body.report());
    assert_analytic_scalar(before.body.report().properties.volume, 24.);
}
#[test]
fn certified_sketch_prism_uses_the_current_placed_plane() {
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    document
        .sketches
        .replace_plane(SKETCH, SketchPlane::yz())
        .unwrap();
    let result =
        retain_certified_sketch_prism(&document, &parameter_definition(), resolution()).unwrap();
    assert_analytic_centroid(result.body.report().properties.centroid, [1., 2., 1.5]);
}

#[test]
fn certified_sketch_prism_accepts_holes_and_reversed_source_lines() {
    let mut document = selected_sketch_document();
    add_fixed_sketch_hole(&mut document);
    reverse_profile_source_line(&mut document, 102);
    solve_selected_sketches(&mut document);
    let mut selection = selected_sketch_boundary();
    selection.outer[1].reversed = true;
    selection.holes.push(selected_sketch_line_ring(111));
    let definition = SketchPrismDefinition::new(
        SKETCH,
        selection,
        [
            DatumLength::Constant(1.),
            DatumLength::Constant(2.),
            DatumLength::Constant(-3.),
        ],
    )
    .unwrap();
    let result = retain_certified_sketch_prism(&document, &definition, resolution()).unwrap();
    assert_analytic_scalar(result.body.report().properties.volume, 30.);
    assert_eq!(result.body.report().faces.len(), 10);
}

#[test]
fn distinct_coincident_endpoint_identities_are_declined_without_moving_points() {
    use oblikovati_model::sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPoint2D,
    };
    let mut document = selected_sketch_document();
    let mut recipe = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    recipe.points.push(SketchPoint2D {
        id: SketchEntityId(50),
        position: [0.; 2],
    });
    recipe.lines[0].endpoints[0] = SketchEntityId(50);
    let mut geometry = Sketch2D::from_recipe(&recipe).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(50),
            PlanarConstraint::Coincident {
                points: [SketchEntityId(1), SketchEntityId(50)],
            },
        )
        .unwrap();
    document
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    solve_selected_sketches(&mut document);
    let definition = parameter_definition();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    let before = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    let error = retain_certified_sketch_prism(&document, &definition, resolution())
        .err()
        .unwrap();
    assert_eq!(
        error.phase(),
        oblikovati_martensite::EvaluationPhase::Profile
    );
    assert_eq!(
        document.sketches.get(SKETCH).unwrap().geometry().recipe(),
        before
    );
    // Keep the authored definition unchanged when the kernel refuses the incidence.
    assert_eq!(definition.selection(), &selected_sketch_boundary());
}
