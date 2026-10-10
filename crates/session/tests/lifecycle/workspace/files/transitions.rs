// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn edit(host: &mut Host, id: u64) {
    host.commands(id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
}
fn refusing_host(observer: RecordingWorkspace) -> Host {
    let files = WorkspaceFiles {
        fail_write: true,
        ..Default::default()
    };
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    FileWorkspace::new(observer, files)
}

#[test]
fn save_as_rebinds_after_success_and_preserves_old_file() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let identity = host.workspace().identity(id).unwrap().internal_name;
    edit(&mut host, id);
    host.save_as(id, &path("new.obk"), "beta", &mut source)
        .unwrap();
    assert_eq!(host.path(id).unwrap(), path("new.obk"));
    assert_eq!(
        host.workspace().identity(id).unwrap().internal_name,
        identity
    );
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
    assert!(!host.workspace().needs_save(id).unwrap());
    assert_eq!(
        host.workspace().observer().notices.last(),
        Some(&WorkspaceNotice::PathChanged(
            id,
            path("a.obk"),
            path("new.obk")
        ))
    );
    assert_eq!(
        open_file(
            &mut host,
            "new.obk",
            &mut source,
            OpenActivation::Background
        ),
        id
    );
    let old_file = open_file(&mut host, "a.obk", &mut source, OpenActivation::Background);
    assert_ne!(old_file, id);
    assert!(host.workspace().needs_save(old_file).unwrap());
}

#[test]
fn destination_owned_by_another_document_is_rejected_before_save_policy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let first = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let second = open_file(&mut host, "b.obk", &mut source, OpenActivation::Background);
    let notices = host.workspace().observer().notices.len();
    let count = source.0;
    let error = host
        .save_as(first, &path("b.obk"), "beta", &mut source)
        .unwrap_err();
    assert!(error.to_string().contains(&format!("document {second}")));
    assert!(std::error::Error::source(&error).is_none());
    assert_eq!(source.0, count);
    assert_eq!(host.workspace().observer().notices.len(), notices);
    assert_eq!(host.path(first).unwrap(), path("a.obk"));
}

#[test]
fn save_as_current_path_is_an_ordinary_save_without_path_notice() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.save_as(id, &path("a.obk"), "beta", &mut source)
        .unwrap();
    assert_eq!(host.path(id).unwrap(), path("a.obk"));
    assert_eq!(
        host.workspace().observer().notices.last(),
        Some(&WorkspaceNotice::Saved(id))
    );
    assert!(
        host.save_as(99, &path("new.obk"), "beta", &mut source)
            .is_err()
    );
}

#[test]
fn failed_save_as_keeps_identity_dirty_state_and_both_file_bindings() {
    let mut host = refusing_host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let before = host.workspace().identity(id).unwrap().clone();
    edit(&mut host, id);
    assert!(
        host.save_as(id, &path("new.obk"), "beta", &mut source)
            .is_err()
    );
    assert_eq!(host.workspace().identity(id).unwrap(), &before);
    assert_eq!(host.path(id).unwrap(), path("a.obk"));
    assert!(host.workspace().needs_save(id).unwrap());
    assert!(
        !host
            .files()
            .contents
            .borrow()
            .contains_key(&path("new.obk"))
    );
    assert_eq!(
        open_file(&mut host, "a.obk", &mut source, OpenActivation::Background),
        id
    );
}

#[test]
fn dirty_close_saves_then_releases_path_with_one_close_policy_call() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    edit(&mut host, id);
    let offset = host.workspace().observer().notices.len();
    let observer = host.close_saving_changes(id, "beta", &mut source).unwrap();
    assert_eq!(
        &host.workspace().observer().notices[offset..],
        [
            WorkspaceNotice::BeforeClose(id),
            WorkspaceNotice::BeforeSave(id),
            WorkspaceNotice::Saved(id),
            WorkspaceNotice::Closed(id, None)
        ]
    );
    assert!(
        matches!(observer.observations.last(), Some(Observation::After(payload, _)) if payload.type_ == EVENT_TRANSACTION_DELETED)
    );
    assert!(host.path(id).is_err());
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(
        width(host.workspace().session(reopened).unwrap().snapshot()),
        2.0
    );
}

#[test]
fn clean_close_does_not_save_or_generate_revisions() {
    let mut host = refusing_host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let count = source.0;
    host.close_saving_changes(id, "beta", &mut source).unwrap();
    assert_eq!(source.0, count);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
    assert!(host.workspace().documents().is_empty());
    assert!(host.close_saving_changes(id, "beta", &mut source).is_err());
}

#[test]
fn close_and_save_vetoes_leave_dirty_documents_open() {
    for observer in [
        RecordingWorkspace {
            deny_close: Some(1),
            ..Default::default()
        },
        RecordingWorkspace {
            deny_save: Some(1),
            ..Default::default()
        },
    ] {
        let mut host = host(observer);
        let mut source = SequenceGuids(1000);
        let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
        edit(&mut host, id);
        let count = source.0;
        assert!(host.close_saving_changes(id, "beta", &mut source).is_err());
        assert_eq!(source.0, count);
        assert_eq!(host.path(id).unwrap(), path("a.obk"));
        assert!(host.workspace().needs_save(id).unwrap());
    }
}

#[test]
fn storage_failure_and_open_preview_do_not_close_or_drop_edits() {
    let mut host = refusing_host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    edit(&mut host, id);
    let error = host
        .close_saving_changes(id, "beta", &mut source)
        .err()
        .unwrap();
    assert!(error.to_string().contains("write denied"));
    host.commands(id).unwrap().begin("preview").unwrap();
    let notices = host.workspace().observer().notices.len();
    let error = host
        .close_saving_changes(id, "beta", &mut source)
        .err()
        .unwrap();
    assert!(error.to_string().contains("bounded transactions closed"));
    assert_eq!(host.workspace().observer().notices.len(), notices);
    assert!(host.workspace().session(id).unwrap().in_transaction());
    assert_eq!(width(host.workspace().session(id).unwrap().snapshot()), 2.0);
}

#[test]
fn native_save_as_creates_new_destination_then_clean_close_keeps_old_file() {
    let directory = tempfile::tempdir().unwrap();
    let original = directory.path().join("original.obk");
    let renamed = directory.path().join("renamed part.obk");
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
    host.save_as(id, &renamed, "beta", &mut source).unwrap();
    assert_eq!(
        host.path(id).unwrap(),
        std::fs::canonicalize(&renamed).unwrap()
    );
    assert_eq!(std::fs::read(&original).unwrap(), fixture(1));
    host.close_saving_changes(id, "beta", &mut source).unwrap();
    assert!(renamed.is_file());
}

#[test]
fn derived_names_follow_repeated_save_as_but_custom_names_survive() {
    for persisted in ["a", "Custom Ω"] {
        let mut host = host(RecordingWorkspace::default());
        let mut bytes = fixture(1);
        bytes.extend_from_slice(format!("displayName: '{persisted}'\n").as_bytes());
        host.files()
            .contents
            .borrow_mut()
            .insert(path("a.obk"), bytes);
        let mut source = SequenceGuids(1000);
        let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
        assert_eq!(host.workspace().display_name(id).unwrap(), persisted);
        for destination in ["new.obk", "final.obk"] {
            host.save_as(id, &path(destination), "beta", &mut source)
                .unwrap();
            let expected = if persisted == "a" {
                destination.trim_end_matches(".obk")
            } else {
                persisted
            };
            assert_eq!(host.workspace().display_name(id).unwrap(), expected);
        }
        host.close_discarding_changes(id).unwrap();
        assert!(host.workspace().display_name(id).is_err());
        let reopened = open_file(
            &mut host,
            "final.obk",
            &mut source,
            OpenActivation::Activate,
        );
        assert_eq!(
            host.workspace().display_name(reopened).unwrap(),
            if persisted == "a" { "final" } else { persisted }
        );
    }
}
