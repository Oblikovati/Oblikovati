// SPDX-License-Identifier: GPL-2.0-only
use crate::SketchPlacementTool;
use oblikovati_model::sketch::{CircumcircleOptions, Sketch2D, SketchPlacementRecipe};
#[test]
fn arc_slots_share_click_drag_preview_commit_restart_and_history() {
    for tool in [
        SketchPlacementTool::CenterPointArcSlot { width: 2. },
        SketchPlacementTool::ThreePointArcSlot {
            width: 2.,
            options: CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    ] {
        let clicked = super::circle_input::place_round_tool(tool, false, verify_slot);
        let dragged = super::circle_input::place_round_tool(tool, true, verify_slot);
        assert!(clicked.same_recipe(&dragged));
    }
}
fn verify_slot(geometry: &Sketch2D, recipe: &SketchPlacementRecipe) {
    assert_eq!(geometry.arcs().len(), 4);
    assert_eq!(geometry.constraints().len(), 4);
    crate::placement_test_samples::assert_recipe_tail(geometry, recipe);
}
