// SPDX-License-Identifier: GPL-2.0-only
//! Move tool transaction, preview, and retry behavior at the document owner.
use super::{geometric_tools::geometric_owner, slot_input::deliver, *};
use crate::placement_test_inputs::*;
use oblikovati_gui::{GuiSketchMoveAction, GuiSketchMoveStage};

#[test]
fn move_refuses_open_transactions_without_starting_or_changing_selection() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Caller".into(),
        })
        .unwrap();
    let error = owner
        .sketch_move_from_gui(GuiSketchMoveAction::Start)
        .unwrap_err();
    assert!(error.contains("expected no open transaction"));
    assert!(owner.sketch_tool.is_none());
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
}
#[test]
fn zero_move_can_be_retried_without_changing_geometry_or_history() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let history = owner.gui_command_state().unwrap().undo_label;
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Start)
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([50, 10], true),
            button([0, 0], true),
            button([0, 0], true),
        ],
    );
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_move
            .map(|status| status.stage),
        Some(GuiSketchMoveStage::Target)
    );
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::AuthoringFailed { cause }) if cause.contains("distinct"))
    );
    assert!(
        before.same_evaluated_state(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    assert_eq!(owner.gui_command_state().unwrap().undo_label, history);
    deliver(&mut owner, &[button([20, 30], true)]);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_move
            .map(|status| status.stage),
        Some(GuiSketchMoveStage::Geometry)
    );
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry();
    assert_ne!(geometry.points()[0].position, [0., 0.]);
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Done)
        .unwrap();
    assert!(owner.sketch_tool.is_none());
}

#[test]
fn failed_projection_clears_old_preview_and_the_move_can_resume() {
    use super::slot_input::PlanarInputWindow;
    let (mut owner, document, _) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Start)
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([50, 10], true),
            button([0, 0], true),
            motion([20, 30]),
        ],
    );
    assert!(owner.sketch_preview().is_some());
    owner
        .process_window_inputs(
            &[motion([30, 40])],
            &PlanarInputWindow {
                fail: true,
                ..Default::default()
            },
        )
        .unwrap();
    assert!(owner.sketch_preview().is_none());
    assert!(
        before.same_evaluated_state(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    deliver(&mut owner, &[motion([20, 30])]);
    assert!(owner.sketch_preview().is_some());
}
#[test]
fn transaction_opened_during_move_cannot_adopt_the_preview() {
    let (mut owner, document, _) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Start)
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([50, 10], true),
            button([0, 0], true),
            motion([20, 30]),
        ],
    );
    assert!(owner.sketch_preview().is_some());
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Caller".into(),
        })
        .unwrap();
    assert!(owner.sketch_preview().is_none());
    deliver(&mut owner, &[button([20, 30], true)]);
    assert!(owner.sketch_preview().is_none());
    let notice = owner.take_command_notice();
    assert!(
        matches!(&notice, Some(DesktopCommandNotice::PlacementPrompt { message }) if message.contains("canceled because the document changed")),
        "{notice:?}"
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
    assert!(
        before.same_evaluated_state(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
}

#[test]
fn multi_selection_toggles_geometry_and_requires_a_pick_before_base() {
    let (mut owner, document, _) = geometric_owner(&[[0., 0.], [100., 20.], [140., 120.]], true);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Start)
        .unwrap();
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::SetOptimizeSingle(false))
        .unwrap();
    assert!(
        owner
            .sketch_move_from_gui(GuiSketchMoveAction::ChooseBase)
            .is_err()
    );
    deliver(
        &mut owner,
        &[button([50, 10], true), button([120, 70], true)],
    );
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_move
            .unwrap()
            .selection_count,
        2
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        2
    );
    deliver(&mut owner, &[button([50, 10], true)]);
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_move
            .unwrap()
            .selection_count,
        1
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        1
    );
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::ChooseBase)
        .unwrap();
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_move
            .unwrap()
            .stage,
        GuiSketchMoveStage::Base
    );
    assert!(
        owner
            .sketch_move_from_gui(GuiSketchMoveAction::SetOptimizeSingle(true))
            .is_err()
    );
    owner
        .sketch_move_from_gui(GuiSketchMoveAction::Done)
        .unwrap();
    assert!(
        before.same_evaluated_state(
            &owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
        )
    );
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
}
