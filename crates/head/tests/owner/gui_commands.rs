// SPDX-License-Identifier: GPL-2.0-only
use super::commands::dirty_property;
use super::*;
use oblikovati_gui::{GuiCommandState, GuiShellAction};
#[test]
fn gui_command_metadata_follows_the_active_owner_history() {
    let (mut owner, awake) = owner();
    assert_eq!(
        owner.gui_command_state().unwrap(),
        GuiCommandState::default()
    );
    let document = create(&mut owner, &awake);
    let active = owner.gui_command_state().unwrap();
    assert_eq!(
        active
            .model_tree
            .as_ref()
            .map(|tree| tree.document.as_str()),
        Some("Owner part")
    );
    dirty_property(&mut owner, document);
    let edited = owner.gui_command_state().unwrap();
    assert!(edited.undo_label.is_some() && edited.redo_label.is_none());
    verify_history_state_moves(&mut owner, &edited);
    verify_transaction_metadata(&mut owner);
    let mut owner = verify_script_owner_command_state(owner);
    verify_document_focus_state(&mut owner, &awake, document);
}
fn verify_history_state_moves(owner: &mut TestOwner, edited: &GuiCommandState) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    let undone = owner.gui_command_state().unwrap();
    assert!(undone.undo_label.is_none());
    assert_eq!(undone.redo_label, edited.undo_label);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(owner.gui_command_state().unwrap(), *edited);
}
fn verify_transaction_metadata(owner: &mut TestOwner) {
    let before = owner.gui_command_state().unwrap();
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Pending edit".into(),
        })
        .unwrap();
    let pending = owner.gui_command_state().unwrap();
    assert!(pending.transaction_open);
    assert_eq!(pending.undo_label, before.undo_label);
    assert!(GuiShellAction::Undo.enabled(&pending));
    assert!(!GuiShellAction::Undo.dispatchable(&pending));
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(owner.gui_command_state().unwrap(), before);
}

fn verify_script_owner_command_state(owner: TestOwner) -> TestOwner {
    let expected = owner.gui_command_state().unwrap();
    let engine =
        oblikovati_script::IsolatedLuaEngine::new(&std::env::current_exe().unwrap(), Vec::new());
    let desktop = crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    assert_eq!(desktop.gui_command_state().unwrap(), expected);
    assert_eq!(
        crate::work::IdleHeadWork.gui_command_state().unwrap(),
        GuiCommandState::default()
    );
    desktop.into_document_owner()
}

fn verify_document_focus_state(owner: &mut TestOwner, awake: &Receiver<()>, first: u64) {
    let previous = owner.gui_command_state().unwrap();
    let second = create(owner, awake);
    assert!(owner.gui_command_state().unwrap().model_tree.is_some());
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: second,
            force: true,
        })
        .unwrap();
    assert_eq!(owner.gui_command_state().unwrap(), previous);
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: first,
            force: true,
        })
        .unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap(),
        GuiCommandState::default()
    );
}
