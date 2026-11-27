// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn removal_keeps_shared_and_missing_occurrences_then_releases_and_restores_the_last_component() {
    let (mut files, mut guids, root, part) = assembly_fixture();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    assert_eq!(remove(&mut files, &mut guids, 1), vec![2, 3]);
    assert_eq!(files.document_reference(root, "part.opd"), Some(part));
    assert_eq!(remove(&mut files, &mut guids, 3), vec![2]);
    assert!(remove(&mut files, &mut guids, 2).is_empty());
    assert_eq!(files.workspace().documents(), &[root]);
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory),
        );
        assert!(
            client
                .assembly()
                .remove(&RemoveOccurrenceArgs { id: 2 })
                .is_err()
        );
        client.transactions().undo().unwrap();
        let restored = client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap();
        assert_eq!(
            restored.iter().map(|row| row.id).collect::<Vec<_>>(),
            vec![2]
        );
        client.transactions().redo().unwrap();
        assert_eq!(
            client
                .assembly()
                .occurrences(&Default::default())
                .unwrap()
                .occurrences,
            Some(vec![])
        );
    }
    assert_eq!(factory.created, 1);
    assert_eq!(files.workspace().documents(), &[root]);

    files.save(root, "test", &mut guids).unwrap();
    files.close_discarding_changes(root).unwrap();
    let host = FileHost::new(&mut files, &mut guids, "test").with_observer_factory(&mut factory);
    host.open_path(Path::new("/fixtures/assembly.oad"), true)
        .unwrap();
    assert_eq!(
        Client::new(host)
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences,
        Some(vec![])
    );
}

#[test]
fn finishing_an_inner_transaction_keeps_the_component_for_outer_abort() {
    let (mut files, mut guids, root, part) = assembly_fixture();
    {
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        let transaction = TransactionBeginArgs {
            label: "Remove placements".into(),
        };
        client.transactions().begin(&transaction).unwrap();
        client.transactions().begin(&transaction).unwrap();
        for id in [1, 2, 3] {
            client
                .assembly()
                .remove(&RemoveOccurrenceArgs { id })
                .unwrap();
        }
        client.transactions().end().unwrap();
        client.transactions().abort().unwrap();
    }
    assert_eq!(files.document_reference(root, "part.opd"), Some(part));
    assert_eq!(
        files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()
            .len(),
        3
    );
}

fn assembly_fixture() -> (DiscoveryWorkspace, QueryGuids, u64, u64) {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    save_component_fixture(&mut files, &mut guids, "part.opd");
    let root = assembly_with_references(&mut files, &mut guids);
    resolve(&mut files, &mut guids, root);
    let part = files.document_reference(root, "part.opd").unwrap();
    (files, guids, root, part)
}

fn remove(files: &mut DiscoveryWorkspace, guids: &mut QueryGuids, id: u64) -> Vec<u64> {
    Client::new(FileHost::new(files, guids, "test"))
        .assembly()
        .remove(&RemoveOccurrenceArgs { id })
        .unwrap()
        .occurrences
        .unwrap()
        .into_iter()
        .map(|row| row.id)
        .collect()
}
