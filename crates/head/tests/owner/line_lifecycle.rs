// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    line_tools::{line_key, line_owner, line_tool_geometry},
    slot_input::{PlanarInputWindow, deliver},
};
use crate::placement_test_inputs::*;
use oblikovati_viewport::{ViewportEvent, ViewportKey, ViewportPhysicalKey};
#[test]
fn retiring_an_old_line_scope_never_aborts_a_new_caller_group() {
    let (mut owner, document, _) = line_owner();
    Client::new(owner.document_host())
        .transactions()
        .end()
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Caller".into(),
        })
        .unwrap();
    let caller = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .transaction_identity();
    deliver(&mut owner, &[]);
    assert!(!owner.placement_active());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .transaction_identity(),
        caller
    );
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
}
#[test]
fn line_scope_releases_on_focus_loss_and_replacement_without_committing_the_draft() {
    for replacement in [false, true] {
        let (mut owner, document, sketch) = line_owner();
        deliver(&mut owner, &[button([0, 0], true), button([50, 20], true)]);
        if replacement {
            owner
                .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Point)
                .unwrap();
        } else {
            deliver(&mut owner, &[ViewportEvent::Focus(false)]);
        }
        assert!(
            !owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .in_transaction()
        );
        assert!(
            line_tool_geometry(&owner, document, sketch)
                .lines()
                .is_empty()
        );
    }
}
#[test]
fn preview_projection_failure_and_duplicate_clicks_leave_a_recoverable_draft() {
    let (mut owner, document, sketch) = line_owner();
    deliver(&mut owner, &[button([0, 0], true)]);
    owner
        .process_window_inputs(
            &[button([50, 20], true)],
            &PlanarInputWindow {
                fail: true,
                ..Default::default()
            },
        )
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([50, 20], true),
            line_key(ViewportPhysicalKey::Enter),
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
            button([50, 20], true),
            line_key(ViewportPhysicalKey::Enter),
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
fn invalid_replacement_does_not_retire_the_armed_line_scope() {
    let (mut owner, document, sketch) = line_owner();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .transaction_identity();
    assert!(
        owner
            .start_slot_placement(document, sketch, f64::NAN)
            .is_err()
    );
    assert!(owner.placement_active());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .transaction_identity(),
        before
    );
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
}
#[test]
fn a_zero_length_closing_edge_is_refused_and_the_draft_can_be_corrected() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([50, 20], true),
            button([0, 0], true),
        ],
    );
    deliver(
        &mut owner,
        &[ViewportEvent::Input(
            oblikovati_viewport::ViewportInput::Text('c'),
        )],
    );
    assert!(owner.placement_active());
    assert!(
        line_tool_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementFailed { cause }) if cause.contains("finite nonzero length"))
    );
    deliver(
        &mut owner,
        &[
            line_key(ViewportPhysicalKey::Backspace),
            line_key(ViewportPhysicalKey::Enter),
        ],
    );
    assert_eq!(
        line_tool_geometry(&owner, document, sketch).lines().len(),
        1
    );
}
