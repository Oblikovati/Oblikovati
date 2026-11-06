// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn copy_package(host: &Host) -> DocumentPackage {
    DocumentPackage::decode(&host.files().contents.borrow()[&path("copy.obk")]).unwrap()
}

#[test]
fn clean_copy_has_fresh_file_identity_but_preserves_known_model_revision() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save(id, "alpha", &mut source).unwrap();
    let before = host.workspace().identity(id).unwrap().clone();
    let notices = host.workspace().observer().notices.len();
    let copy = host
        .save_copy(id, &path("copy.obk"), None, "beta", &mut source)
        .unwrap();
    assert_ne!(copy.internal_name, before.internal_name);
    assert_ne!(copy.revision_id, before.revision_id);
    assert_eq!(copy.database_revision_id, before.database_revision_id);
    assert_eq!(copy.save_counter, 1);
    assert_eq!(host.workspace().identity(id).unwrap(), &before);
    assert_eq!(host.workspace().observer().notices.len(), notices);
    assert!(!host.workspace().needs_save(id).unwrap());
    assert_eq!(
        copy_package(&host)
            .identity_record()
            .unwrap()
            .unwrap()
            .restore(&before)
            .unwrap(),
        copy
    );
}

#[test]
fn dirty_copy_uses_current_recipe_and_new_model_revision_without_saving_source() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save(id, "alpha", &mut source).unwrap();
    let before = host.workspace().identity(id).unwrap().clone();
    let bytes = host.files().contents.borrow()[&path("a.obk")].clone();
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let generation = host.workspace().session(id).unwrap().generation();
    let copy = host
        .save_copy(id, &path("copy.obk"), None, "beta", &mut source)
        .unwrap();
    assert_ne!(copy.database_revision_id, before.database_revision_id);
    assert_eq!(
        copy.model_digest,
        copy_package(&host).recipe_digest().unwrap()
    );
    assert_eq!(
        copy_package(&host)
            .restore_document_parameters()
            .unwrap()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(host.workspace().identity(id).unwrap(), &before);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], bytes);
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(
        host.workspace().session(id).unwrap().generation(),
        generation
    );
    assert_eq!(
        host.workspace().session(id).unwrap().undo_labels(),
        ["edit"]
    );
    assert_eq!(host.path(id).unwrap(), path("a.obk"));
}

#[test]
fn copy_metadata_changes_only_copy_manifest_and_inherits_empty_fields() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save(id, "alpha", &mut source).unwrap();
    let metadata = NewFileMetadata {
        display_name: "Copy Ω".into(),
        sub_type: "flavored".into(),
        file_name: "ignored".into(),
        template_file: "not-read".into(),
    };
    host.save_copy(id, &path("copy.obk"), Some(&metadata), "beta", &mut source)
        .unwrap();
    assert_eq!(copy_package(&host).manifest().display_name, "Copy Ω");
    assert_eq!(copy_package(&host).manifest().sub_type, "flavored");
    host.save(id, "beta", &mut source).unwrap();
    let original =
        DocumentPackage::decode(&host.files().contents.borrow()[&path("a.obk")]).unwrap();
    assert_eq!(original.manifest().display_name, "a");
    let inherited = copy_package(&host).with_copy_metadata(&NewFileMetadata::default());
    assert_eq!(inherited.manifest(), copy_package(&host).manifest());
    assert_eq!(
        inherited.recipe_digest().unwrap(),
        copy_package(&host).recipe_digest().unwrap()
    );
}

#[test]
fn every_open_destination_including_source_is_rejected_without_entropy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let _other = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    let count = source.0;
    for target in [path("a.obk"), path("b.obk")] {
        let error = host
            .save_copy(id, &target, None, "beta", &mut source)
            .unwrap_err();
        assert!(error.to_string().contains("owned by an open document"));
    }
    assert_eq!(source.0, count);
    assert!(
        host.save_copy(99, &path("copy.obk"), None, "beta", &mut source)
            .is_err()
    );
}

#[test]
fn open_transaction_and_guid_failure_never_create_copy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.commands(id).unwrap().begin("preview").unwrap();
    assert!(
        host.save_copy(id, &path("copy.obk"), None, "beta", &mut source)
            .is_err()
    );
    host.commands(id).unwrap().abort().unwrap();
    let error = host
        .save_copy(id, &path("copy.obk"), None, "beta", &mut SequenceGuids(0))
        .unwrap_err();
    assert!(error.to_string().contains("fresh identity"));
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("copy.obk"))
    );
    assert_eq!(
        host.workspace().identity(id).unwrap().internal_name,
        guid(1)
    );
}

#[test]
fn failed_copy_write_keeps_dirty_source_and_original_file() {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let before = host.workspace().identity(id).unwrap().clone();
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let error = host
        .save_copy(id, &path("copy.obk"), None, "beta", &mut source)
        .unwrap_err();
    assert!(error.to_string().contains("write denied"));
    assert_eq!(host.workspace().identity(id).unwrap(), &before);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
    assert!(host.workspace().needs_save(id).unwrap());
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("copy.obk"))
    );
}

#[test]
fn native_copy_opens_as_an_independent_document_and_preserves_source_bytes() {
    let directory = tempfile::tempdir().unwrap();
    let original = directory.path().join("source.obk");
    let destination = directory.path().join("copy.obk");
    std::fs::write(&original, fixture(1)).unwrap();
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .open(
            &original,
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let copy = host
        .save_copy(id, &destination, None, "beta", &mut source)
        .unwrap();
    assert_eq!(std::fs::read(&original).unwrap(), fixture(1));
    let opened = host
        .open(
            &destination,
            RecordingObserver::default(),
            "beta",
            &mut source,
            OpenActivation::Background,
        )
        .unwrap();
    assert!(matches!(
        opened,
        FileOpenOutcome::Opened(RegisteredDocument {
            identity: IdentityRegistration::Preserved,
            ..
        })
    ));
    assert_eq!(
        host.workspace().identity(opened.session_id()).unwrap(),
        &copy
    );
    assert_eq!(host.workspace().active_document(), Some(id));
}
