// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn cached_graph() -> (ReferenceHost, u64, PathBuf, DocumentPackage) {
    let (mut host, _, part) = source_graph();
    let mut guids = SequenceGuids(6000);
    let leaf = path("a.obk").to_str().unwrap().to_owned();
    let root = assembly(&mut host, &mut guids, &[&leaf]);
    host.attach_document_reference(root, leaf, part, false)
        .unwrap();
    let destination = path("cached.oad");
    let mut view = synthetic_view(&mut host, root, &destination);
    view.catalog_digest = host.assembly_view_catalog_digest().unwrap();
    host.stage_assembly_view(root, &destination, view).unwrap();
    host.save_as(root, &destination, "alpha", &mut guids)
        .unwrap();
    let package = DocumentPackage::decode(&host.files().contents.borrow()[&destination]).unwrap();
    (host, part, destination, package)
}

#[test]
fn validates_unchanged_sources_and_rejects_changed_or_missing_bytes() {
    let (mut host, _, destination, package) = cached_graph();
    assert!(
        host.validated_assembly_view(&destination, &package, &"c".repeat(64))
            .unwrap()
            .is_some()
    );
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(2));
    let error = host
        .validated_assembly_view(&destination, &package, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected saved content digest"), "{error}");
    host.files().contents.borrow_mut().remove(&path("a.obk"));
    let error = host
        .validated_assembly_view(&destination, &package, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected an existing source"), "{error}");
}

#[test]
fn saved_assembly_view_rejects_another_producer() {
    let (mut host, _, destination, package) = cached_graph();
    let error = host
        .validated_assembly_view(&destination, &package, &"d".repeat(64))
        .unwrap_err();
    assert!(error.contains("producer"), "{error}");
}

#[test]
fn rejects_dirty_open_component_and_a_changed_catalog() {
    let (mut host, part, destination, package) = cached_graph();
    host.commands(part)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let error = host
        .validated_assembly_view(&destination, &package, &"c".repeat(64))
        .unwrap_err();
    assert!(
        error.contains("expected a saved committed document"),
        "{error}"
    );
    let mut view = package.restore_saved_assembly_view().unwrap().unwrap();
    view.catalog_digest = "b".repeat(64);
    let stale = package.with_saved_assembly_view(&view).unwrap();
    let error = host
        .validated_assembly_view(&destination, &stale, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("current catalog"), "{error}");
}

#[test]
fn absent_cache_is_distinct_from_an_invalid_cache() {
    let (mut host, _, destination, _) = cached_graph();
    let plain = DocumentPackage::decode(&fixture(1)).unwrap();
    assert_eq!(
        host.validated_assembly_view(&destination, &plain, &"c".repeat(64))
            .unwrap(),
        None
    );
}

#[test]
fn nested_cache_rejects_a_changed_leaf_and_an_incomplete_graph() {
    let (mut host, _, part) = source_graph();
    let mut guids = SequenceGuids(7000);
    let leaf = path("a.obk").to_str().unwrap().to_owned();
    let nested = assembly(&mut host, &mut guids, &[&leaf]);
    host.attach_document_reference(nested, leaf, part, false)
        .unwrap();
    let nested_path = path("nested.oad");
    host.save_as(nested, &nested_path, "alpha", &mut guids)
        .unwrap();
    let child = nested_path.to_str().unwrap().to_owned();
    let root = assembly(&mut host, &mut guids, &[&child]);
    host.attach_document_reference(root, child, nested, false)
        .unwrap();
    let destination = path("nested-root.oad");
    let mut view = synthetic_view(&mut host, root, &destination);
    view.catalog_digest = host.assembly_view_catalog_digest().unwrap();
    host.stage_assembly_view(root, &destination, view.clone())
        .unwrap();
    host.save_as(root, &destination, "alpha", &mut guids)
        .unwrap();
    let package = DocumentPackage::decode(&host.files().contents.borrow()[&destination]).unwrap();
    assert!(
        host.validated_assembly_view(&destination, &package, &"c".repeat(64))
            .unwrap()
            .is_some()
    );
    let mut incomplete = view;
    incomplete.sources.pop();
    let tampered = package.with_saved_assembly_view(&incomplete).unwrap();
    let error = host
        .validated_assembly_view(&destination, &tampered, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected a source record"), "{error}");
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(2));
    let error = host
        .validated_assembly_view(&destination, &package, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected saved content digest"), "{error}");
}

#[test]
fn rejects_changed_resolution_and_source_identity() {
    let (mut host, _, destination, package) = cached_graph();
    let mut view = package.restore_saved_assembly_view().unwrap().unwrap();
    view.sources[0].resolved_path = path("other.obk");
    let moved = package.with_saved_assembly_view(&view).unwrap();
    let error = host
        .validated_assembly_view(&destination, &moved, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected saved path"), "{error}");
    view.sources[0].resolved_path = path("a.obk");
    view.sources[0].identity = guid(2).to_string();
    let changed = package.with_saved_assembly_view(&view).unwrap();
    let error = host
        .validated_assembly_view(&destination, &changed, &"c".repeat(64))
        .unwrap_err();
    assert!(error.contains("expected saved identity"), "{error}");
}

#[test]
fn express_open_keeps_components_unloaded_and_rejects_mode_changes() {
    let (saved, _, destination, _) = cached_graph();
    let files = WorkspaceFiles {
        contents: Rc::clone(&saved.files().contents),
        ..Default::default()
    };
    let mut fresh = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut guids = SequenceGuids(8000);
    let id = fresh
        .open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            &mut guids,
            (0.05, 0.5),
            &"c".repeat(64),
        )
        .unwrap()
        .session_id();
    assert_eq!(fresh.workspace().documents(), &[id]);
    assert!(fresh.express_assembly_view(id).unwrap().is_some());
    assert!(
        matches!(fresh.commands(id), Err(WorkspaceError::ExpressReadOnly(found)) if found == id)
    );
    let derived = fresh.add_derived_table(id, "missing.opd", &[]).unwrap_err();
    assert!(derived.to_string().contains("expected express mode false"));
    let inference = fresh.set_sketch_inference_options_projected(
        &oblikovati_api::wire::InferenceOptionsView::default(),
        |_| Ok::<(), String>(()),
    );
    assert!(
        matches!(inference, Err(oblikovati_session::workspace::SketchInferenceCommandError::Workspace(WorkspaceError::ExpressReadOnly(found))) if found == id)
    );
    let original = fresh.files().contents.borrow()[&destination].clone();
    assert!(fresh.save(id, "alpha", &mut guids).is_err());
    assert!(
        fresh
            .save_as(id, &path("copy.oad"), "alpha", &mut guids)
            .is_err()
    );
    assert!(
        fresh
            .save_copy(id, &path("copy.oad"), None, "alpha", &mut guids)
            .is_err()
    );
    assert_eq!(fresh.files().contents.borrow()[&destination], original);
    let error = fresh
        .open(
            &destination,
            ContentObserver,
            "alpha",
            &mut guids,
            OpenActivation::Activate,
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("expected express mode false"),
        "{error}"
    );
    fresh.close_saving_changes(id, "alpha", &mut guids).unwrap();
    assert!(fresh.workspace().documents().is_empty());
    assert_eq!(fresh.files().contents.borrow()[&destination], original);
}

#[test]
fn express_open_of_stale_cache_does_not_register_a_document() {
    let (saved, _, destination, _) = cached_graph();
    saved
        .files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(2));
    let files = WorkspaceFiles {
        contents: Rc::clone(&saved.files().contents),
        ..Default::default()
    };
    let mut fresh = FileWorkspace::new(RecordingWorkspace::default(), files);
    let error = fresh
        .open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            &mut SequenceGuids(9000),
            (0.05, 0.5),
            &"c".repeat(64),
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("expected saved content digest"),
        "{error}"
    );
    assert!(fresh.workspace().documents().is_empty());
}

#[test]
fn express_open_refuses_insufficient_mesh_quality_before_registration_or_reuse() {
    let (saved, _, destination, _) = cached_graph();
    let files = WorkspaceFiles {
        contents: Rc::clone(&saved.files().contents),
        ..Default::default()
    };
    let mut fresh = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut guids = SequenceGuids(9500);
    let invalid_producer = |host: &mut ReferenceHost, guids: &mut SequenceGuids| {
        host.open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            guids,
            (0.05, 0.5),
            &"d".repeat(64),
        )
    };
    assert!(invalid_producer(&mut fresh, &mut guids).is_err());
    assert!(fresh.workspace().documents().is_empty());
    let open = |host: &mut ReferenceHost, guids: &mut SequenceGuids, limits| {
        host.open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            guids,
            limits,
            &"c".repeat(64),
        )
    };
    for limits in [(0.001, 0.5), (0.05, 0.001), (f64::NAN, 0.5)] {
        assert!(open(&mut fresh, &mut guids, limits).is_err());
        assert!(fresh.workspace().documents().is_empty());
    }
    let id = open(&mut fresh, &mut guids, (0.05, 0.5))
        .unwrap()
        .session_id();
    assert!(invalid_producer(&mut fresh, &mut guids).is_err());
    assert!(open(&mut fresh, &mut guids, (0.001, 0.5)).is_err());
    assert_eq!(fresh.workspace().documents(), &[id]);
}

#[test]
fn express_open_without_cache_is_refused_without_changing_the_workspace() {
    let files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("plain.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let error = host
        .open_express_with_factory(
            &path("plain.obk"),
            || Ok(ContentObserver),
            "alpha",
            &mut SequenceGuids(10000),
            (0.05, 0.5),
            &"c".repeat(64),
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("expected a saved assembly view"),
        "{error}"
    );
    assert!(host.workspace().documents().is_empty());
}

#[test]
fn express_open_does_not_reuse_a_fully_open_document() {
    let (mut host, _, destination, _) = cached_graph();
    let error = host
        .open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            &mut SequenceGuids(11000),
            (0.05, 0.5),
            &"c".repeat(64),
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("expected express mode true"),
        "{error}"
    );
}

#[test]
fn load_full_requires_attached_components_before_enabling_edits() {
    let (saved, _, destination, _) = cached_graph();
    let files = WorkspaceFiles {
        contents: Rc::clone(&saved.files().contents),
        ..Default::default()
    };
    let mut fresh = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut guids = SequenceGuids(12000);
    let root = fresh
        .open_express_with_factory(
            &destination,
            || Ok(ContentObserver),
            "alpha",
            &mut guids,
            (0.05, 0.5),
            &"c".repeat(64),
        )
        .unwrap()
        .session_id();
    let error = fresh.finish_express_assembly(root).unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected a fully loaded component"),
        "{error}"
    );
    assert!(fresh.express_assembly_view(root).unwrap().is_some());
    let part = fresh
        .open(
            &path("a.obk"),
            ContentObserver,
            "alpha",
            &mut guids,
            OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    fresh
        .attach_document_reference(root, path("a.obk").to_str().unwrap().into(), part, true)
        .unwrap();
    assert!(fresh.finish_express_assembly(root).unwrap());
    assert!(fresh.express_assembly_view(root).unwrap().is_none());
    assert!(fresh.commands(root).is_ok());
    assert!(!fresh.finish_express_assembly(root).unwrap());
}
