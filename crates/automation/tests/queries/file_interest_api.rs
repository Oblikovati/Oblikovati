// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn request(document: u64, name: &str, version: i64) -> AddDocumentInterestArgs {
    AddDocumentInterestArgs {
        document,
        interest: DocumentInterestRecord {
            client_id: "cam".into(),
            name: name.into(),
            data_version: version,
            client_data: "µm\0".into(),
            ..Default::default()
        },
    }
}

#[test]
fn typed_interest_commands_preserve_order_and_survive_undo_save_and_reopen() {
    let (mut files, id, mut guids) = created_files(false);
    let list = ListDocumentInterestsArgs { document: id };
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert_eq!(
        client.documents().interests(&list).unwrap().interests,
        Some(Vec::new())
    );
    for (name, version) in [("paths", 1), ("setup", 2), ("paths", 3)] {
        assert!(
            client
                .documents()
                .add_interest(&request(id, name, version))
                .unwrap()
                .ok
        );
    }
    let records = client
        .documents()
        .interests(&list)
        .unwrap()
        .interests
        .unwrap();
    assert_eq!(
        records
            .iter()
            .map(|record| (record.name.as_str(), record.data_version))
            .collect::<Vec<_>>(),
        [("paths", 3), ("setup", 2)]
    );
    assert!(
        records
            .iter()
            .all(|record| record.interest_type == INTERESTED)
    );
    for (probe, expected) in [("cam", true), ("paths", true), ("CAM", false)] {
        assert_eq!(
            client
                .documents()
                .has_interest(&HasDocumentInterestArgs {
                    document: id,
                    client: probe.into()
                })
                .unwrap()
                .has_interest,
            expected
        );
    }
    assert!(
        client
            .documents()
            .remove_interest(&RemoveDocumentInterestArgs {
                document: id,
                client_id: "cam".into(),
                name: "paths".into()
            })
            .unwrap()
            .ok
    );
    files.commands(id).unwrap().undo().unwrap();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert_eq!(
        client
            .documents()
            .interests(&list)
            .unwrap()
            .interests
            .unwrap(),
        records
    );
    client
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 1,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
    );
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        client
            .documents()
            .interests(&ListDocumentInterestsArgs {
                document: reopened.id
            })
            .unwrap()
            .interests
            .unwrap(),
        records
    );
}

#[test]
fn invalid_interest_requests_do_not_change_state_or_create_history() {
    let (mut files, id, mut guids) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    for document in [0, u64::MAX] {
        assert!(
            client
                .documents()
                .interests(&ListDocumentInterestsArgs { document })
                .is_err()
        );
        assert!(
            client
                .documents()
                .has_interest(&HasDocumentInterestArgs {
                    document,
                    client: "cam".into()
                })
                .is_err()
        );
        assert!(
            client
                .documents()
                .add_interest(&request(document, "paths", 1))
                .is_err()
        );
        assert!(
            client
                .documents()
                .remove_interest(&RemoveDocumentInterestArgs {
                    document,
                    client_id: "cam".into(),
                    name: "paths".into()
                })
                .is_err()
        );
    }
    assert!(
        client
            .documents()
            .add_interest(&request(id, "", 1))
            .is_err()
    );
    let mut invalid = request(id, "paths", 1);
    invalid.interest.interest_type = DocumentInterestType(1);
    assert!(client.documents().add_interest(&invalid).is_err());
    let missing = client
        .documents()
        .remove_interest(&RemoveDocumentInterestArgs {
            document: id,
            client_id: "cam".into(),
            name: "missing".into(),
        })
        .unwrap_err();
    assert!(missing.to_string().contains("missing"));
    let host = FileHost::new(&mut files, &mut guids, "test");
    assert!(host.call(METHOD_DOCUMENTS_ADD_INTEREST, b"[").is_err());
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .interests
            .records()
            .is_empty()
    );
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
}

#[test]
fn interest_commands_use_explicit_document_and_do_not_change_focus() {
    let (mut files, first, mut guids) = created_files(false);
    let second = files
        .create(
            Path::new("/fixtures/second.opd"),
            DocumentPackage::empty(DOCUMENT_PART, "Second"),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let focus = files.workspace().active_document();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client
        .documents()
        .add_interest(&request(first, "paths", 1))
        .unwrap();
    assert_eq!(
        client
            .documents()
            .interests(&ListDocumentInterestsArgs { document: second })
            .unwrap()
            .interests,
        Some(Vec::new())
    );
    assert_eq!(files.workspace().active_document(), focus);
    let mut withdrawn = request(first, "paths", 1);
    withdrawn.interest.interest_type = INTEREST_NONE;
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    client.documents().add_interest(&withdrawn).unwrap();
    assert!(
        client
            .documents()
            .has_interest(&HasDocumentInterestArgs {
                document: first,
                client: "paths".into()
            })
            .unwrap()
            .has_interest
    );
}

#[test]
fn vetoed_interest_command_leaves_no_record_or_undo_step() {
    let (mut workspace, id) = crate::parameters::parameter_workspace_observer(
        ParameterCollection::new(UnitsOfMeasure::default()),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let client = Client::new(oblikovati_automation::SessionHost::new(&mut workspace));
    let error = client
        .documents()
        .add_interest(&request(id, "paths", 1))
        .unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .interests
            .records()
            .is_empty()
    );
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
}
