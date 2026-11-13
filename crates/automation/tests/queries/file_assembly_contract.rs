// SPDX-License-Identifier: GPL-2.0-only
use super::assembly_references::save_component_fixture;
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

#[test]
fn file_placement_keeps_focus_and_releases_refused_background_documents() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "background.opd");
    let assembly = Path::new("/fixtures/background.oad");
    files.files().packages.borrow_mut().insert(
        assembly.into(),
        b"schemaVersion: 2\ndocumentType: 2\n".to_vec(),
    );
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(assembly, true)
        .unwrap();
    let drawing = Path::new("/fixtures/not-a-component.odd");
    files.files().packages.borrow_mut().insert(
        drawing.into(),
        b"schemaVersion: 2\ndocumentType: 3\n".to_vec(),
    );
    assert!(
        FileHost::new(&mut files, &mut source, "test")
            .with_observer_factory(&mut factory)
            .open_component_preview(drawing)
            .is_err()
    );
    assert_eq!(files.workspace().documents(), &[root]);
    let matrix = oblikovati_api::types::Matrix {
        cells: [
            1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
    };
    let mut invalid = matrix;
    invalid.cells[0] = 0.;
    let path = Path::new("/fixtures/background.opd");
    assert!(
        FileHost::new(&mut files, &mut source, "test")
            .with_observer_factory(&mut factory)
            .place_component_file(path, "Part:1", invalid)
            .is_err()
    );
    assert_eq!(files.workspace().documents(), &[root]);
    assert_eq!(files.workspace().active_document(), Some(root));
    FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .place_component_file(path, "Part:1", matrix)
        .unwrap();
    let target = files
        .document_reference(root, path.to_str().unwrap())
        .unwrap();
    assert!(!files.document_info(target).unwrap().visible);
    assert_eq!(files.workspace().active_document(), Some(root));
    files.close_discarding_changes(root).unwrap();
    assert!(!files.workspace().documents().contains(&target));
}

#[test]
fn placement_respects_disabled_first_component_grounding() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "free-part.opd");
    let assembly = Path::new("/fixtures/free-assembly.oad");
    files.files().packages.borrow_mut().insert(assembly.into(), b"schemaVersion: 2\ndocumentType: 2\nmodel: {options: {placeAndGroundFirstComponentAtOrigin: false}}\n".to_vec());
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    let part = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/free-part.opd"), true)
        .unwrap();
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(assembly, true)
        .unwrap();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let placed = client
        .assembly()
        .place(&PlaceOccurrenceArgs {
            document: part,
            name: "Part:1".into(),
            transform: oblikovati_api::types::Matrix {
                cells: [
                    1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
                ],
            },
        })
        .unwrap();
    assert!(!placed.occurrence.grounded);
    drop(client);
    files.save(root, "test", &mut source).unwrap();
    let bytes = files.files().packages.borrow()[assembly].clone();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&bytes).unwrap();
    assert!(!package.ground_first_component().unwrap());
    assert!(!package.assembly_occurrences().unwrap().unwrap()[0].grounded);
}

#[test]
fn redo_reopens_a_component_closed_after_undo() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "redo-part.opd");
    let path = Path::new("/fixtures/redo-root.oad");
    files
        .files()
        .packages
        .borrow_mut()
        .insert(path.into(), b"schemaVersion: 2\ndocumentType: 2\n".to_vec());
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    let part = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/redo-part.opd"), true)
        .unwrap();
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    let matrix = oblikovati_api::types::Matrix {
        cells: [
            1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
    };
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        client
            .assembly()
            .place(&PlaceOccurrenceArgs {
                document: part,
                name: "Part:1".into(),
                transform: matrix,
            })
            .unwrap();
        client.transactions().undo().unwrap();
    }
    files.close_discarding_changes(part).unwrap();
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    client.transactions().redo().unwrap();
    let rows = client
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap();
    assert_eq!((rows.len(), rows[0].name.as_str()), (1, "Part:1"));
    drop(client);
    assert_ne!(
        files.document_reference(root, "/fixtures/redo-part.opd"),
        Some(part)
    );
    assert!(
        files
            .document_reference(root, "/fixtures/redo-part.opd")
            .is_some()
    );
    let reopened = files
        .document_reference(root, "/fixtures/redo-part.opd")
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .transactions()
        .undo()
        .unwrap();
    if files.workspace().documents().contains(&reopened) {
        files.close_discarding_changes(reopened).unwrap();
    }
    files
        .files()
        .packages
        .borrow_mut()
        .remove(Path::new("/fixtures/redo-part.opd"));
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    client.transactions().redo().unwrap();
    let error = client
        .assembly()
        .occurrences(&Default::default())
        .unwrap_err();
    assert!(error.to_string().contains("redo-part.opd"));
    assert!(error.to_string().contains("reference"));
}

#[test]
fn canonical_placement_survives_history_save_and_reopen() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "placed.opd");
    let assembly_path = Path::new("/fixtures/placed.oad");
    files.files().packages.borrow_mut().insert(
        assembly_path.into(),
        b"schemaVersion: 2\ndocumentType: 2\n".to_vec(),
    );
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 6,
    };
    let part = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/placed.opd"), true)
        .unwrap();
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(assembly_path, true)
        .unwrap();
    let identity = oblikovati_api::types::Matrix {
        cells: [
            1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
    };
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        let args = PlaceOccurrenceArgs {
            document: part,
            name: "Part:1".into(),
            transform: identity,
        };
        let placed = client.assembly().place(&args).unwrap().occurrence;
        assert_eq!((placed.id, placed.name.as_str()), (1, "Part:1"));
        assert!(placed.grounded);
        let duplicate = client.assembly().place(&args).unwrap().occurrence;
        assert_ne!(duplicate.id, placed.id);
        assert!(!duplicate.grounded);
        client.transactions().undo().unwrap();
        let mut invalid = args.clone();
        invalid.name = "Part:2".into();
        invalid.transform.cells[0] = 0.;
        assert!(client.assembly().place(&invalid).is_err());
        client.transactions().undo().unwrap();
        assert!(
            client
                .assembly()
                .occurrences(&Default::default())
                .unwrap()
                .occurrences
                .unwrap()
                .is_empty()
        );
        client.transactions().redo().unwrap();
        assert_eq!(
            client
                .assembly()
                .occurrences(&Default::default())
                .unwrap()
                .occurrences
                .unwrap()[0]
                .id,
            placed.id
        );
    }
    files.save(root, "test", &mut source).unwrap();
    files.close_discarding_changes(root).unwrap();
    FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(assembly_path, true)
        .unwrap();
    let rows = Client::new(FileQueryHost::new(&files))
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap();
    assert_eq!(
        (rows.len(), rows[0].name.as_str(), &rows[0].transform),
        (1, "Part:1", &identity)
    );
}

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
