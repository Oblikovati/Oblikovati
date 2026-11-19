// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const HISTORY_CONTROL_CASES: &[(bool, bool, bool, [bool; 2])] = &[
    (false, false, false, [false, false]),
    (false, true, false, [false, true]),
    (true, false, false, [true, false]),
    (true, true, false, [true, true]),
    (false, false, true, [false, false]),
    (false, true, true, [false, false]),
    (true, false, true, [false, false]),
    (true, true, true, [false, false]),
];
#[test]
fn history_controls_preserve_go_enabled_but_inert_transaction_rules() {
    for &(undo, redo, open, dispatch) in HISTORY_CONTROL_CASES {
        let state = GuiCommandState {
            undo_label: undo.then(|| "Edit".into()),
            redo_label: redo.then(|| "Redo edit".into()),
            transaction_open: open,
            ..Default::default()
        };
        verify_history_commands(&state, [undo, redo], dispatch);
        verify_nonhistory_commands(&state);
    }
}
#[test]
fn sketch_commands_follow_the_active_document_and_edit_context() {
    for (authoring, start, edit) in [
        (GuiAuthoringContext::NoDocument, false, false),
        (GuiAuthoringContext::OtherDocument, false, false),
        (GuiAuthoringContext::Part, true, false),
        (GuiAuthoringContext::Sketch, false, true),
    ] {
        let state = GuiCommandState {
            authoring,
            ..Default::default()
        };
        assert_eq!(
            GuiShellAction::ChooseSketchPlane.dispatchable(&state),
            start
        );
        assert_eq!(GuiShellAction::FinishSketch.dispatchable(&state), edit);
        assert_eq!(
            GuiShellAction::PlaceSketchTool(crate::GuiSketchTool::Rectangle).dispatchable(&state),
            edit
        );
    }
}
fn verify_history_commands(state: &GuiCommandState, enabled: [bool; 2], dispatch: [bool; 2]) {
    for (index, action) in [GuiShellAction::Undo, GuiShellAction::Redo]
        .into_iter()
        .enumerate()
    {
        assert_eq!(action.enabled(state), enabled[index]);
        assert_eq!(action.dispatchable(state), dispatch[index]);
    }
}
fn verify_nonhistory_commands(state: &GuiCommandState) {
    state.validate().unwrap();
    for action in [
        GuiShellAction::NewPart,
        GuiShellAction::Save,
        GuiShellAction::Home,
    ] {
        assert!(action.enabled(state) && action.dispatchable(state));
        assert!(action.next_label(state).is_none());
    }
    for (action, label) in [
        (GuiShellAction::Undo, state.undo_label.as_deref()),
        (GuiShellAction::Redo, state.redo_label.as_deref()),
    ] {
        assert_eq!(action.next_label(state), label);
    }
}
#[test]
fn history_labels_refuse_embedded_nul_before_widget_calls() {
    for state in [
        GuiCommandState {
            undo_label: Some("bad\0undo".into()),
            ..Default::default()
        },
        GuiCommandState {
            redo_label: Some("bad\0redo".into()),
            ..Default::default()
        },
    ] {
        assert!(
            state
                .validate()
                .unwrap_err()
                .contains("expected text without NUL")
        );
    }
}

#[test]
fn a_running_line_command_leaves_sketch_commands_available_and_opens_no_transaction() {
    let mut state = GuiCommandState {
        document: Some(1),
        authoring: GuiAuthoringContext::Sketch,
        sketch_tool_keyboard: true,
        undo_label: Some("Previous".into()),
        ..Default::default()
    };
    state.validate().unwrap();
    for action in [
        GuiShellAction::PlaceSketchTool(crate::GuiSketchTool::Point),
        GuiShellAction::ConstrainSketch(crate::GuiSketchConstraint::Fix),
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
    ] {
        assert!(action.dispatchable(&state));
        state.transaction_open = true;
        assert!(
            !action.dispatchable(&state),
            "a caller's transaction refuses {action:?}"
        );
        state.transaction_open = false;
    }
}

#[test]
fn express_assembly_offers_full_load_but_refuses_edits_and_save() {
    let mut state = GuiCommandState {
        document: Some(7),
        documents: vec![oblikovati_api::wire::DocumentInfo {
            id: 7,
            express: true,
            ..Default::default()
        }],
        undo_label: Some("Edit".into()),
        redo_label: Some("Edit".into()),
        ..Default::default()
    };
    assert!(GuiShellAction::LoadFullDocument(7).enabled(&state));
    assert!(!GuiShellAction::Save.enabled(&state));
    assert!(!GuiShellAction::Undo.enabled(&state));
    assert!(!GuiShellAction::Redo.enabled(&state));
    state.documents[0].express = false;
    assert!(!GuiShellAction::LoadFullDocument(7).enabled(&state));
    assert!(GuiShellAction::Save.enabled(&state));
}

#[test]
fn sketch_context_choice_requires_current_offered_identity() {
    use crate::{GuiSketchCandidate, GuiSketchContext};
    use oblikovati_model::sketch::SketchEntityId;
    let entity = SketchEntityId(4);
    let mut state = GuiCommandState {
        sketch_at_ready: true,
        sketch_context: Some(GuiSketchContext {
            token: 7,
            candidates: vec![GuiSketchCandidate {
                entity,
                label: "Line".into(),
            }],
            ..Default::default()
        }),
        ..Default::default()
    };
    let action = GuiShellAction::SelectSketchContextCandidate { token: 7, entity };
    assert!(action.enabled(&state) && action.dispatchable(&state));
    state.sketch_context.as_mut().unwrap().candidates[0].entity = SketchEntityId(5);
    assert!(!action.enabled(&state));
    state.sketch_context.as_mut().unwrap().candidates[0].entity = entity;
    state.sketch_context.as_mut().unwrap().token = 8;
    assert!(!action.enabled(&state));
    state.sketch_context.as_mut().unwrap().token = 7;
    state.sketch_at_ready = false;
    assert!(!action.enabled(&state));
}

#[test]
fn explode_context_requires_the_current_block_and_token() {
    use crate::GuiSketchContext;
    use oblikovati_model::sketch::SketchEntityId;
    let instance = SketchEntityId(7);
    let mut state = GuiCommandState {
        sketch_at_ready: true,
        sketch_context: Some(GuiSketchContext {
            token: 3,
            block: Some(instance),
            ..Default::default()
        }),
        ..Default::default()
    };
    let action = GuiShellAction::ExplodeSketchContext { token: 3, instance };
    assert!(action.enabled(&state));
    state.sketch_context.as_mut().unwrap().token = 4;
    assert!(!action.enabled(&state));
    state.sketch_context.as_mut().unwrap().token = 3;
    state.sketch_context.as_mut().unwrap().block = Some(SketchEntityId(8));
    assert!(!action.enabled(&state));
}
