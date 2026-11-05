// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::SketchPrismSelection;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
fn refusal(
    document: &oblikovati_model::history::DocumentSnapshot,
    selection: &SketchPrismSelection,
) -> String {
    retain_selected_sketch_prism(document, SKETCH, selection, [0., 0., 2.])
        .err()
        .unwrap()
}
#[test]
fn absent_free_and_failed_verification_never_reuse_a_previous_body() {
    let mut document = selected_sketch_document();
    let mut recipe = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    recipe.constraints.pop();
    document
        .sketches
        .replace_geometry(SKETCH, Sketch2D::from_recipe(&recipe).unwrap())
        .unwrap();
    assert!(refusal(&document, &selected_sketch_boundary()).contains("verification absent"));
    solve_selected_sketches(&mut document);
    assert!(refusal(&document, &selected_sketch_boundary()).contains("degrees of freedom"));
    document.parameters.set_expression(1, "missing").unwrap();
    solve_selected_sketches(&mut document);
    assert!(refusal(&document, &selected_sketch_boundary()).contains("verification failed"));
}
#[test]
fn boundary_classification_refuses_short_duplicate_missing_and_open_selections() {
    let document = selected_sketch_document();
    let mut selected = selected_sketch_boundary();
    selected.outer.pop();
    assert!(refusal(&document, &selected).contains("closed directed boundary"));
    selected.outer.pop();
    assert!(refusal(&document, &selected).contains("at least three"));
    selected = selected_sketch_boundary();
    selected.holes.push(selected.outer.clone());
    assert!(refusal(&document, &selected).contains("one occurrence"));
    selected = selected_sketch_boundary();
    selected.outer[0].line = SketchEntityId(1);
    assert!(refusal(&document, &selected).contains("expected an existing line"));
    selected.outer[0].line = SketchEntityId(999);
    assert!(refusal(&document, &selected).contains("SketchEntityId(999)"));
}
#[test]
fn construction_and_reversed_open_boundaries_cannot_be_published() {
    let mut document = selected_sketch_document();
    let mut geometry = document.sketches.get(SKETCH).unwrap().geometry().clone();
    geometry
        .set_curve_construction(&[SketchEntityId(101)], true)
        .unwrap();
    document
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    solve_selected_sketches(&mut document);
    assert!(refusal(&document, &selected_sketch_boundary()).contains("non-construction"));
    let document = selected_sketch_document();
    let mut selected = selected_sketch_boundary();
    selected.outer[0].reversed = true;
    assert!(refusal(&document, &selected).contains("shared authored endpoint identity"));
}
#[test]
fn missing_sketch_is_a_named_owner_decline() {
    let document = selected_sketch_document();
    let error = retain_selected_sketch_prism(
        &document,
        oblikovati_model::document_sketch::DocumentSketchId(99),
        &selected_sketch_boundary(),
        [0., 0., 2.],
    )
    .err()
    .unwrap();
    assert!(error.contains("99"));
    assert!(error.contains("expected"));
}
#[test]
fn geometrically_equal_distinct_endpoints_need_kernel_arrangement() {
    let mut document = selected_sketch_document();
    let geometry = distinct_endpoint_geometry(&document);
    document
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
    solve_selected_sketches(&mut document);
    assert!(
        refusal(&document, &selected_sketch_boundary())
            .contains("distinct coincident points require kernel profile arrangement")
    );
}
#[test]
fn invalid_displacement_retains_the_source_sketch_in_the_kernel_decline() {
    let document = selected_sketch_document();
    let error =
        retain_selected_sketch_prism(&document, SKETCH, &selected_sketch_boundary(), [0.; 3])
            .err()
            .unwrap();
    assert!(error.contains("DocumentSketchId(1)"));
    assert!(error.contains("nonzero profile-normal"));
}

fn distinct_endpoint_geometry(document: &oblikovati_model::history::DocumentSnapshot) -> Sketch2D {
    let mut recipe = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    recipe.points.push(oblikovati_model::sketch::SketchPoint2D {
        id: SketchEntityId(50),
        position: [0.; 2],
    });
    recipe.lines[0].endpoints[0] = SketchEntityId(50);
    let mut geometry = Sketch2D::from_recipe(&recipe).unwrap();
    geometry
        .add_constraint(
            oblikovati_model::sketch::SketchConstraintId(50),
            oblikovati_model::sketch::PlanarConstraint::Fix {
                point: SketchEntityId(50),
                position: [0.; 2],
            },
        )
        .unwrap();
    geometry
}
