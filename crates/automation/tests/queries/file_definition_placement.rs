// SPDX-License-Identifier: GPL-2.0-only
use super::assembly_references::save_component_fixture;
use super::*;
use oblikovati_api::types::Matrix;

#[test]
fn definition_placement_batch_is_ordered_atomic_and_one_undo_step() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    let (root, part) = open_assembly_and_part(&mut files, &mut guids);
    let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
    let source = client
        .assembly()
        .place(&PlaceOccurrenceArgs {
            document: part,
            name: "Box:1".into(),
            transform: translation(0.),
        })
        .unwrap()
        .occurrence;
    let invalid = BatchPlacement {
        name: "Invalid".into(),
        transform: Matrix::default(),
    };
    let mut args = PlaceByDefinitionBatchArgs {
        source: source.id,
        placements: Some(vec![placement("Box:2", 3.), invalid]),
    };
    assert!(client.assembly().place_by_definition_batch(&args).is_err());
    assert_eq!(rows(&client), vec![source.clone()]);
    args.placements = Some(vec![]);
    assert_eq!(
        client
            .assembly()
            .place_by_definition_batch(&args)
            .unwrap()
            .occurrences,
        Some(vec![])
    );
    args.placements = Some(vec![placement("Box:2", 3.), placement("Box:3", 6.)]);
    let batch = client
        .assembly()
        .place_by_definition_batch(&args)
        .unwrap()
        .occurrences
        .unwrap();
    assert_eq!(
        batch.iter().map(|row| row.id).collect::<Vec<_>>(),
        vec![source.id + 1, source.id + 2]
    );
    assert_eq!(
        batch
            .iter()
            .map(|row| (row.name.as_str(), row.transform.cells[3], row.grounded))
            .collect::<Vec<_>>(),
        vec![("Box:2", 3., false), ("Box:3", 6., false)]
    );
    client.transactions().undo().unwrap();
    assert_eq!(rows(&client), vec![source.clone()]);
    client.transactions().redo().unwrap();
    assert_eq!(rows(&client)[1..], batch);

    let copy = client
        .assembly()
        .place_by_definition(&PlaceByDefinitionArgs {
            source: batch[0].id,
            name: "Box:4".into(),
            transform: translation(9.),
        })
        .unwrap()
        .occurrence;
    assert_eq!(rows(&client)[3], copy);
    assert!(
        client
            .assembly()
            .place_by_definition(&PlaceByDefinitionArgs {
                source: u64::MAX,
                name: "Missing".into(),
                transform: translation(12.),
            })
            .is_err()
    );
    assert_eq!(rows(&client).len(), 4);
    drop(client);

    let records = files
        .workspace()
        .session(root)
        .unwrap()
        .snapshot()
        .assembly
        .as_ref()
        .unwrap();
    let reference = records[0].component_reference().unwrap();
    assert!(
        records
            .iter()
            .all(|row| row.component_reference() == Some(reference))
    );
    assert_eq!(files.document_reference(root, reference), Some(part));
    assert_eq!(files.workspace().documents(), &[part, root]);
}

fn rows(client: &Client<impl oblikovati_api::client::Caller>) -> Vec<OccurrenceInfo> {
    client
        .assembly()
        .occurrences(&Default::default())
        .unwrap()
        .occurrences
        .unwrap()
}

fn translation(x: f64) -> Matrix {
    Matrix {
        cells: [
            1., 0., 0., x, 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
        ],
    }
}

fn placement(name: &str, x: f64) -> BatchPlacement {
    BatchPlacement {
        name: name.into(),
        transform: translation(x),
    }
}

fn open_assembly_and_part(files: &mut DiscoveryWorkspace, guids: &mut QueryGuids) -> (u64, u64) {
    save_component_fixture(files, guids, "definition.opd");
    let assembly = Path::new("/fixtures/definition.oad");
    files.files().packages.borrow_mut().insert(
        assembly.into(),
        b"schemaVersion: 2\ndocumentType: 2\n".to_vec(),
    );
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    let part = FileHost::new(files, guids, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/definition.opd"), true)
        .unwrap();
    let root = FileHost::new(files, guids, "test")
        .with_observer_factory(&mut factory)
        .open_path(assembly, true)
        .unwrap();
    (root, part)
}

#[test]
fn occurrence_transform_and_grounding_round_trip_through_history_and_file() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    let (root, part) = open_assembly_and_part(&mut files, &mut guids);
    let moved = {
        let client = Client::new(FileHost::new(&mut files, &mut guids, "test"));
        let original = client
            .assembly()
            .place(&PlaceOccurrenceArgs {
                document: part,
                name: "Box:1".into(),
                transform: translation(0.),
            })
            .unwrap()
            .occurrence;
        let id = original.id;
        let ground = GroundOccurrenceArgs {
            id,
            grounded: false,
        };
        let released = client.assembly().ground(&ground).unwrap().occurrence;
        assert!(!released.grounded);
        assert_eq!(released.transform, original.transform);
        let transform = TransformOccurrenceArgs {
            id,
            transform: translation(6.),
        };
        let moved = client.assembly().transform(&transform).unwrap().occurrence;
        assert_eq!(moved.id, id);
        assert_eq!(moved.transform, transform.transform);
        assert!(!moved.grounded);
        assert_eq!(
            client.assembly().transform(&transform).unwrap().occurrence,
            moved
        );
        assert!(
            client
                .assembly()
                .transform(&TransformOccurrenceArgs {
                    id,
                    transform: Matrix::default(),
                })
                .is_err()
        );
        assert!(
            client
                .assembly()
                .ground(&GroundOccurrenceArgs {
                    id: u64::MAX,
                    grounded: true,
                })
                .is_err()
        );
        client.transactions().undo().unwrap();
        assert_eq!(rows(&client), vec![released]);
        client.transactions().undo().unwrap();
        assert_eq!(rows(&client), vec![original]);
        client.transactions().redo().unwrap();
        client.transactions().redo().unwrap();
        assert_eq!(rows(&client), vec![moved.clone()]);
        moved
    };
    files.save(root, "test", &mut guids).unwrap();
    files.close_discarding_changes(root).unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 4,
    };
    FileHost::new(&mut files, &mut guids, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/definition.oad"), true)
        .unwrap();
    assert_eq!(
        rows(&Client::new(FileHost::new(&mut files, &mut guids, "test"))),
        vec![moved]
    );
}
