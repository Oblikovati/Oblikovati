// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

fn assembly_without_presenter() -> (DiscoveryWorkspace, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let package = DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly");
    let id = files
        .create(
            Path::new("/fixtures/assembly.oad"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    (files, id, source)
}
fn warning(warnings: Option<Vec<SaveWarning>>) {
    let warnings = warnings.expect("expected save warnings");
    assert_eq!(warnings.len(), 1);
    assert_eq!(warnings[0].code, SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE);
    assert!(warnings[0].detail.contains("presentation service"));
}
#[test]
fn every_typed_save_variant_reports_missing_presentation_service() {
    let (mut files, id, mut source) = assembly_without_presenter();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    warning(
        client
            .documents()
            .save(&SaveDocumentArgs { document: id })
            .unwrap()
            .warnings,
    );
    warning(
        client
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document: id,
                new_full_document_name: "/fixtures/renamed.oad".into(),
            })
            .unwrap()
            .warnings,
    );
    warning(
        client
            .documents()
            .save_copy_as(&SaveCopyAsArgs {
                document: id,
                target_file_name: "/fixtures/copy.oad".into(),
                metadata: None,
            })
            .unwrap()
            .warnings,
    );
    for operation in ["save", "saveAs", "saveCopyAs"] {
        let args = BatchSaveArgs {
            operation: operation.into(),
            items: Some(vec![BatchSaveItem {
                document: id,
                target_file_name: format!("/fixtures/{operation}.oad"),
            }]),
        };
        let result = client.documents().batch_save(&args).unwrap();
        let item = result.results.unwrap().pop().unwrap();
        assert!(item.ok, "{}", item.error);
        warning(item.warnings);
    }
}
#[test]
fn save_on_close_reports_missing_presentation_service() {
    let (mut files, id, mut source) = assembly_without_presenter();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .save(&SaveDocumentArgs { document: id })
        .unwrap();
    files
        .commands(id)
        .unwrap()
        .run(
            "Add virtual occurrence",
            |snapshot, ids| -> Result<(), ()> {
                snapshot
                    .assembly
                    .as_mut()
                    .unwrap()
                    .push(AssemblyOccurrenceRecipe {
                        session_id: ids.allocate_occurrence().unwrap(),
                        name: "Virtual".into(),
                        virtual_: true,
                        ..Default::default()
                    });
                Ok(())
            },
        )
        .unwrap();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close(&CloseDocumentArgs { id, force: false })
        .unwrap();
    warning(result.warnings);
}

#[test]
fn close_all_reports_warnings_for_each_saved_assembly() {
    let (mut files, first, mut source) = assembly_without_presenter();
    let second = files
        .create(
            Path::new("/fixtures/other.oad"),
            DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Other"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .documents()
        .close_all(&CloseAllDocumentsArgs { force: false })
        .unwrap();
    let warnings = result.warnings.unwrap();
    assert_eq!(warnings.len(), 2);
    assert!(
        warnings
            .iter()
            .any(|item| item.document == first
                && item.code == SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE)
    );
    assert!(warnings.iter().any(
        |item| item.document == second && item.code == SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE
    ));
}

#[test]
fn typed_express_open_and_load_full_keep_the_document_identity() {
    use oblikovati_persistence::package::SavedAssemblyView;
    let (mut files, original, mut source) = assembly_without_presenter();
    let destination = files.path(original).unwrap().to_owned();
    let view = SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences: Vec::new(),
        sources: Vec::new(),
        catalog_digest: files.assembly_view_catalog_digest().unwrap(),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: Vec::new(),
    };
    files
        .stage_assembly_view(original, &destination, view)
        .unwrap();
    files.save(original, "test", &mut source).unwrap();
    files.close_discarding_changes(original).unwrap();
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 2,
    };
    let request = OpenDocumentArgs {
        full_document_name: destination.to_string_lossy().into_owned(),
        visible: true,
        express: true,
        ..Default::default()
    };
    let no_policy = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_observer_factory(&mut factory),
    );
    assert!(
        no_policy
            .documents()
            .open(&request)
            .unwrap_err()
            .to_string()
            .contains("display policy")
    );
    assert!(files.workspace().documents().is_empty());
    let strict_policy = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_observer_factory(&mut factory)
            .with_assembly_display_limits((0.01, 0.5)),
    );
    assert!(
        strict_policy
            .documents()
            .open(&request)
            .unwrap_err()
            .to_string()
            .contains("chord")
    );
    assert!(files.workspace().documents().is_empty());
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test")
            .with_observer_factory(&mut factory)
            .with_assembly_display_limits((0.05, 0.5)),
    );
    let opened = client.documents().open(&request).unwrap();
    assert!(opened.express);
    let full = client
        .documents()
        .load_full(&LoadFullDocumentArgs { id: opened.id })
        .unwrap();
    assert_eq!(full.id, opened.id);
    assert!(!full.express);
}

fn assembly_with_saved_component() -> (DiscoveryWorkspace, QueryGuids, PathBuf, PathBuf) {
    use oblikovati_persistence::package::{SavedAssemblyTriangle, SavedAssemblyView};
    let (mut files, root, mut source) = assembly_without_presenter();
    let part_path = fixture_native_path(Path::new("/fixtures/component.opd"));
    let part = files
        .create(
            &part_path,
            DocumentPackage::empty(DOCUMENT_PART, "Component"),
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    files.save(part, "test", &mut source).unwrap();
    let name = part_path.to_string_lossy().into_owned();
    files
        .commands(root)
        .unwrap()
        .run("Place component", |snapshot, ids| -> Result<(), ()> {
            snapshot
                .assembly
                .as_mut()
                .unwrap()
                .push(AssemblyOccurrenceRecipe {
                    session_id: ids.allocate_occurrence().unwrap(),
                    name: "Component:1".into(),
                    component: name.clone(),
                    ..Default::default()
                });
            Ok(())
        })
        .unwrap();
    files
        .attach_document_reference(root, name, part, false)
        .unwrap();
    let destination = files.path(root).unwrap().to_owned();
    let view = SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences: files
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .clone()
            .unwrap(),
        sources: files.assembly_saved_sources(root, &destination).unwrap(),
        catalog_digest: files.assembly_view_catalog_digest().unwrap(),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: vec![SavedAssemblyTriangle {
            positions: [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]],
            color: [0.5, 0.5, 0.5, 1.],
        }],
    };
    files.stage_assembly_view(root, &destination, view).unwrap();
    files.save(root, "test", &mut source).unwrap();
    files.close_discarding_changes(root).unwrap();
    files.close_discarding_changes(part).unwrap();
    (files, source, destination, part_path)
}

#[test]
fn failed_load_full_keeps_express_mode_until_components_resolve() {
    let (mut files, mut source, destination, part_path) = assembly_with_saved_component();
    let storage = Rc::clone(&files.files().packages);
    let mut factory = QueryObserverFactory {
        created: 0,
        limit: 5,
    };
    let opened = {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_observer_factory(&mut factory)
                .with_assembly_display_limits((0.05, 0.5)),
        );
        client
            .documents()
            .open(&OpenDocumentArgs {
                full_document_name: destination.to_string_lossy().into_owned(),
                visible: true,
                express: true,
                ..Default::default()
            })
            .unwrap()
    };
    assert!(opened.express);
    assert_eq!(files.workspace().documents(), &[opened.id]);
    let part_bytes = storage.borrow_mut().remove(&part_path).unwrap();
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_observer_factory(&mut factory)
                .with_assembly_display_limits((0.05, 0.5)),
        );
        assert!(
            client
                .documents()
                .load_full(&LoadFullDocumentArgs { id: opened.id })
                .is_err()
        );
    }
    assert!(files.express_assembly_view(opened.id).unwrap().is_some());
    storage.borrow_mut().insert(part_path, part_bytes);
    let full = {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test")
                .with_observer_factory(&mut factory)
                .with_assembly_display_limits((0.05, 0.5)),
        );
        client
            .documents()
            .load_full(&LoadFullDocumentArgs { id: opened.id })
            .unwrap()
    };
    assert_eq!(full.id, opened.id);
    assert!(!full.express);
    assert_eq!(files.workspace().documents().len(), 2);
}
