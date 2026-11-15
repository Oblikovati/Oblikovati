// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn source_graph() -> (ReferenceHost, u64, u64) {
    let files = WorkspaceFiles::default();
    files
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(1));
    let mut host = FileWorkspace::new(RecordingWorkspace::default(), files);
    let mut guids = SequenceGuids(1000);
    let part = host
        .open(
            &path("a.obk"),
            ContentObserver,
            "alpha",
            &mut guids,
            OpenActivation::Background,
        )
        .unwrap()
        .session_id();
    let root = assembly(&mut host, &mut guids, &["a.obk", "a.obk"]);
    host.attach_document_reference(root, "a.obk".into(), part, false)
        .unwrap();
    (host, root, part)
}

#[test]
fn repeated_sources_keep_occurrence_order_and_exact_bytes() {
    let (mut host, root, _) = source_graph();
    let sources = host
        .assembly_saved_sources(root, &path("root.oad"))
        .unwrap();
    assert_eq!(sources.len(), 2);
    assert_eq!(sources[0].occurrence_path, vec![0]);
    assert_eq!(sources[1].occurrence_path, vec![1]);
    assert_eq!(sources[0].digest, sources[1].digest);
    assert_eq!(sources[0].resolved_path, path("a.obk"));
    let bytes = fixture(1);
    assert_eq!(
        sources[0].digest,
        oblikovati_model::identity::RecipeDigest::of_recipe(&bytes).to_string()
    );
}

#[test]
fn changed_disk_bytes_and_wrong_destination_are_refused() {
    let (mut host, root, _) = source_graph();
    assert!(
        host.assembly_saved_sources(root, &path("elsewhere/root.oad"))
            .is_err()
    );
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), fixture(2));
    let error = host
        .assembly_saved_sources(root, &path("root.oad"))
        .unwrap_err();
    assert!(error.contains("expected the package loaded"), "{error}");
}

#[test]
fn unsaved_source_is_refused() {
    let (mut host, root, part) = source_graph();
    host.commands(part)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    let error = host
        .assembly_saved_sources(root, &path("root.oad"))
        .unwrap_err();
    assert!(
        error.contains("expected a saved committed source"),
        "{error}"
    );
}

#[test]
fn nested_sources_retain_parent_paths_and_repeated_leaves() {
    let (mut host, _, part) = source_graph();
    let mut guids = SequenceGuids(2000);
    let leaf = path("a.obk").to_str().unwrap().to_owned();
    let nested = assembly(&mut host, &mut guids, &[&leaf, &leaf]);
    host.attach_document_reference(nested, leaf, part, false)
        .unwrap();
    host.save_as(nested, &path("nested.oad"), "alpha", &mut guids)
        .unwrap();
    let child = path("nested.oad").to_str().unwrap().to_owned();
    let root = assembly(&mut host, &mut guids, &[&child]);
    host.attach_document_reference(root, child, nested, false)
        .unwrap();
    let records = host
        .assembly_saved_sources(root, &path("root.oad"))
        .unwrap();
    let paths: Vec<_> = records
        .iter()
        .map(|record| record.occurrence_path.as_slice())
        .collect();
    assert_eq!(paths, vec![&[0][..], &[0, 0][..], &[0, 1][..]]);
}

#[test]
fn missing_component_reference_is_refused() {
    let (mut host, _, _) = source_graph();
    let mut guids = SequenceGuids(3000);
    let root = assembly(&mut host, &mut guids, &[""]);
    let error = host
        .assembly_saved_sources(root, &path("root.oad"))
        .unwrap_err();
    assert!(
        error.contains("expected a saved component reference"),
        "{error}"
    );
}

fn synthetic_view(
    host: &mut ReferenceHost,
    root: u64,
    destination: &Path,
) -> oblikovati_persistence::package::SavedAssemblyView {
    use oblikovati_persistence::package::{SavedAssemblyTriangle, SavedAssemblyView};
    SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences: host
            .workspace()
            .session(root)
            .unwrap()
            .snapshot()
            .assembly
            .clone()
            .unwrap(),
        sources: host.assembly_saved_sources(root, destination).unwrap(),
        catalog_digest: "a".repeat(64),
        producer_digest: "c".repeat(64),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: vec![SavedAssemblyTriangle {
            positions: [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]],
            color: [0.2, 0.3, 0.4, 1.],
        }],
    }
}

#[test]
fn failed_atomic_save_keeps_previous_cache_and_a_retry_drops_unstaged_cache() {
    let (mut host, _, part) = source_graph();
    let mut guids = SequenceGuids(4000);
    let leaf = path("a.obk").to_str().unwrap().to_owned();
    let root = assembly(&mut host, &mut guids, &[&leaf]);
    host.attach_document_reference(root, leaf, part, false)
        .unwrap();
    let destination = path("saved.oad");
    let view = synthetic_view(&mut host, root, &destination);
    host.stage_assembly_view(root, &destination, view.clone())
        .unwrap();
    host.save_as(root, &destination, "alpha", &mut guids)
        .unwrap();
    let original = host.files().contents.borrow()[&destination].clone();
    assert!(
        DocumentPackage::decode(&original)
            .unwrap()
            .restore_saved_assembly_view()
            .unwrap()
            .is_some()
    );
    host.stage_assembly_view(root, &destination, view).unwrap();
    host.files().write_control.set(true);
    assert!(host.save(root, "alpha", &mut guids).is_err());
    assert_eq!(host.files().contents.borrow()[&destination], original);
    host.files().write_control.set(false);
    host.save(root, "alpha", &mut guids).unwrap();
    let retry = host.files().contents.borrow()[&destination].clone();
    assert_eq!(
        DocumentPackage::decode(&retry)
            .unwrap()
            .restore_saved_assembly_view()
            .unwrap(),
        None
    );
}

#[test]
fn staged_cache_cannot_follow_a_changed_save_destination() {
    let (mut host, _, part) = source_graph();
    let mut guids = SequenceGuids(5000);
    let leaf = path("a.obk").to_str().unwrap().to_owned();
    let root = assembly(&mut host, &mut guids, &[&leaf]);
    host.attach_document_reference(root, leaf, part, false)
        .unwrap();
    let planned = path("planned.oad");
    let other = path("other.oad");
    let view = synthetic_view(&mut host, root, &planned);
    host.stage_assembly_view(root, &planned, view).unwrap();
    let error = host.save_as(root, &other, "alpha", &mut guids).unwrap_err();
    assert!(
        error.to_string().contains("expected staged destination"),
        "{error}"
    );
    assert!(!host.files().contents.borrow().contains_key(&other));
}

#[path = "assembly_cache_validation.rs"]
mod cache_validation;
