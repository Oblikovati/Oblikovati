// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{CloseDocumentArgs, OpenDocumentArgs, SaveDocumentAsArgs};

#[test]
fn a_save_dialog_needs_a_real_document_identity() {
    let mut files = GuiDocumentFiles::default();
    let refusal = files.request_save_path(0, "Part").unwrap_err();
    assert!(refusal.contains("nonzero") && refusal.contains('0'));
    files.request_save_path(7, "Part").unwrap();
    assert!(files.needs_frame());
}
#[test]
fn file_actions_require_live_identity_and_no_open_transaction() {
    let open = GuiShellAction::OpenDocument(OpenDocumentArgs {
        full_document_name: "part.opd".into(),
        ..Default::default()
    });
    let save = GuiShellAction::SaveDocumentAs(SaveDocumentAsArgs {
        document: 7,
        new_full_document_name: "part.opd".into(),
    });
    let mut state = GuiCommandState::default();
    let close = GuiShellAction::CloseDocument(CloseDocumentArgs {
        id: 7,
        force: false,
    });
    assert!(!close.dispatchable(&state));
    assert!(open.dispatchable(&state) && !save.dispatchable(&state));
    state.document = Some(7);
    assert!(close.dispatchable(&state));
    assert!(save.dispatchable(&state));
    state.transaction_open = true;
    assert!(!close.dispatchable(&state));
    assert!(!open.dispatchable(&state) && !save.dispatchable(&state));
}
#[test]
fn an_answer_joins_the_frame_actions_and_queues_each_first_save() {
    let mut files = GuiDocumentFiles::default();
    let mut actions = vec![GuiShellAction::Save];
    files.apply_close_outcome(
        CloseOutcome {
            actions: vec![GuiShellAction::Home],
            first_saves: vec![7],
        },
        &mut actions,
    );
    assert_eq!(actions, [GuiShellAction::Save, GuiShellAction::Home]);
    assert!(files.first_saves.is_waiting() && files.needs_frame());
}
#[test]
fn the_active_name_falls_back_when_no_row_is_known() {
    let mut state = GuiCommandState {
        document: Some(7),
        ..Default::default()
    };
    assert_eq!(active_name(&state), "Part");
    state.documents.push(oblikovati_api::wire::DocumentInfo {
        id: 7,
        name: "Bracket".into(),
        ..Default::default()
    });
    assert_eq!(active_name(&state), "Bracket");
}
