// SPDX-License-Identifier: GPL-2.0-only
use crate::SketchPlacementTool;
use oblikovati_model::sketch::{Sketch2D, SketchPlacementRecipe};
#[test]
fn polygon_click_and_drag_share_preview_commit_restart_and_history() {
    for sides in [3, 6, 17] {
        let tool = SketchPlacementTool::Polygon { sides };
        let clicked = super::circle_input::place_round_tool(tool, false, verify_polygon);
        let dragged = super::circle_input::place_round_tool(tool, true, verify_polygon);
        assert!(clicked.same_recipe(&dragged));
    }
}
fn verify_polygon(geometry: &Sketch2D, recipe: &SketchPlacementRecipe) {
    assert_eq!(geometry.lines().len(), recipe.members.len() - 1);
    assert_eq!(geometry.constraints().len(), 2 * geometry.lines().len() - 1);
    assert_eq!(geometry.circles().len(), 1);
    assert!(geometry.is_construction(geometry.circles()[0].id).unwrap());
    crate::placement_test_samples::assert_recipe_tail(geometry, recipe);
}
#[test]
fn invalid_polygon_side_count_preserves_the_active_tool() {
    let (mut owner, document, sketch) =
        super::rectangle_input::placement_owner(SketchPlacementTool::Polygon { sides: 6 });
    for sides in [0, 1, 2] {
        assert!(
            owner
                .start_sketch_placement(document, sketch, SketchPlacementTool::Polygon { sides })
                .unwrap_err()
                .contains("at least 3")
        );
        assert!(owner.placement_active());
    }
}

#[test]
fn collapsed_polygon_can_retry_then_cancel_without_history_changes() {
    use super::slot_input::deliver;
    use crate::{
        HeadWork,
        placement_test_inputs::{button, key, motion},
    };
    let (mut owner, _, _) =
        super::rectangle_input::placement_owner(SketchPlacementTool::Polygon { sides: 6 });
    let before = owner.gui_command_state().unwrap();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([0, 0], true),
        ],
    );
    assert!(matches!(
        owner.take_command_notice(),
        Some(crate::DesktopCommandNotice::PlacementFailed { .. })
    ));
    assert_eq!(owner.gui_command_state().unwrap(), before);
    deliver(&mut owner, &[motion([10, 5])]);
    assert!(owner.placement_preview().is_some());
    deliver(
        &mut owner,
        &[key(oblikovati_viewport::ViewportKey::Cancel, true)],
    );
    assert!(!owner.placement_active());
    assert_eq!(owner.gui_command_state().unwrap(), before);
}
