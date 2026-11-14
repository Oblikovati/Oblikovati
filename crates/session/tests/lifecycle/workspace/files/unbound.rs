// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::{BatchSaveArgs, BatchSaveItem};
use oblikovati_session::files::FileWorkspaceError;

fn new_unbound(host: &mut Host, source: &mut SequenceGuids) -> u64 {
    host.create_unbound(
        DocumentPackage::decode(&fixture(1))
            .unwrap()
            .with_display_name("Untitled"),
        RecordingObserver::default(),
        "alpha",
        source,
    )
    .unwrap()
    .session_id
}

#[test]
fn unbound_documents_have_independent_identity_and_no_false_path() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = new_unbound(&mut host, &mut source);
    let second = new_unbound(&mut host, &mut source);
    assert_ne!(
        host.workspace().identity(first).unwrap(),
        host.workspace().identity(second).unwrap()
    );
    assert_eq!(host.document_info(first).unwrap().name, "Untitled");
    assert_eq!(host.document_info(second).unwrap().name, "Untitled");
    assert!(
        matches!(host.path(first), Err(FileWorkspaceError::UnboundDocument(id)) if id == first)
    );
    assert!(
        host.save(first, "alpha", &mut source)
            .unwrap_err()
            .to_string()
            .contains("Save As")
    );
    assert!(
        host.close_saving_changes(first, "alpha", &mut source)
            .is_err()
    );
    assert_eq!(host.workspace().documents(), [first, second]);
    host.close_discarding_changes(first).unwrap();
    assert_eq!(host.workspace().documents(), [second]);
    assert!(matches!(
        host.path(first),
        Err(FileWorkspaceError::MissingFile(_))
    ));
}

#[test]
fn first_save_as_binds_only_after_write_and_reopens_the_same_recipe() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = new_unbound(&mut host, &mut source);
    let identity = host.workspace().identity(id).unwrap().internal_name;
    host.commands(id)
        .unwrap()
        .run("Resize", |snapshot, _| {
            snapshot.parameters.set_expression(1, "3 cm")
        })
        .unwrap();
    host.save_as(id, &path("first.opd"), "beta", &mut source)
        .unwrap();
    assert_eq!(host.path(id).unwrap(), path("first.opd"));
    assert_eq!(
        host.workspace().observer().notices.last(),
        Some(&WorkspaceNotice::PathChanged(
            id,
            PathBuf::new(),
            path("first.opd")
        ))
    );
    assert!(!host.workspace().needs_save(id).unwrap());
    assert_eq!(
        host.workspace().identity(id).unwrap().internal_name,
        identity
    );
    assert_eq!(host.document_info(id).unwrap().name, "Untitled");
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(
        &mut host,
        "first.opd",
        &mut source,
        OpenActivation::Activate,
    );
    assert_eq!(
        host.workspace().identity(reopened).unwrap().internal_name,
        identity
    );
    assert_eq!(host.document_info(reopened).unwrap().name, "Untitled");
    let saved =
        DocumentPackage::decode(&host.files().contents.borrow()[&path("first.opd")]).unwrap();
    assert!(
        String::from_utf8(saved.encode().unwrap())
            .unwrap()
            .contains("3 cm")
    );
}

#[test]
fn failed_first_save_as_preserves_unbound_dirty_identity_and_title() {
    let mut host = FileWorkspace::new(
        RecordingWorkspace::default(),
        WorkspaceFiles {
            fail_write: true,
            ..Default::default()
        },
    );
    let mut source = SequenceGuids(1000);
    let id = new_unbound(&mut host, &mut source);
    let identity = host.workspace().identity(id).unwrap().clone();
    assert!(
        host.save_as(id, &path("failed.opd"), "beta", &mut source)
            .is_err()
    );
    assert!(matches!(
        host.path(id),
        Err(FileWorkspaceError::UnboundDocument(_))
    ));
    assert_eq!(host.workspace().identity(id).unwrap(), &identity);
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(host.document_info(id).unwrap().name, "Untitled");
    assert!(host.files().contents.borrow().is_empty());
}

#[test]
fn unbound_copy_preserves_source_and_batch_save_as_assigns_first_destination() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = new_unbound(&mut host, &mut source);
    let identity = host.workspace().identity(id).unwrap().clone();
    let copy = host
        .save_copy(id, &path("copy.opd"), None, "beta", &mut source)
        .unwrap();
    assert_ne!(copy.internal_name, identity.internal_name);
    assert_eq!(host.workspace().identity(id).unwrap(), &identity);
    assert!(host.path(id).is_err());
    assert!(host.workspace().needs_save(id).unwrap());
    for (operation, saved) in [("save", 0), ("saveAs", 1), ("save", 1)] {
        let result = host
            .batch_save(
                BatchSaveArgs {
                    operation: operation.into(),
                    items: Some(vec![BatchSaveItem {
                        document: id,
                        target_file_name: path("batch.opd").to_str().unwrap().into(),
                    }]),
                },
                "beta",
                &mut source,
            )
            .unwrap();
        assert_eq!(result.saved, saved);
    }
    assert_eq!(host.path(id).unwrap(), path("batch.opd"));
}
