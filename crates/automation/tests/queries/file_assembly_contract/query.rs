// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
