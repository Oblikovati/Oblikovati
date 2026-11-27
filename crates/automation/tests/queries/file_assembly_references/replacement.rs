// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn replacement_preserves_occurrence_state_and_restores_shared_references_through_history() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    save_component_fixture(&mut files, &mut guids, "part.opd");
    let replacement = open_replacement_part(&mut files, &mut guids);
    let root = assembly_with_references(&mut files, &mut guids);
    resolve(&mut files, &mut guids, root);
    let old_part = files.document_reference(root, "part.opd").unwrap();
    let swap = ReplaceOccurrenceArgs {
        id: 1,
        document: replacement,
    };
    let replacement_path = files
        .path(replacement)
        .unwrap()
        .to_str()
        .unwrap()
        .to_owned();

    {
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        client
            .assembly()
            .remove(&RemoveOccurrenceArgs { id: 3 })
            .unwrap();
        client
            .assembly()
            .ground(&GroundOccurrenceArgs {
                id: 1,
                grounded: true,
            })
            .unwrap();
        client
            .assembly()
            .suppress(&SuppressOccurrenceArgs {
                id: 1,
                suppressed: true,
            })
            .unwrap();
        let before = client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap();
        assert!(
            client
                .assembly()
                .replace(&ReplaceOccurrenceArgs {
                    id: 1,
                    document: root
                })
                .is_err()
        );
        assert_eq!(
            client.assembly().replace(&swap).unwrap().occurrence,
            before[0]
        );
    }
    assert_eq!(files.document_reference(root, "part.opd"), Some(old_part));
    assert_eq!(
        files.document_reference(root, &replacement_path),
        Some(replacement)
    );
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 12,
    };
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
        );
        let last = ReplaceOccurrenceArgs {
            id: 2,
            document: replacement,
        };
        let result = client.assembly().replace(&last).unwrap().occurrence;
        assert_eq!(client.assembly().replace(&last).unwrap().occurrence, result);
    }
    assert!(!files.workspace().documents().contains(&old_part));
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
        );
        client.transactions().undo().unwrap();
    }
    assert!(files.document_reference(root, "part.opd").is_some());
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
        );
        client.transactions().redo().unwrap();
    }
    assert_eq!(files.document_reference(root, "part.opd"), None);
    files.save(root, "test", &mut guids).unwrap();
    files.close_discarding_changes(root).unwrap();
    let root = FileHost::new(&mut files, &mut guids, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/assembly.oad"), true)
        .unwrap();
    let rows = files
        .workspace()
        .session(root)
        .unwrap()
        .snapshot()
        .assembly
        .as_ref()
        .unwrap();
    assert_eq!(rows.len(), 2);
    assert!(
        rows.iter()
            .all(|row| row.component_reference() == Some(&replacement_path))
    );
    assert_eq!(
        files.document_reference(root, &replacement_path),
        Some(replacement)
    );
}

#[test]
fn replacement_veto_discards_the_proposed_binding() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    let replacement = open_replacement_part(&mut files, &mut guids);
    let path = Path::new("/fixtures/replace-veto.oad");
    files.files().packages.borrow_mut().insert(
        path.into(),
        b"schemaVersion: 2
documentType: 2
model: {occurrences: [{name: 'Virtual', virtual: true}]}
"
        .to_vec(),
    );
    let root = files
        .open(
            path,
            QueryObserver {
                veto: true,
                ..Default::default()
            },
            "test",
            &mut guids,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    assert!(
        client
            .assembly()
            .replace(&ReplaceOccurrenceArgs {
                id: 1,
                document: replacement
            })
            .is_err()
    );
    assert!(
        client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap()[0]
            .virtual_
    );
    files.close_discarding_changes(replacement).unwrap();
    assert_eq!(files.workspace().active_document(), Some(root));
}

fn open_replacement_part(files: &mut DiscoveryWorkspace, guids: &mut QueryGuids) -> u64 {
    save_component_fixture(files, guids, "replacement.opd");
    files
        .open(
            Path::new("/fixtures/replacement.opd"),
            QueryObserver::default(),
            "test",
            guids,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id()
}
