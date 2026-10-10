// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn creation_preserves_duplicate_titles_and_activates_each_document_kind() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
        );
        let mut ids = Vec::new();
        for kind in ["part", "assembly", "drawing", "presentation"] {
            let created = client
                .documents()
                .create(&CreateDocumentArgs {
                    type_: format!(" {} ", kind.to_uppercase()),
                    name: "Same: 日本語\n{title}".into(),
                    ..Default::default()
                })
                .unwrap();
            assert_eq!(created.name, "Same: 日本語\n{title}");
            assert_eq!(created.type_, kind);
            assert!(created.dirty && created.active && created.visible);
            assert!(created.sub_type.is_empty());
            ids.push(created.id);
        }
        let list = client.documents().list().unwrap().documents.unwrap();
        assert_eq!(list.iter().map(|entry| entry.id).collect::<Vec<_>>(), ids);
        assert_eq!(list.iter().filter(|entry| entry.active).count(), 1);
        assert!(list.last().unwrap().active);
    }
    assert_eq!(factory.created, 4);
    assert!(files.files().packages.borrow().is_empty());
    for id in files.workspace().documents() {
        assert!(files.path(*id).is_err());
    }
}

#[test]
fn created_part_edits_saves_and_reopens_through_the_typed_client() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 2,
    };
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    let created = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Bracket".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "width".into(),
            expression: "25 mm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(
        client
            .documents()
            .save(&SaveDocumentArgs {
                document: created.id
            })
            .unwrap_err()
            .to_string()
            .contains("Save As")
    );
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: created.id,
            new_full_document_name: "/fixtures/bracket.opd".into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: created.id,
            force: false,
        })
        .unwrap();
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/bracket.opd".into(),
            visible: true,
            defer_content: false,
        })
        .unwrap();
    assert_ne!(reopened.id, created.id);
    assert_eq!(reopened.name, "Bracket");
    assert!(!reopened.dirty);
    assert_eq!(
        client
            .parameters()
            .get(&ParameterNameArgs {
                name: "width".into()
            })
            .unwrap()
            .expression,
        "25 mm"
    );
}

#[test]
fn rejected_creation_leaves_no_owner_or_observer_and_preserves_active_document() {
    let (mut files, original, mut source) = created_files(false);
    let before = source.0;
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 0,
    };
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
        );
        for request in [
            CreateDocumentArgs {
                type_: "part".into(),
                ..Default::default()
            },
            CreateDocumentArgs {
                type_: "invalid".into(),
                name: "Bad".into(),
                ..Default::default()
            },
            CreateDocumentArgs {
                type_: "part".into(),
                name: "Bad".into(),
                sub_type: "sheet-metal".into(),
            },
            CreateDocumentArgs {
                type_: "part".into(),
                name: "Factory refusal".into(),
                ..Default::default()
            },
        ] {
            assert!(client.documents().create(&request).is_err());
        }
        assert_eq!(
            client.documents().list().unwrap().documents.unwrap().len(),
            1
        );
    }
    assert_eq!(factory.created, 0);
    assert_eq!(source.0, before);
    assert_eq!(files.workspace().active_document(), Some(original));
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .documents()
            .create(&CreateDocumentArgs {
                type_: "part".into(),
                name: "No factory".into(),
                ..Default::default()
            })
            .unwrap_err()
            .to_string()
            .contains("factory")
    );
}
