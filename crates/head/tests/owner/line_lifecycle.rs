// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    line_tools::{line_owner, line_tool_geometry},
    slot_input::{PlanarInputWindow, deliver},
};
use crate::placement_test_inputs::*;
use oblikovati_viewport::{ViewportEvent, ViewportKey};
#[test]
fn focus_loss_and_replacement_end_the_command_and_drop_only_the_pending_segment() {
    for replacement in [false, true] {
        let (mut owner, document, sketch) = line_owner();
        deliver(
            &mut owner,
            &[
                button([0, 0], true),
                button([0, 0], false),
                button([50, 20], true),
                button([50, 20], false),
                motion([80, 90]),
            ],
        );
        if replacement {
            owner
                .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Point)
                .unwrap();
        } else {
            deliver(&mut owner, &[ViewportEvent::Focus(false)]);
        }
        assert_eq!(
            line_tool_geometry(&owner, document, sketch).lines().len(),
            1
        );
    }
}
#[test]
fn a_failed_preview_and_a_repeated_click_leave_a_recoverable_command() {
    let (mut owner, document, sketch) = line_owner();
    deliver(&mut owner, &[button([0, 0], true), button([0, 0], false)]);
    owner
        .process_window_inputs(
            &[motion([50, 20])],
            &PlanarInputWindow {
                fail: true,
                ..Default::default()
            },
        )
        .unwrap();
    assert!(owner.placement_active());
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([50, 20], true),
            button([50, 20], false),
        ],
    );
    let geometry = line_tool_geometry(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 1);
    assert_eq!(geometry.points().len(), 2);
}
#[test]
fn arming_and_canceling_line_preserve_the_redo_branch_and_refuse_history_mid_draft() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([50, 20], true),
            button([50, 20], false),
            key(ViewportKey::Cancel, true),
        ],
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    deliver(
        &mut owner,
        &[key(ViewportKey::Undo, true), key(ViewportKey::Redo, true)],
    );
    assert!(
        line_tool_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        before
    );
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        line_tool_geometry(&owner, document, sketch).lines().len(),
        1
    );
}

#[test]
fn invalid_replacement_keeps_the_running_line_command() {
    let (mut owner, document, sketch) = line_owner();
    assert!(
        owner
            .start_slot_placement(document, sketch, f64::NAN)
            .is_err()
    );
    assert!(owner.placement_active());
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
}
#[test]
fn a_zero_length_closing_edge_is_refused_and_the_command_can_continue() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([50, 20], true),
            button([50, 20], false),
            button([0, 0], true),
            button([0, 0], false),
        ],
    );
    let lines = line_tool_geometry(&owner, document, sketch).lines().len();
    assert!(
        owner
            .line_command_from_gui(oblikovati_gui::GuiLineCommand::Close)
            .is_err()
    );
    assert!(owner.placement_active());
    assert_eq!(
        line_tool_geometry(&owner, document, sketch).lines().len(),
        lines
    );
}
