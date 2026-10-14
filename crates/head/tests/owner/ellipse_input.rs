// SPDX-License-Identifier: GPL-2.0-only
use crate::SketchPlacementTool;
use oblikovati_model::sketch::{Sketch2D, SketchPlacementRecipe};
#[test]
fn ellipse_uses_shared_click_drag_preview_commit_and_history() {
    let clicked =
        super::circle_input::place_round_tool(SketchPlacementTool::Ellipse, false, verify_ellipse);
    let dragged =
        super::circle_input::place_round_tool(SketchPlacementTool::Ellipse, true, verify_ellipse);
    assert!(clicked.same_recipe(&dragged));
}
fn verify_ellipse(geometry: &Sketch2D, recipe: &SketchPlacementRecipe) {
    assert_eq!(geometry.ellipses().len(), 1);
    assert_eq!(geometry.ellipses()[0].radii(), [10., 5.]);
    crate::placement_test_samples::assert_recipe_tail(geometry, recipe);
}
