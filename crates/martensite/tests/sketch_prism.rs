// SPDX-License-Identifier: GPL-2.0-only
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixtures;
use fixtures::*;
use oblikovati_martensite::{PrismFaceRole, PrismMeshOptions, retain_selected_sketch_prism};
use oblikovati_model::sketch::SketchPlane;
#[test]
fn parameter_driven_prism_consumes_current_fully_constrained_sketch() {
    let mut document = selected_sketch_document();
    let selection = selected_sketch_boundary();
    let before = retain_selected_sketch_prism(&document, SKETCH, &selection, [0., 0., 2.]).unwrap();
    assert_volume(before.report().properties.volume, 24.);
    document.parameters.set_expression(1, "6 cm").unwrap();
    assert!(
        retain_selected_sketch_prism(&document, SKETCH, &selection, [0., 0., 2.])
            .err()
            .unwrap()
            .contains("changed")
    );
    solve_selected_sketches(&mut document);
    let after = retain_selected_sketch_prism(&document, SKETCH, &selection, [0., 0., 2.]).unwrap();
    assert_volume(after.report().properties.volume, 36.);
    assert_volume(before.report().properties.volume, 24.);
    assert_source_roles(after.report());
}
#[test]
fn directed_lines_and_holes_reuse_the_same_kernel_path() {
    let mut document = selected_sketch_document();
    add_fixed_sketch_hole(&mut document);
    let mut selection = selected_sketch_boundary();
    selection.holes.push(selected_sketch_line_ring(111));
    reverse_profile_source_line(&mut document, 102);
    selection.outer[1].reversed = true;
    solve_selected_sketches(&mut document);
    let before = document.sketches.recipe();
    let body = retain_selected_sketch_prism(&document, SKETCH, &selection, [1., 2., -3.]).unwrap();
    assert_volume(body.report().properties.volume, 30.);
    assert_eq!(body.report().faces.len(), 10);
    body.presentation(PrismMeshOptions {
        chord_error: 0.05,
        angular_error: 0.5,
    })
    .unwrap();
    assert_eq!(document.sketches.recipe(), before);
}
#[test]
fn current_plane_changes_placement_without_editing_local_geometry() {
    let mut document = selected_sketch_document();
    let before = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    document
        .sketches
        .replace_plane(SKETCH, SketchPlane::yz())
        .unwrap();
    let body =
        retain_selected_sketch_prism(&document, SKETCH, &selected_sketch_boundary(), [0., 0., 2.])
            .unwrap();
    assert_volume(body.report().properties.volume, 24.);
    assert_eq!(body.report().properties.centroid, [1., 2., 1.5]);
    assert_eq!(
        document.sketches.get(SKETCH).unwrap().geometry().recipe(),
        before
    );
}
fn assert_volume(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < expected * 1e-9,
        "volume {actual}, expected {expected}"
    );
}
#[path = "sketch_prism/placement.rs"]
mod placement;
#[path = "sketch_prism/refusals.rs"]
mod refusals;

fn assert_source_roles(report: &oblikovati_martensite::PrismProbeReport) {
    for key in 101..=104 {
        assert!(
            report
                .faces
                .iter()
                .any(|face| face.role == PrismFaceRole::ProfileEdge(key))
        );
    }
}
