// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadWork, SketchPlacementTool};

use oblikovati_model::sketch::{CircumcircleOptions, Sketch2D, SketchPlacementRecipe};
#[test]
fn arc_tools_share_click_drag_preview_commit_restart_and_history() {
    for tool in [
        SketchPlacementTool::CenterPointArc,
        SketchPlacementTool::ThreePointArc {
            options: CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    ] {
        let clicked = super::circle_input::place_round_tool(tool, false, verify_solved_arc);
        let dragged = super::circle_input::place_round_tool(tool, true, verify_solved_arc);
        assert!(clicked.same_recipe(&dragged));
    }
}
fn verify_solved_arc(geometry: &Sketch2D, recipe: &SketchPlacementRecipe) {
    assert_eq!(geometry.arcs().len(), 1);
    crate::placement_test_samples::assert_recipe_tail(geometry, recipe);
}

#[test]
fn invalid_arc_endpoints_keep_the_stage_for_retry_and_cancel() {
    for tool in [
        SketchPlacementTool::CenterPointArc,
        SketchPlacementTool::ThreePointArc {
            options: CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    ] {
        refuse_then_retry(tool);
    }
}
fn refuse_then_retry(tool: SketchPlacementTool) {
    use super::{rectangle_input::placement_owner, slot_input::deliver};
    use crate::placement_test_inputs::{button, key, motion};
    let (mut owner, _, _) = placement_owner(tool);
    let before = owner.gui_command_state().unwrap();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([10, 0], true),
            button([10, 0], false),
        ],
    );
    deliver(&mut owner, &[button([20, 0], true)]);
    assert!(matches!(
        owner.take_command_notice(),
        Some(crate::DesktopCommandNotice::PlacementFailed { .. })
    ));
    assert_eq!(owner.gui_command_state().unwrap(), before);
    deliver(&mut owner, &[motion([5, 5])]);
    assert!(owner.placement_preview().is_some());
    deliver(
        &mut owner,
        &[key(oblikovati_viewport::ViewportKey::Cancel, true)],
    );
    assert!(!owner.placement_active());
    assert_eq!(owner.gui_command_state().unwrap(), before);
}

#[test]
fn placement_packet_retains_the_document_solver_policy() {
    use crate::placement_test_inputs::{button, motion};
    use oblikovati_solve::nonlinear::SolveOptions;
    let options = SolveOptions {
        max_iterations: 1,
        ..Default::default()
    };
    let (mut owner, _, _) = super::rectangle_input::placement_owner_with_options(
        SketchPlacementTool::CenterPointArc,
        options,
    );
    super::slot_input::deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([10, 0], true),
            button([10, 0], false),
            motion([5, 5]),
        ],
    );
    let packet = owner.sketch_preview().unwrap();
    assert_eq!(packet.solve_options, options);
    assert!(owner.placement_active());
}
