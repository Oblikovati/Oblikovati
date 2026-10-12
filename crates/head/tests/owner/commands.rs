// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};
fn save_input(pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::Key {
        key: ViewportKey::Save,
        pressed,
    })
}
fn command_save(owner: &mut TestOwner) -> DesktopCommandNotice {
    assert!(owner.process_inputs(&[save_input(true)]).unwrap());
    let notice = owner.take_command_notice().unwrap();
    assert!(owner.take_command_notice().is_none());
    notice
}
fn bind_saved_document(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/save.opd".into(),
        })
        .unwrap();
}
fn dirty_property(owner: &mut TestOwner, document: u64) {
    use oblikovati_api::types::{Variant, VariantValue};
    Client::new(owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: oblikovati_model::property::USER_DEFINED.into(),
            name: "native-save".into(),
            value: Variant {
                value: VariantValue::String("edited".into()),
                ..Default::default()
            },
        })
        .unwrap();
}
#[test]
fn save_without_a_document_or_path_reports_the_required_state() {
    let (mut owner, awake) = owner();
    assert_eq!(
        command_save(&mut owner),
        DesktopCommandNotice::NoActiveDocument
    );
    let document = create(&mut owner, &awake);
    assert_eq!(
        command_save(&mut owner),
        DesktopCommandNotice::SavePathRequired { document }
    );
    assert!(!owner.close_allowed().unwrap());
    let release = [save_input(false), ViewportEvent::Expose];
    assert!(!owner.process_inputs(&release).unwrap());
    assert!(owner.take_command_notice().is_none());
}
#[test]
fn save_writes_the_active_dirty_document_and_clears_the_close_blocker() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    bind_saved_document(&mut owner, document);
    dirty_property(&mut owner, document);
    assert!(!owner.close_allowed().unwrap());
    let notice = command_save(&mut owner);
    assert_eq!(
        notice,
        DesktopCommandNotice::Saved {
            document,
            path: "/memory/save.opd".into()
        }
    );
    assert!(owner.close_allowed().unwrap());
    assert!(notice.message().contains("/memory/save.opd"));
}
#[test]
fn failed_save_retains_the_committed_file_and_dirty_document_for_retry() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    bind_saved_document(&mut owner, document);
    let committed = owner.documents().files().saved.borrow().clone();
    dirty_property(&mut owner, document);
    owner.documents().files().fail_stage.set(true);
    let notice = command_save(&mut owner);
    assert!(
        matches!(notice, DesktopCommandNotice::SaveFailed { document: id, .. } if id == document)
    );
    assert!(notice.message().contains("expected writable destination"));
    assert_eq!(*owner.documents().files().saved.borrow(), committed);
    assert!(!owner.close_allowed().unwrap());
    owner.documents().files().fail_stage.set(false);
    let retry = command_save(&mut owner);
    assert!(matches!(retry, DesktopCommandNotice::Saved { .. }));
    assert!(owner.close_allowed().unwrap());
}
#[test]
fn command_feedback_names_missing_context() {
    let cases = [
        (
            DesktopCommandNotice::NoActiveDocument,
            "Save: no active document",
        ),
        (
            DesktopCommandNotice::SavePathRequired { document: 42 },
            "42",
        ),
        (
            DesktopCommandNotice::SaveFailed {
                document: 42,
                cause: "fault".into(),
            },
            "fault",
        ),
    ];
    for (notice, expected) in cases {
        assert!(notice.message().contains(expected));
    }
}

#[test]
fn saving_an_active_bound_document_keeps_another_unsaved_document_dirty() {
    let (mut owner, awake) = owner();
    let saved = create(&mut owner, &awake);
    bind_saved_document(&mut owner, saved);
    dirty_property(&mut owner, saved);
    let unsaved = create(&mut owner, &awake);
    let activation = ActivateDocumentArgs { id: saved };
    Client::new(owner.document_host())
        .documents()
        .activate(&activation)
        .unwrap();
    assert!(
        matches!(command_save(&mut owner), DesktopCommandNotice::Saved { document, .. } if document == saved)
    );
    let workspace = owner.documents().workspace();
    assert!(!workspace.needs_save(saved).unwrap());
    assert!(workspace.needs_save(unsaved).unwrap());
    assert!(owner.documents().path(unsaved).is_err());
}

#[path = "history.rs"]
mod history;
