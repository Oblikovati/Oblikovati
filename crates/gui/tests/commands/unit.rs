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
        assert_eq!(GuiShellAction::StartSketchXY.dispatchable(&state), start);
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
fn owned_line_scope_allows_tool_replacement_but_not_sketch_exit_or_history() {
    let mut state = GuiCommandState {
        document: Some(1),
        authoring: GuiAuthoringContext::Sketch,
        transaction_open: true,
        sketch_tool_transaction: true,
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
        state.sketch_tool_transaction = false;
        assert!(!action.dispatchable(&state));
        state.sketch_tool_transaction = true;
    }
    assert!(!GuiShellAction::FinishSketch.dispatchable(&state));
    assert!(!GuiShellAction::Undo.dispatchable(&state));
    state.transaction_open = false;
    assert!(
        state
            .validate()
            .unwrap_err()
            .contains("expected an open owned sketch-tool transaction")
    );
}
