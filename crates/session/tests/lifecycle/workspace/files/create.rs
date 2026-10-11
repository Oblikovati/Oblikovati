// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn create_recipe(
    host: &mut Host,
    source: &mut dyn GuidSource,
) -> Result<RegisteredDocument, oblikovati_session::files::FileWorkspaceError> {
    host.create(
        &path("new.obk"),
        DocumentPackage::decode(&fixture(1)).unwrap(),
        RecordingObserver::default(),
        "beta",
        source,
    )
}

#[test]
fn creation_mints_unsaved_identity_and_emits_created_then_activation() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let opened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let start = host.workspace().observer().notices.len();
    let created = create_recipe(&mut host, &mut source).unwrap();
    let id = created.session_id;
    let identity = host.workspace().identity(id).unwrap();
    assert_ne!(identity.internal_name, guid(1));
    assert_eq!(identity.save_counter, 0);
    assert_eq!(identity.version_created, "beta");
    assert!(identity.version_saved.is_empty());
    assert!(identity.revision_id.is_none());
    assert!(identity.database_revision_id.is_none());
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(host.workspace().active_document(), Some(id));
    assert_eq!(host.workspace().documents(), [opened, id]);
    assert_eq!(
        &host.workspace().observer().notices[start..],
        [
            WorkspaceNotice::Created(id, IdentityRegistration::Preserved),
            WorkspaceNotice::Activated(id)
        ]
    );
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("new.obk"))
    );
    assert!(
        host.workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn undo_to_initial_recipe_stays_dirty_until_first_successful_save() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = create_recipe(&mut host, &mut source).unwrap().session_id;
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    host.commands(id).unwrap().undo().unwrap();
    assert!(host.workspace().needs_save(id).unwrap());
    host.save(id, "gamma", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    let identity = host.workspace().identity(id).unwrap();
    assert_eq!(identity.save_counter, 1);
    assert_eq!(identity.version_created, "beta");
    assert_eq!(identity.version_saved, "gamma");
    let saved = DocumentPackage::decode(&host.files().contents.borrow()[&path("new.obk")]).unwrap();
    assert_eq!(
        saved
            .identity_record()
            .unwrap()
            .unwrap()
            .restore(identity)
            .unwrap(),
        *identity
    );
    assert_eq!(saved.streams()["marker"], [1, 2, 3]);
}

#[test]
fn unchanged_creation_is_saved_before_close_and_failed_write_keeps_it_open() {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut source = SequenceGuids(1000);
    let id = create_recipe(&mut host, &mut source).unwrap().session_id;
    assert!(host.close_saving_changes(id, "beta", &mut source).is_err());
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(host.workspace().documents(), [id]);
    assert_eq!(host.workspace().identity(id).unwrap().save_counter, 0);
    host.close_discarding_changes(id).unwrap();
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("new.obk"))
    );
}

#[test]
fn duplicate_destination_refuses_creation_without_entropy_or_events() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = create_recipe(&mut host, &mut source).unwrap().session_id;
    let count = source.0;
    let notices = host.workspace().observer().notices.len();
    assert!(
        create_recipe(&mut host, &mut source)
            .unwrap_err()
            .to_string()
            .contains("owned by an open document")
    );
    assert_eq!(source.0, count);
    assert_eq!(host.workspace().observer().notices.len(), notices);
    assert_eq!(host.workspace().documents(), [first]);
}

#[test]
fn invalid_recipe_and_template_identity_never_publish_a_document() {
    let mut host = host(RecordingWorkspace::default());
    for yaml in [
        "schemaVersion: 2\ndocumentType: 0\n",
        "schemaVersion: 2\ndocumentType: 999\n",
        "schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: [{name: p, kind: user, expression: '???'}]}\n",
        "schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: invalid}\n",
        "schemaVersion: 2\ndocumentType: 1\nidentity: {unexpected: value}\n",
    ] {
        let package = DocumentPackage::decode(yaml.as_bytes()).unwrap();
        let error = host
            .create(
                &path("new.obk"),
                package,
                RecordingObserver::default(),
                "beta",
                &mut SequenceGuids(1000),
            )
            .unwrap_err();
        assert!(!error.to_string().is_empty());
        assert!(host.workspace().documents().is_empty());
        assert!(host.workspace().observer().notices.is_empty());
    }
}

#[test]
fn template_guid_reuse_and_live_guid_collision_are_refused() {
    let mut host = host(RecordingWorkspace::default());
    assert!(
        create_recipe(&mut host, &mut SequenceGuids(0))
            .unwrap_err()
            .to_string()
            .contains("distinct from the new document")
    );
    let id = open_file(
        &mut host,
        "b.obk",
        &mut SequenceGuids(1000),
        OpenActivation::Activate,
    );
    assert!(
        create_recipe(&mut host, &mut SequenceGuids(1))
            .unwrap_err()
            .to_string()
            .contains("fresh identity")
    );
    assert_eq!(host.workspace().documents(), [id]);
    assert_eq!(host.workspace().active_document(), Some(id));
}

#[test]
fn native_creation_close_save_and_reopen_preserve_recipe_and_new_identity() {
    let directory = tempfile::tempdir().unwrap();
    let destination = directory.path().join("new.obk");
    let package = DocumentPackage::decode(&fixture(1)).unwrap();
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .create(
            &destination,
            package,
            RecordingObserver::default(),
            "beta",
            &mut source,
        )
        .unwrap()
        .session_id;
    let guid = host.workspace().identity(id).unwrap().internal_name;
    assert!(!destination.exists());
    host.close_saving_changes(id, "beta", &mut source).unwrap();
    assert!(destination.is_file());
    let opened = host
        .open(
            &destination,
            RecordingObserver::default(),
            "beta",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(
        host.workspace().identity(opened).unwrap().internal_name,
        guid
    );
    assert_eq!(host.workspace().identity(opened).unwrap().save_counter, 1);
    assert!(!host.workspace().needs_save(opened).unwrap());
    assert_eq!(
        host.workspace()
            .session(opened)
            .unwrap()
            .snapshot()
            .parameters
            .ids(),
        [1]
    );
}

#[test]
fn creation_accepts_an_identity_free_package_and_reports_generator_failure() {
    let mut host = host(RecordingWorkspace::default());
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let mut failing = WorkspaceGuidSource {
        result: Err(IdentityTransitionError::Randomness("offline".into())),
        calls: 0,
    };
    let error = host
        .create(
            &path("new.obk"),
            package.clone(),
            RecordingObserver::default(),
            "beta",
            &mut failing,
        )
        .unwrap_err();
    assert!(error.to_string().contains("offline"));
    assert!(host.workspace().documents().is_empty());
    let created = host
        .create(
            &path("new.obk"),
            package,
            RecordingObserver::default(),
            "beta",
            &mut SequenceGuids(1000),
        )
        .unwrap();
    assert_eq!(created.session_id, 1);
}

#[path = "save_names.rs"]
mod save_names;
