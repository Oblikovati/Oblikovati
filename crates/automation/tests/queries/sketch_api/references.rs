// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{ResolveSketchReferenceArgs, ResolveSketchReferenceResult};
use oblikovati_model::document_sketch::DocumentSketchReferenceIndex;

fn sketch_key(files: &DiscoveryWorkspace, index: i64) -> String {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .reference_key(&SketchArgs {
            sketch_index: index,
        })
        .unwrap()
        .reference_key
}
fn resolve(files: &DiscoveryWorkspace, key: &str) -> ResolveSketchReferenceResult {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .resolve_reference(&ResolveSketchReferenceArgs {
            reference_key: key.into(),
        })
        .unwrap()
}
fn entity_keys(files: &DiscoveryWorkspace, document: u64) -> Vec<(String, u64)> {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let guid = files.workspace().identity(document).unwrap().internal_name;
    let names = DocumentSketchReferenceIndex::new(&snapshot.sketches, guid).unwrap();
    let sketch = &snapshot.sketches.sketches()[0];
    sketch
        .geometry()
        .entity_ids()
        .iter()
        .map(|entity| (names.entity_key(sketch.id(), *entity).unwrap(), entity.0))
        .collect()
}
fn assert_target(files: &DiscoveryWorkspace, key: &str, index: i64, entity: Option<u64>) {
    let target = resolve(files, key);
    assert!(target.found);
    assert_eq!(target.sketch_index, index);
    assert_eq!(target.entity_id, entity.unwrap_or(0));
    assert_eq!(
        target.kind,
        if entity.is_some() {
            "sketchEntity"
        } else {
            "sketch"
        }
    );
}
#[test]
fn canonical_queries_round_trip_entities_and_are_read_only_in_both_hosts() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let key = sketch_key(&files, 0);
    assert_target(&files, &key, 0, None);
    for (key, entity) in entity_keys(&files, document) {
        assert_target(&files, &key, 0, Some(entity));
    }
    let reply = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .reference_key(&SketchArgs::default())
        .unwrap();
    assert_eq!(reply.reference_key, key);
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn deletion_changes_indices_without_changing_surviving_keys_and_undo_restores_targets() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    insert_circle(&mut files, document);
    let first = sketch_key(&files, 0);
    let second = sketch_key(&files, 1);
    let entities = entity_keys(&files, document);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .delete(&SketchArgs::default())
        .unwrap();
    assert!(!resolve(&files, &first).found);
    assert!(entities.iter().all(|(key, _)| !resolve(&files, key).found));
    assert_target(&files, &second, 0, None);
    assert_eq!(sketch_key(&files, 0), second);
    files.commands(document).unwrap().undo().unwrap();
    assert_target(&files, &first, 0, None);
    assert_target(&files, &second, 1, None);
}
#[test]
fn save_reopen_preserves_document_and_entity_names() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let key = sketch_key(&files, 0);
    let entities = entity_keys(&files, document);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(sketch_key(&files, 0), key);
    assert_target(&files, &key, 0, None);
    for (key, entity) in entities {
        assert_target(&files, &key, 0, Some(entity));
    }
    assert!(!files.workspace().needs_save(reopened).unwrap());
}
#[test]
fn save_copy_creates_a_new_namespace_and_activation_uses_only_that_document() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let key = sketch_key(&files, 0);
    let entities = entity_keys(&files, document);
    let path = Path::new("/fixtures/reference-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    let copied = open_copy(&mut files, &mut source, path);
    assert_ne!(sketch_key(&files, 0), key);
    assert!(!resolve(&files, &key).found);
    assert!(entities.iter().all(|(key, _)| !resolve(&files, key).found));
    for (key, entity) in entity_keys(&files, copied) {
        assert_target(&files, &key, 0, Some(entity));
    }
}
#[test]
fn save_as_keeps_the_document_namespace() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let key = sketch_key(&files, 0);
    files
        .save_as(
            document,
            Path::new("/fixtures/reference-renamed.obk"),
            "test",
            &mut source,
        )
        .unwrap();
    assert_eq!(sketch_key(&files, 0), key);
    assert_target(&files, &key, 0, None);
}
#[test]
fn unknown_and_noncanonical_reference_text_is_a_named_miss() {
    let (files, _, _, _) = files_with_circle(Default::default());
    let key = sketch_key(&files, 0);
    for unknown in [
        "".into(),
        "invalid".into(),
        key.to_uppercase(),
        format!(" {key}"),
    ] {
        assert_eq!(
            resolve(&files, &unknown),
            ResolveSketchReferenceResult::default()
        );
    }
}
#[test]
fn assembly_sketches_have_the_same_reference_contract() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    assert_target(&files, &sketch_key(&files, 0), 0, None);
    assert_eq!(owner_revision(&files, document).1, ["Create sketch"]);
}
#[test]
fn invalid_indices_and_malformed_requests_decline_without_mutation() {
    let (files, document, _, _) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    let host = FileQueryHost::new(&files);
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .reference_key(&SketchArgs {
                    sketch_index: index,
                }),
            &index.to_string(),
        );
    }
    for method in [METHOD_SKETCH_REFERENCE_KEY, METHOD_SKETCH_RESOLVE_REFERENCE] {
        expect_api_decline(host.call(method, b"{"), method);
    }
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn reference_queries_require_an_active_part_or_assembly_owner() {
    let files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    expect_api_decline(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .reference_key(&SketchArgs::default()),
        "active part",
    );
    for kind in [DOCUMENT_DRAWING, DOCUMENT_PRESENTATION] {
        let (files, document, _) = configured_files(kind, Default::default());
        let before = owner_revision(&files, document);
        expect_api_decline(
            Client::new(FileQueryHost::new(&files))
                .sketch()
                .resolve_reference(&Default::default()),
            "kind",
        );
        assert_eq!(owner_revision(&files, document), before);
    }
}

fn open_copy(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, path: &Path) -> u64 {
    files
        .open(
            path,
            Default::default(),
            "test",
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id()
}

#[test]
fn grouped_deletion_abort_and_commit_use_only_current_targets() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let key = sketch_key(&files, 0);
    begin_grouped_deletion(&mut files, document, &mut source);
    assert!(!resolve(&files, &key).found);
    files.commands(document).unwrap().abort().unwrap();
    assert_target(&files, &key, 0, None);
    begin_grouped_deletion(&mut files, document, &mut source);
    files.commands(document).unwrap().end().unwrap();
    assert!(!resolve(&files, &key).found);
    files.commands(document).unwrap().undo().unwrap();
    assert_target(&files, &key, 0, None);
    files.commands(document).unwrap().redo().unwrap();
    assert!(!resolve(&files, &key).found);
}

fn begin_grouped_deletion(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    files
        .commands(document)
        .unwrap()
        .begin("Reference deletion")
        .unwrap();
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .delete(&SketchArgs::default())
        .unwrap();
}
