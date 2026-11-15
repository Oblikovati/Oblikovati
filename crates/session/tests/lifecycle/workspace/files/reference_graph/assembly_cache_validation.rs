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
