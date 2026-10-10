// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_automation::DocumentObserverFactory;

struct QueryObserverFactory {
    created: usize,
    limit: usize,
}
impl DocumentObserverFactory for QueryObserverFactory {
    type Observer = QueryObserver;
    fn create(&mut self) -> Result<QueryObserver, String> {
        if self.created == self.limit {
            return Err("observer limit reached".into());
        }
        self.created += 1;
        Ok(QueryObserver::default())
    }
}

#[test]
fn open_reuses_unsaved_state_and_constructs_observers_only_for_new_owners() {
    let (mut files, original, mut source) = created_files(false);
    files.save(original, "test", &mut source).unwrap();
    files.close_discarding_changes(original).unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 2,
    };
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
        );
        let opened = client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/fixtures/part.opd".into(),
                visible: true,
                defer_content: false,
            })
            .unwrap();
        assert_ne!(opened.id, original);
        assert_eq!(opened.type_, "part");
        assert_eq!(opened.name, "part");
        assert!(opened.active && opened.visible && !opened.dirty);
        client
            .parameters()
            .set(&ParameterSetArgs {
                name: "width".into(),
                expression: "2 cm".into(),
                ..Default::default()
            })
            .unwrap();
        let cached = client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/fixtures/part.opd".into(),
                ..Default::default()
            })
            .unwrap();
        assert_eq!(cached.id, opened.id);
        assert!(cached.active && cached.dirty && !cached.visible);
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "2 cm"
        );
        client
            .documents()
            .close(&CloseDocumentArgs {
                id: cached.id,
                force: false,
            })
            .unwrap();
        let reopened = client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: "/fixtures/part.opd".into(),
                visible: true,
                defer_content: false,
            })
            .unwrap();
        assert_ne!(reopened.id, cached.id);
        assert!(!reopened.dirty);
        assert_eq!(
            client
                .parameters()
                .get(&ParameterNameArgs {
                    name: "width".into()
                })
                .unwrap()
                .expression,
            "2 cm"
        );
    }
    assert_eq!(factory.created, 2);
}

#[test]
fn cached_open_needs_no_factory_and_visibility_is_not_a_recipe_edit() {
    let (mut files, id, mut source) = created_files(false);
    files.save(id, "test", &mut source).unwrap();
    files.files().packages.borrow_mut().clear();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/fixtures/part.opd".into(),
            visible: false,
            defer_content: false,
        })
        .unwrap();
    assert_eq!(result.id, id);
    assert!(!result.visible && !result.dirty);
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    assert!(files.set_visible(999, false).is_err());
}

#[test]
fn bad_open_options_factory_and_package_failures_leave_no_owner() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 0,
    };
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
        );
        for request in [
            OpenDocumentArgs::default(),
            OpenDocumentArgs {
                full_document_name: "/fixtures/part.opd".into(),
                defer_content: true,
                ..Default::default()
            },
            OpenDocumentArgs {
                full_document_name: "/fixtures/part.opd".into(),
                ..Default::default()
            },
        ] {
            assert!(client.documents().open(&request).is_err());
        }
        assert_eq!(client.documents().list().unwrap().documents, Some(vec![]));
    }
    assert_eq!(factory.created, 0);
    assert_eq!(source.0, 0);
    let request = OpenDocumentArgs {
        full_document_name: "/fixtures/bad.opd".into(),
        ..Default::default()
    };
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .documents()
            .open(&request)
            .unwrap_err()
            .to_string()
            .contains("factory")
    );
    factory.limit = 2;
    files
        .files()
        .packages
        .borrow_mut()
        .insert("/fixtures/bad.opd".into(), b"not a document".to_vec());
    assert!(
        Client::new(
            FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory)
        )
        .documents()
        .open(&request)
        .is_err()
    );
    assert!(files.workspace().documents().is_empty());
    assert_eq!(files.workspace().active_document(), None);
    files.files().packages.borrow_mut().insert(
        "/fixtures/bad.opd".into(),
        b"schemaVersion: 2\ndocumentType: 1\n".to_vec(),
    );
    let opened = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    )
    .documents()
    .open(&request)
    .unwrap();
    assert_eq!(opened.name, "bad");
    assert_eq!(files.workspace().documents(), [opened.id]);
}
