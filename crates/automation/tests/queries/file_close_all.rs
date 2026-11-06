// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn forced_close_all_clears_owners_and_allows_recreation() {
    let (mut files, first, mut source) = created_files(false);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 3,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    let request = CreateDocumentArgs {
        type_: "part".into(),
        name: "part".into(),
        ..Default::default()
    };
    let second = client.documents().create(&request).unwrap();
    client.documents().create(&request).unwrap();
    assert_eq!(
        client
            .documents()
            .close_all(&CloseAllDocumentsArgs { force: true })
            .unwrap()
            .closed,
        3
    );
    assert_eq!(client.documents().list().unwrap().documents, Some(vec![]));
    assert_eq!(
        client
            .documents()
            .close_all(&CloseAllDocumentsArgs { force: false })
            .unwrap()
            .closed,
        0
    );
    let recreated = client.documents().create(&request).unwrap();
    assert!(recreated.id > second.id && second.id > first);
    assert!(recreated.active && recreated.dirty);
}

#[test]
fn close_all_saves_bound_documents_before_releasing_them() {
    let (mut files, first, mut source) = created_files(false);
    let second = files
        .create(
            Path::new("/fixtures/second.opd"),
            DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Second"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let response = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap();
    assert_eq!(response.closed, 2);
    assert!(files.workspace().documents().is_empty());
    assert_eq!(files.workspace().active_document(), None);
    assert!(files.path(first).is_err() && files.path(second).is_err());
    let packages = files.files().packages.borrow();
    for path in ["/fixtures/part.opd", "/fixtures/second.opd"] {
        let package = DocumentPackage::decode(&packages[Path::new(path)]).unwrap();
        assert!(package.identity_record().unwrap().is_some());
    }
}

#[test]
fn close_all_stops_on_first_error_and_retains_later_documents() {
    let (mut files, first, mut source) = created_files(false);
    let second = files
        .create_unbound(
            DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Unbound"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let third = files
        .create(
            Path::new("/fixtures/later.opd"),
            DocumentPackage::empty(oblikovati_api::types::DOCUMENT_PART, "Later"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap_err()
        .to_string();
    assert!(error.contains(&format!("closing {second}")) && error.contains("Save As"));
    assert!(files.path(first).is_err());
    assert_eq!(files.workspace().documents(), [second, third]);
    assert_eq!(files.workspace().active_document(), Some(third));
    assert!(
        files
            .files()
            .packages
            .borrow()
            .contains_key(Path::new("/fixtures/part.opd"))
    );
    assert!(
        !files
            .files()
            .packages
            .borrow()
            .contains_key(Path::new("/fixtures/later.opd"))
    );
    assert_eq!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .documents()
            .close_all(&CloseAllDocumentsArgs { force: true })
            .unwrap()
            .closed,
        2
    );
}
