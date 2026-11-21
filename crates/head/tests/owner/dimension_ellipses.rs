// SPDX-License-Identifier: GPL-2.0-only
//! The same General Dimension command chooses an ellipse axis at label placement.
use super::*;
use oblikovati_api::types::{DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS, DIM_CONSTRAINT_ELLIPSE_RADIUS};

#[test]
fn ellipse_axis_dimensions_wait_for_placement_and_use_the_canonical_contract() {
    for axis in [[1., 0.], [0.6, 0.8]] {
        for minor in [false, true] {
            check_axis_dimension(axis, minor);
        }
    }
}
fn check_axis_dimension(axis: [f64; 2], minor: bool) {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    let recipe = SketchPlacementRecipe::ellipse([0.; 2], axis, [100., 40.]).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &recipe, &[])
        .unwrap();
    let stored_axis = current_ellipse(&owner).major_axis();
    let before = owner.gui_command_state().unwrap().undo_label;
    start(&mut owner);
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementPrompt { message }) if message.contains("ellipse"))
    );
    let pixel = axis.map(|coordinate| (coordinate * 100.).round() as i32);
    deliver(&mut owner, &[button(pixel, true)]);
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(dimensions(&mut owner).is_empty());
    assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
    place_axis_dimension(&mut owner, axis, stored_axis, minor);
}
fn place_axis_dimension(owner: &mut TestOwner, axis: [f64; 2], stored_axis: [f64; 2], minor: bool) {
    let direction = if minor { axis } else { [-axis[1], axis[0]] };
    let at = direction.map(|coordinate| (coordinate * 200.).round() as i32);
    deliver(owner, &[button(at, true)]);
    let expected = if minor {
        DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS
    } else {
        DIM_CONSTRAINT_ELLIPSE_RADIUS
    };
    assert_eq!(draft(owner).kind, expected.0.as_ref());
    assert!(inspection(owner).healthy);
    apply(owner, if minor { "50 cm" } else { "120 cm" }, false).unwrap();
    assert_axis_dimension(owner, at, minor, expected.0.as_ref());
    assert_ellipse_axis_frame(owner, stored_axis, minor);
}
fn assert_axis_dimension(owner: &mut TestOwner, at: [i32; 2], minor: bool, kind: &str) {
    let rows = dimensions(owner);
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].kind, kind);
    assert_eq!(rows[0].text_point, Some(at.map(f64::from).to_vec()));
    assert_solved_radius(rows[0].value, if minor { 50. } else { 120. });
    assert!(inspection(owner).healthy);
    assert_eq!(inspection(owner).dof, 4);
}
fn assert_ellipse_axis_frame(owner: &TestOwner, axis: [f64; 2], minor: bool) {
    let ellipse = current_ellipse(owner);
    assert_eq!(ellipse.major_axis(), axis, "retained local axes");
    let expected = if minor { [100., 50.] } else { [120., 40.] };
    for (actual, expected) in ellipse.radii().into_iter().zip(expected) {
        assert_solved_radius(actual, expected);
    }
}
fn current_ellipse(owner: &TestOwner) -> oblikovati_model::sketch::SketchEllipse2D {
    let edit = owner.active_sketch_edit().unwrap();
    let snapshot = owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot();
    let geometry = snapshot.sketches.get(edit.sketch).unwrap().geometry();
    let ellipse = geometry.ellipses()[0];
    assert_eq!(geometry.point(ellipse.center()).unwrap().position, [0.; 2]);
    ellipse
}
fn assert_solved_radius(actual: f64, expected: f64) {
    let bound = expected.abs().max(1.)
        * oblikovati_solve::nonlinear::SolveOptions::default().relative_tolerance;
    assert!(
        (actual - expected).abs() <= bound,
        "radius {actual}: expected {expected} within {bound}"
    );
}

#[test]
fn a_curve_cannot_replace_a_point_that_general_dimension_already_selected() {
    let circle = SketchPlacementRecipe::circle([0.; 2], 100.).unwrap();
    let arc = SketchPlacementRecipe::arc([0.; 2], [[100., 0.], [0., 100.]], true).unwrap();
    let ellipse = SketchPlacementRecipe::ellipse([0.; 2], [1., 0.], [100., 40.]).unwrap();
    let recipes = [(circle, [100, 0]), (arc, [70, 70]), (ellipse, [100, 0])];
    for (recipe, pixel) in recipes {
        check_mixed_curve_rejection(&recipe, pixel);
    }
}
fn check_mixed_curve_rejection(recipe: &SketchPlacementRecipe, pixel: [i32; 2]) {
    let (mut owner, document, sketch) = geometric_owner(&[[200., 100.], [-200., -100.]], false);
    owner
        .commit_sketch_placement(document, sketch, recipe, &[])
        .unwrap();
    start(&mut owner);
    reject_curve_after_point(&mut owner, pixel);
    deliver(
        &mut owner,
        &[button([-200, -100], true), button([0, 300], true)],
    );
    assert_eq!(
        draft(&mut owner).kind,
        oblikovati_api::types::DIM_CONSTRAINT_DISTANCE.0.as_ref()
    );
}
fn reject_curve_after_point(owner: &mut TestOwner, pixel: [i32; 2]) {
    deliver(owner, &[button([200, 100], true)]);
    let selected = selected_dimension_picks(owner);
    assert_eq!(selected.len(), 1);
    deliver(owner, &[button(pixel, true)]);
    assert_eq!(selected_dimension_picks(owner), selected);
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(dimensions(owner).is_empty());
}
fn selected_dimension_picks(owner: &TestOwner) -> Vec<oblikovati_session::selection::SketchPick> {
    let edit = owner.active_sketch_edit().unwrap();
    owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .selected_sketch_picks(edit.sketch)
}
