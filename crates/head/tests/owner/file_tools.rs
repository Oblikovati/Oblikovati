// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;

#[test]
fn gui_file_operations_save_close_and_reopen_the_same_sketch_recipe() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let before = Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs::default())
        .unwrap();
    assert!(
        owner
            .complete_save_path(document, Some(std::path::Path::new("/memory/part.opd")))
            .unwrap()
    );
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::Saved { .. })
    ));
    assert!(
        owner
            .close_document_from_gui(&CloseDocumentArgs {
                id: document,
                force: false,
            })
            .unwrap()
    );
    let closed = owner.take_command_notice().unwrap();
    assert_eq!(closed, DesktopCommandNotice::DocumentClosed { document });
    assert!(closed.message().contains("Closed document"));
    assert!(
        owner
            .open_document_from_gui(&OpenDocumentArgs {
                full_document_name: "/memory/part.opd".into(),
                visible: true,
                defer_content: false
            })
            .unwrap()
    );
    let opened = owner.take_command_notice().unwrap();
    assert!(matches!(
        opened,
        DesktopCommandNotice::DocumentOpened { .. }
    ));
    assert!(opened.message().contains("/memory/part.opd"));
    assert_ne!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    let reopened = Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs::default())
        .unwrap();
    assert_eq!(reopened, before);
}
#[test]
fn gui_close_refuses_a_dirty_document_without_discarding_it() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let before = owner.gui_command_state().unwrap();
    assert!(
        owner
            .close_document_from_gui(&CloseDocumentArgs {
                id: document,
                force: false
            })
            .is_err()
    );
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert!(!owner.close_allowed().unwrap());
}
#[test]
fn failed_gui_open_preserves_the_active_document_and_its_history() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    verify_failed_open(&mut owner, "/memory/missing.opd");
}
#[test]
fn unsupported_package_version_keeps_the_previous_document_live() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    owner.documents().files().saved.borrow_mut().insert(
        "/memory/future.opd".into(),
        b"schemaVersion: 999\ndocumentType: 1\n".to_vec(),
    );
    verify_failed_open(&mut owner, "/memory/future.opd");
}
fn verify_failed_open(owner: &mut TestOwner, path: &str) {
    let document = owner.documents().workspace().active_document();
    let before = owner.gui_command_state().unwrap();
    let error = owner
        .open_document_from_gui(&OpenDocumentArgs {
            full_document_name: path.into(),
            visible: true,
            defer_content: false,
        })
        .unwrap_err();
    assert!(
        error.contains(path),
        "open {path:?}: expected an error naming the rejected path, got {error:?}"
    );
    assert_eq!(owner.documents().workspace().active_document(), document);
    assert_eq!(owner.gui_command_state().unwrap(), before);
}
