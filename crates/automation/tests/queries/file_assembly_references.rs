// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

pub(super) fn save_component_fixture(
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
        .run("Add references", |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(
                ["part.opd", "part.opd", "missing.opd"]
                    .iter()
                    .enumerate()
                    .map(|(index, name)| AssemblyOccurrenceRecipe {
                        session_id: ids.allocate_occurrence().unwrap(),
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
        .run("Reference drawing", |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
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

#[test]
fn occurrence_suppression_is_undoable_saved_and_rejects_stale_paths() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let path = Path::new("/fixtures/edit.oad");
    files.files().packages.borrow_mut().insert(path.into(), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}]}\n".to_vec());
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 8,
    };
    let root = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    let error = Client::new(FileQueryHost::new(&files))
        .assembly()
        .occurrences(&Default::default())
        .unwrap_err();
    assert!(error.to_string().contains("part.opd"), "{error}");
    let rejected = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .assembly()
        .suppress(&SuppressOccurrenceArgs {
            id: 1,
            suppressed: true,
        });
    assert!(rejected.is_err());
    assert!(
        !files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
    let generation = files.workspace().session(root).unwrap().generation();
    let selection = [(root, generation, 0)];
    FileHost::new(&mut files, &mut source, "test")
        .set_occurrence_suppressed(root, &selection, false)
        .unwrap();
    assert_eq!(
        files.workspace().session(root).unwrap().generation(),
        generation
    );
    FileHost::new(&mut files, &mut source, "test")
        .set_occurrence_suppressed(root, &selection, true)
        .unwrap();
    assert!(
        files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
    assert!(
        FileHost::new(&mut files, &mut source, "test")
            .set_occurrence_suppressed(root, &selection, false)
            .is_err()
    );
    files.commands(root).unwrap().undo().unwrap();
    assert!(
        !files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
    files.commands(root).unwrap().redo().unwrap();
    assert!(
        files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
    files.save(root, "test", &mut source).unwrap();
    files.close_discarding_changes(root).unwrap();
    let reopened = FileHost::new(&mut files, &mut source, "test")
        .with_observer_factory(&mut factory)
        .open_path(path, true)
        .unwrap();
    assert!(
        files
            .workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
}

#[test]
fn occurrence_edit_rejects_a_path_through_an_unrelated_owner() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let first = assembly_with_references(&mut files, &mut source);
    let other = files
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Other"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    files
        .commands(other)
        .unwrap()
        .run("Add virtual", |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "Virtual".into(),
                virtual_: true,
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    let first_generation = files.workspace().session(first).unwrap().generation();
    let other_generation = files.workspace().session(other).unwrap().generation();
    let path = [(first, first_generation, 0), (other, other_generation, 0)];
    let error = FileHost::new(&mut files, &mut source, "test")
        .set_occurrence_suppressed(first, &path, true)
        .unwrap_err();
    assert!(
        error.to_string().contains("top-level occurrence"),
        "{error}"
    );
    assert!(
        !files
            .workspace()
            .session(other)
            .unwrap()
            .snapshot()
            .assembly
            .as_ref()
            .unwrap()[0]
            .suppressed
    );
}

#[test]
fn occurrence_suppression_honors_a_transaction_veto() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let path = Path::new("/fixtures/veto.oad");
    files.files().packages.borrow_mut().insert(path.into(), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Virtual', virtual: true}]}\n".to_vec());
    let root = files
        .open(
            path,
            QueryObserver {
                veto: true,
                ..Default::default()
            },
            "test",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    let generation = files.workspace().session(root).unwrap().generation();
    let history = files.workspace().session(root).unwrap().history_entries();
    let result = FileHost::new(&mut files, &mut source, "test").set_occurrence_suppressed(
        root,
        &[(root, generation, 0)],
        true,
    );
    assert!(result.is_err());
    let session = files.workspace().session(root).unwrap();
    assert!(!session.snapshot().assembly.as_ref().unwrap()[0].suppressed);
    assert_eq!(session.history_entries(), history);
    assert!(!session.in_transaction());
}
