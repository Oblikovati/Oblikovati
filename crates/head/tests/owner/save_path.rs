// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::path::Path;
#[test]
fn selected_save_path_keeps_its_document_when_focus_changes() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    assert_eq!(
        command_save(&mut owner),
        DesktopCommandNotice::SavePathRequired { document: first }
    );
    let second = create(&mut owner, &awake);
    choose_path(&mut owner, first, "/memory/first.opd");
    assert_saved_choice(&mut owner, first, "/memory/first.opd");
    let workspace = owner.documents().workspace();
    assert_eq!(workspace.active_document(), Some(second));
    assert!(!workspace.needs_save(first).unwrap());
    assert!(workspace.needs_save(second).unwrap());
    assert!(owner.documents().path(second).is_err());
}
fn assert_saved_choice(owner: &mut TestOwner, document: u64, path: &str) {
    assert_eq!(
        owner.take_command_notice().unwrap(),
        DesktopCommandNotice::Saved {
            document,
            path: path.into()
        }
    );
    assert_eq!(owner.documents().path(document).unwrap(), Path::new(path));
}
#[test]
fn cancellation_and_closed_request_leave_files_and_other_documents_intact() {
    let (mut owner, awake) = owner();
    let first = create(&mut owner, &awake);
    verify_cancelled_path(&mut owner, first);
    let second = create(&mut owner, &awake);
    close_requested_document(&mut owner, first);
    choose_path(&mut owner, first, "/memory/closed.opd");
    assert_failed_choice(&mut owner, first);
    assert!(owner.documents().files().saved.borrow().is_empty());
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(second)
    );
}

#[test]
fn failed_choice_can_retry_without_binding_a_path_or_losing_history() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    dirty_property(&mut owner, document);
    let before = owner.gui_command_state().unwrap();
    owner.documents().files().fail_stage.set(true);
    choose_path(&mut owner, document, "/memory/retry.opd");
    assert_failed_choice(&mut owner, document);
    assert!(owner.documents().path(document).is_err());
    assert_eq!(owner.gui_command_state().unwrap(), before);
    owner.documents().files().fail_stage.set(false);
    choose_path(&mut owner, document, "/memory/retry.opd");
    assert_saved_choice(&mut owner, document, "/memory/retry.opd");
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert!(!owner.documents().workspace().needs_save(document).unwrap());
}

fn choose_path(owner: &mut TestOwner, document: u64, path: &str) {
    assert!(
        owner
            .complete_save_path(document, Some(Path::new(path)))
            .unwrap()
    );
}
fn assert_failed_choice(owner: &mut TestOwner, document: u64) -> String {
    match owner.take_command_notice().unwrap() {
        DesktopCommandNotice::SaveFailed {
            document: actual,
            cause,
        } => {
            assert_eq!(actual, document);
            cause
        }
        notice => panic!("save choice {notice:?}: expected SaveFailed for document {document}"),
    }
}
fn verify_cancelled_path(owner: &mut TestOwner, document: u64) {
    let before = owner.gui_command_state().unwrap();
    assert!(!owner.complete_save_path(document, None).unwrap());
    assert!(owner.take_command_notice().is_none());
    assert_eq!(owner.gui_command_state().unwrap(), before);
}
fn close_requested_document(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
}
#[test]
fn selected_path_refuses_open_edits_without_ending_or_saving_them() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Pending".into(),
        })
        .unwrap();
    let before = owner.gui_command_state().unwrap();
    choose_path(&mut owner, document, "/memory/pending.opd");
    assert!(assert_failed_choice(&mut owner, document).contains("transaction"));
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert!(owner.documents().files().saved.borrow().is_empty());
    assert!(owner.documents().path(document).is_err());
}
#[cfg(unix)]
#[test]
fn non_unicode_selected_path_is_a_named_decline_before_file_io() {
    use std::os::unix::ffi::OsStringExt;
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let path = std::path::PathBuf::from(std::ffi::OsString::from_vec(vec![b'/', 0xff]));
    assert!(owner.complete_save_path(document, Some(&path)).unwrap());
    assert!(assert_failed_choice(&mut owner, document).contains("expected a UTF-8"));
    assert!(owner.documents().files().saved.borrow().is_empty());
}
#[test]
fn script_owner_forwards_choices_and_idle_owner_refuses_them() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let engine =
        oblikovati_script::IsolatedLuaEngine::new(&std::env::current_exe().unwrap(), Vec::new());
    let mut desktop =
        crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    assert!(
        desktop
            .complete_save_path(document, Some(Path::new("/memory/script-choice.opd")))
            .unwrap()
    );
    assert_saved_choice(
        desktop.document_owner_mut(),
        document,
        "/memory/script-choice.opd",
    );
    let mut idle = crate::work::IdleHeadWork;
    assert!(!idle.complete_save_path(document, None).unwrap());
    assert!(
        idle.complete_save_path(document, Some(Path::new("/memory/idle.opd")))
            .unwrap_err()
            .contains("expected a document-save owner")
    );
}
