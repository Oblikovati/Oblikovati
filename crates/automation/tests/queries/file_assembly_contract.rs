// SPDX-License-Identifier: GPL-2.0-only
use super::assembly_references::save_component_fixture;
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

#[test]
fn canonical_assembly_api_keeps_occurrence_ids_through_edits_and_history() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let path = Path::new("/fixtures/api-assembly.oad");
    files.files().packages.borrow_mut().insert(path.into(), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: Same, virtual: true, virtualPartNumber: A}, {name: Same, virtual: true, virtualPartNumber: B}]}\n".to_vec());
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    let readonly = Client::new(FileQueryHost::new(&files))
        .assembly()
        .occurrences(&Default::default())
        .unwrap();
    assert_eq!(readonly.occurrences.unwrap().len(), 2);
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        let rows = client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap();
        assert_eq!((rows[0].id, rows[1].id), (1, 2));
        let result = client
            .assembly()
            .suppress(&SuppressOccurrenceArgs {
                id: 2,
                suppressed: true,
            })
            .unwrap();
        assert!(result.occurrence.suppressed);
        assert_eq!(result.occurrence.part_number, "B");
        client.transactions().undo().unwrap();
        let undone = client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap();
        assert_eq!(undone[1].id, 2);
        assert!(!undone[1].suppressed);
        client.transactions().redo().unwrap();
        assert!(
            client
                .assembly()
                .occurrences(&Default::default())
                .unwrap()
                .occurrences
                .unwrap()[1]
                .suppressed
        );
    }
    files
        .commands(root)
        .unwrap()
        .run("Reorder", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly.as_mut().unwrap().swap(0, 1);
            Ok(())
        })
        .unwrap();
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        let rows = client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap();
        assert_eq!((rows[0].id, rows[0].part_number.as_str()), (2, "B"));
        let restored = client
            .assembly()
            .suppress(&SuppressOccurrenceArgs {
                id: 2,
                suppressed: false,
            })
            .unwrap();
        assert_eq!(restored.occurrence.part_number, "B");
        assert!(!restored.occurrence.suppressed);
        assert!(
            client
                .assembly()
                .suppress(&SuppressOccurrenceArgs {
                    id: 99,
                    suppressed: true
                })
                .is_err()
        );
    }
    files.save(root, "test", &mut source).unwrap();
    let bytes = files.files().packages.borrow()[path].clone();
    assert!(!String::from_utf8(bytes).unwrap().contains("session_id"));
    files.close_discarding_changes(root).unwrap();
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    let rows = Client::new(FileQueryHost::new(&files))
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap();
    assert_eq!((rows[0].id, rows[0].part_number.as_str()), (1, "B"));
    assert_eq!((rows[1].id, rows[1].part_number.as_str()), (2, "A"));
    files
        .commands(root)
        .unwrap()
        .run("Replace occurrence", |snapshot, ids| -> Result<(), ()> {
            let rows = snapshot.assembly.as_mut().unwrap();
            rows.retain(|row| row.session_id != 2);
            rows.push(AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "New".into(),
                virtual_: true,
                ..Default::default()
            });
            Ok(())
        })
        .unwrap();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let current = client
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap();
    assert_eq!((current[1].id, current[1].name.as_str()), (3, "New"));
    assert!(
        client
            .assembly()
            .suppress(&SuppressOccurrenceArgs {
                id: 2,
                suppressed: true
            })
            .is_err()
    );
    assert!(
        client
            .assembly()
            .suppress(&SuppressOccurrenceArgs {
                id: 1,
                suppressed: true
            })
            .is_ok()
    );
    drop(client);
    assert!(
        !files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()
            .last()
            .unwrap()
            .suppressed
    );
}

#[test]
fn canonical_occurrence_query_returns_the_resolved_nested_tree() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "part.opd");
    for (name, bytes) in [
        ("nested.oad", &b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Child:1', component: part.opd}]}\n"[..]),
        ("root.oad", &b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Nested:1', component: nested.oad}]}\n"[..]),
    ] {
        files.files().packages.borrow_mut().insert(PathBuf::from(format!("/fixtures/{name}")), bytes.to_vec());
    }
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/root.oad"), true)
        .unwrap();
    let result = Client::new(FileQueryHost::new(&files))
        .assembly()
        .occurrences(&Default::default())
        .unwrap();
    let root = result.occurrences.unwrap();
    assert_eq!((root[0].id, root[0].name.as_str()), (1, "Nested:1"));
    let children = root[0].children.as_ref().unwrap();
    assert_eq!((children[0].id, children[0].name.as_str()), (1, "Child:1"));
}

#[test]
fn canonical_occurrence_api_refuses_duplicate_session_ids() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let path = Path::new("/fixtures/duplicate-ids.oad");
    files.files().packages.borrow_mut().insert(path.into(), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: A, virtual: true}, {name: B, virtual: true}]}\n".to_vec());
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 2,
    };
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    let result =
        files
            .commands(root)
            .unwrap()
            .run("Invalid identity", |snapshot, _| -> Result<(), ()> {
                snapshot.assembly.as_mut().unwrap()[1].session_id = 1;
                Ok(())
            });
    assert!(result.is_err());
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert_eq!(
        client
            .assembly()
            .occurrences(&Default::default())
            .unwrap()
            .occurrences
            .unwrap()
            .len(),
        2
    );
}
