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

fn verify_script_owner_command_state(mut owner: TestOwner) -> TestOwner {
    let expected = owner.gui_command_state().unwrap();
    let engine =
        oblikovati_script::IsolatedLuaEngine::new(&std::env::current_exe().unwrap(), Vec::new());
    let mut desktop =
        crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    assert_eq!(desktop.gui_command_state().unwrap(), expected);
    let mut idle = crate::work::IdleHeadWork;
    assert_eq!(
        idle.gui_command_state().unwrap(),
        GuiCommandState::default()
    );
    desktop.into_document_owner()
}

fn verify_document_focus_state(owner: &mut TestOwner, awake: &Receiver<()>, first: u64) {
    let previous = owner.gui_command_state().unwrap();
    let second = create(owner, awake);
    let second_active = owner.gui_command_state().unwrap();
    assert_eq!(second_active.documents.len(), 2);
    assert!(
        second_active
            .documents
            .iter()
            .any(|tab| tab.id == second && tab.active)
    );
    assert!(
        owner
            .apply_gui_shell_action(GuiShellAction::ActivateDocument(first))
            .unwrap()
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .documents
            .iter()
            .any(|tab| tab.id == first && tab.active)
    );
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

#[test]
fn gui_model_browser_state_projects_the_typed_model_tree() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    let expected = Client::new(owner.document_host()).model().tree().unwrap();
    let actual = owner.gui_command_state().unwrap().model_tree.unwrap();
    assert_eq!(actual.document, expected.document);
    assert_eq!(actual.parameters, expected.parameters.unwrap_or_default());
    assert_eq!(actual.sketches, usize::try_from(expected.sketches).unwrap());
    let expected_bodies = Client::new(owner.document_host())
        .body()
        .list(&oblikovati_api::json::EmptyObject {})
        .unwrap()
        .bodies
        .unwrap_or_default();
    assert_eq!(
        actual.bodies.len(),
        usize::try_from(expected.bodies).unwrap()
    );
    assert_eq!(actual.bodies.len(), expected_bodies.len());
    let expected_features = expected.features.unwrap_or_default();
    assert_eq!(actual.features.len(), expected_features.len());
    for (actual, expected) in actual.features.iter().zip(expected_features) {
        assert_eq!(actual.name, expected.name);
        assert_eq!(actual.kind, expected.kind);
        assert_eq!(actual.health, expected.health);
    }
}
