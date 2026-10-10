// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn template_metadata() -> NewFileMetadata {
    NewFileMetadata {
        file_name: path("new.obk").to_str().unwrap().into(),
        template_file: path("a.obk").to_str().unwrap().into(),
        display_name: "New Ω".into(),
        sub_type: "flavored".into(),
    }
}

#[test]
fn template_creation_applies_metadata_and_preserves_disk_and_open_template() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let template_id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.commands(template_id)
        .unwrap()
        .run("edit", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let identity = host.workspace().identity(template_id).unwrap().clone();
    let reads = host.files().reads;
    let created = host
        .create_from_template(
            &template_metadata(),
            RecordingObserver::default(),
            "beta",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_eq!(host.files().reads, reads + 1);
    assert_eq!(host.workspace().display_name(created).unwrap(), "New Ω");
    assert_eq!(host.workspace().identity(template_id).unwrap(), &identity);
    assert!(host.workspace().needs_save(template_id).unwrap());
    assert!(host.workspace().needs_save(created).unwrap());
    assert_eq!(
        host.workspace()
            .session(created)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1.0
    );
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
    host.save(created, "beta", &mut source).unwrap();
    let package =
        DocumentPackage::decode(&host.files().contents.borrow()[&path("new.obk")]).unwrap();
    assert_eq!(package.manifest().display_name, "New Ω");
    assert_eq!(package.manifest().sub_type, "flavored");
    assert_eq!(package.streams()["marker"], [1, 2, 3]);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
}

#[test]
fn empty_overrides_inherit_template_metadata() {
    let mut host = host(RecordingWorkspace::default());
    let template = DocumentPackage::decode(&fixture(1))
        .unwrap()
        .with_copy_metadata(&template_metadata());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), template.encode().unwrap());
    let metadata = NewFileMetadata {
        display_name: String::new(),
        sub_type: String::new(),
        ..template_metadata()
    };
    let mut source = SequenceGuids(1000);
    let id = host
        .create_from_template(&metadata, RecordingObserver::default(), "beta", &mut source)
        .unwrap()
        .session_id;
    host.save(id, "beta", &mut source).unwrap();
    let package =
        DocumentPackage::decode(&host.files().contents.borrow()[&path("new.obk")]).unwrap();
    assert_eq!(package.manifest(), template.manifest());
}

#[test]
fn missing_paths_and_same_template_destination_refuse_before_reads_or_entropy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    for metadata in [
        NewFileMetadata {
            file_name: String::new(),
            ..template_metadata()
        },
        NewFileMetadata {
            template_file: String::new(),
            ..template_metadata()
        },
        NewFileMetadata {
            file_name: path("a.obk").to_str().unwrap().into(),
            ..template_metadata()
        },
    ] {
        let error = host
            .create_from_template(&metadata, RecordingObserver::default(), "beta", &mut source)
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
    }
    assert_eq!(source.0, 1000);
    assert_eq!(host.files().reads, 0);
    assert!(host.workspace().documents().is_empty());
}

#[test]
fn alias_to_template_cannot_become_the_new_destination() {
    let mut files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    files.aliases.insert(path("new.obk"), path("a.obk"));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let error = host
        .create_from_template(
            &template_metadata(),
            RecordingObserver::default(),
            "beta",
            &mut SequenceGuids(1000),
        )
        .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("different file from the source template")
    );
    assert_eq!(host.files().reads, 0);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], fixture(1));
}

#[test]
fn template_read_and_decode_failures_leave_the_workspace_empty() {
    for fail_read in [true, false] {
        let files = WorkspaceFiles {
            fail_read,
            ..Default::default()
        };
        files
            .contents
            .borrow_mut()
            .insert(path("a.obk"), b"not a package".to_vec());
        let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
        let mut source = SequenceGuids(1000);
        let error = host
            .create_from_template(
                &template_metadata(),
                RecordingObserver::default(),
                "beta",
                &mut source,
            )
            .unwrap_err();
        assert!(matches!(
            error,
            oblikovati_session::files::FileWorkspaceError::Load(_)
        ));
        assert_eq!(source.0, 1000);
        assert!(host.workspace().documents().is_empty());
        assert!(host.workspace().observer().notices.is_empty());
    }
}

#[test]
fn native_template_first_save_and_reopen_preserve_template_bytes() {
    let directory = tempfile::tempdir().unwrap();
    let template = directory.path().join("template.obk");
    let destination = directory.path().join("new.obk");
    std::fs::write(&template, fixture(1)).unwrap();
    let metadata = NewFileMetadata {
        file_name: destination.to_str().unwrap().into(),
        template_file: template.to_str().unwrap().into(),
        ..template_metadata()
    };
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), NativeDocumentFiles);
    let mut source = SequenceGuids(1000);
    let id = host
        .create_from_template(&metadata, RecordingObserver::default(), "beta", &mut source)
        .unwrap()
        .session_id;
    assert!(!destination.exists());
    host.close_saving_changes(id, "beta", &mut source).unwrap();
    let reopened = host
        .open(
            &destination,
            RecordingObserver::default(),
            "beta",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(std::fs::read(template).unwrap(), fixture(1));
    assert_ne!(
        host.workspace().identity(reopened).unwrap().internal_name,
        guid(1)
    );
    assert_eq!(host.workspace().identity(reopened).unwrap().save_counter, 1);
    assert!(!host.workspace().needs_save(reopened).unwrap());
}
