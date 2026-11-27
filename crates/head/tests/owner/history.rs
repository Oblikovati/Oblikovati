// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn history_input(key: ViewportKey, pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Key { key, pressed })
}
fn history_command(owner: &mut TestOwner, key: ViewportKey) -> DesktopCommandNotice {
    assert!(owner.process_inputs(&[history_input(key, true)]).unwrap());
    owner.take_command_notice().unwrap()
}
#[test]
fn native_history_restores_the_saved_state_and_redoes_the_edit() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    bind_saved_document(&mut owner, document);
    dirty_property(&mut owner, document);
    let undone = history_command(&mut owner, ViewportKey::Undo);
    assert_eq!(
        undone,
        DesktopCommandNotice::HistoryCompleted {
            document,
            action: DesktopHistoryAction::Undo
        }
    );
    assert!(!owner.documents().workspace().needs_save(document).unwrap());
    let redone = history_command(&mut owner, ViewportKey::Redo);
    assert_eq!(
        redone,
        DesktopCommandNotice::HistoryCompleted {
            document,
            action: DesktopHistoryAction::Redo
        }
    );
    assert!(owner.documents().workspace().needs_save(document).unwrap());
    assert!(redone.message().starts_with("Redo"));
}
#[test]
fn native_history_does_not_change_an_open_transaction() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    dirty_property(&mut owner, document);
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Bounded edit".into(),
        })
        .unwrap();
    let before = Client::new(owner.document_host())
        .transactions()
        .state()
        .unwrap();
    for key in [ViewportKey::Undo, ViewportKey::Redo] {
        assert!(!owner.process_inputs(&[history_input(key, true)]).unwrap());
        assert!(owner.take_command_notice().is_none());
    }
    let after = Client::new(owner.document_host())
        .transactions()
        .state()
        .unwrap();
    assert_eq!(before, after);
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .in_transaction()
    );
}
#[test]
fn history_without_an_active_document_reports_failure_and_release_is_idle() {
    let (mut owner, _) = owner();
    for key in [ViewportKey::Undo, ViewportKey::Redo] {
        let notice = history_command(&mut owner, key);
        assert!(matches!(notice, DesktopCommandNotice::HistoryFailed { .. }));
        assert!(notice.message().contains("expected an active document"));
        assert!(!owner.process_inputs(&[history_input(key, false)]).unwrap());
        assert!(owner.take_command_notice().is_none());
    }
}
