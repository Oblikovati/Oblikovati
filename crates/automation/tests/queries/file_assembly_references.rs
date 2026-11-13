// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

fn save_component_fixture(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    name: &str,
) -> Vec<u8> {
    let path = PathBuf::from(format!("/fixtures/{name}"));
    let id = files
        .create(
            &path,
            DocumentPackage::empty(DOCUMENT_PART, name),
            QueryObserver::default(),
            "test",
            source,
        )
        .unwrap()
        .session_id;
    files.save(id, "test", source).unwrap();
    let bytes = files.files().packages.borrow()[&path].clone();
    files.close_discarding_changes(id).unwrap();
    bytes
}
fn assembly_with_references(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> u64 {
    let id = files
        .create(
            Path::new("/fixtures/assembly.oad"),
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly"),
            QueryObserver::default(),
            "test",
            source,
        )
        .unwrap()
        .session_id;
    files
        .commands(id)
        .unwrap()
        .run("Add references", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(
                ["part.opd", "part.opd", "missing.opd"]
                    .iter()
                    .enumerate()
                    .map(|(index, name)| AssemblyOccurrenceRecipe {
                        name: format!("Part:{index}"),
                        component: (*name).into(),
                        ..Default::default()
                    })
                    .collect(),
            );
            Ok(())
        })
        .unwrap();
    id
}
fn resolve(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    owner: u64,
) -> oblikovati_automation::AssemblyReferenceReport {
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    FileHost::new(files, source, "test")
        .with_observer_factory(&mut factory)
        .resolve_assembly_references(owner)
        .unwrap()
}

#[test]
fn shared_components_resolve_in_background_and_missing_references_recover() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "part.opd");
    let missing = save_component_fixture(&mut files, &mut source, "missing.opd");
    files
        .files()
        .packages
        .borrow_mut()
        .remove(Path::new("/fixtures/missing.opd"));
    let owner = assembly_with_references(&mut files, &mut source);
    let first = resolve(&mut files, &mut source, owner);
    assert_eq!(first.resolved.len(), 1);
    assert_eq!(first.resolved[0].0, "part.opd");
    assert_eq!(first.failures.len(), 1);
    assert_eq!(first.failures[0].reference, "missing.opd");
    let part = first.resolved[0].1;
    assert!(!files.document_info(part).unwrap().visible);
    assert_eq!(files.workspace().active_document(), Some(owner));
    assert!(
        files
            .document_reference_error(owner, "missing.opd")
            .is_some()
    );
    files
        .files()
        .packages
        .borrow_mut()
        .insert(PathBuf::from("/fixtures/missing.opd"), missing);
    let recovered = resolve(&mut files, &mut source, owner);
    assert!(recovered.failures.is_empty());
    assert_eq!(recovered.resolved.len(), 2);
    assert_eq!(recovered.resolved[0].1, part);
    assert_eq!(files.document_reference_error(owner, "missing.opd"), None);
    files
        .commands(owner)
        .unwrap()
        .run("Remove part", |snapshot, _| -> Result<(), ()> {
            snapshot
                .assembly
                .as_mut()
                .unwrap()
                .retain(|record| record.component != "part.opd");
            Ok(())
        })
        .unwrap();
    let remaining = resolve(&mut files, &mut source, owner);
    assert_eq!(remaining.resolved.len(), 1);
    assert!(!files.workspace().documents().contains(&part));
    files.close_discarding_changes(owner).unwrap();
    assert!(files.workspace().documents().is_empty());
}

#[test]
fn wrong_component_kind_is_reported_and_new_background_document_is_closed() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let path = Path::new("/fixtures/drawing.odd");
    let drawing = files
        .create(
            path,
            DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    files.save(drawing, "test", &mut source).unwrap();
    files.close_discarding_changes(drawing).unwrap();
    let owner = assembly_with_references(&mut files, &mut source);
    files
        .commands(owner)
        .unwrap()
        .run("Reference drawing", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly = Some(vec![AssemblyOccurrenceRecipe {
                name: "Wrong:1".into(),
                component: "drawing.odd".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    let report = resolve(&mut files, &mut source, owner);
    assert!(report.resolved.is_empty());
    assert_eq!(report.failures.len(), 1);
    assert!(
        report.failures[0]
            .cause
            .contains("expected a part or assembly")
    );
    assert_eq!(files.workspace().documents(), &[owner]);
    assert_eq!(files.workspace().active_document(), Some(owner));
    assert!(
        files
            .document_reference_error(owner, "drawing.odd")
            .is_some()
    );
}

#[test]
fn opening_an_assembly_resolves_nested_components_without_activating_them() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    save_component_fixture(&mut files, &mut source, "part.opd");
    for (name, bytes) in [
        ("nested.oad", &b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}]}\n"[..]),
        ("root.oad", &b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Nested:1', component: nested.oad}, {name: 'Missing:1', component: missing.opd}]}\n"[..]),
    ] {
        files.files().packages.borrow_mut().insert(PathBuf::from(format!("/fixtures/{name}")), bytes.to_vec());
    }
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/root.oad"), true)
        .unwrap();
    let nested = files.document_reference(root, "nested.oad").unwrap();
    let part = files.document_reference(nested, "part.opd").unwrap();
    assert!(!files.document_info(nested).unwrap().visible);
    assert!(!files.document_info(part).unwrap().visible);
    assert_eq!(files.workspace().active_document(), Some(root));
    assert!(
        files
            .document_reference_error(root, "missing.opd")
            .is_some()
    );
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "No change".into(),
        })
        .unwrap();
    client.transactions().end().unwrap();
    assert_eq!(files.document_reference(root, "nested.oad"), Some(nested));
    assert_eq!(files.document_reference(nested, "part.opd"), Some(part));
    files.save(root, "test", &mut source).unwrap();
    files.close_discarding_changes(root).unwrap();
    assert!(files.workspace().documents().is_empty());
    let reopened = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(Path::new("/fixtures/root.oad"), true)
        .unwrap();
    let nested = files.document_reference(reopened, "nested.oad").unwrap();
    assert!(files.document_reference(nested, "part.opd").is_some());
    assert!(
        files
            .document_reference_error(reopened, "missing.opd")
            .is_some()
    );
    files.close_discarding_changes(reopened).unwrap();
}
