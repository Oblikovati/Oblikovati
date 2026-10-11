// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{METHOD_SKETCH_DELETE, OKResult};

fn delete_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .delete(&SketchArgs {
            sketch_index: index,
        })
}

#[test]
fn typed_deletion_removes_only_the_selected_identity_and_is_one_undo_step() {
    let (mut files, document, first, mut source) = files_with_circle(Default::default());
    let second = insert_circle(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let before = owned_geometry(&files, document).clone();
    assert!(delete_call(&mut files, &mut source, 0).unwrap().ok);
    assert_eq!(owned_geometry(&files, document).sketch_ids(), [second.0]);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Delete sketch"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert!(before.same_evaluated_state(owned_geometry(&files, document)));
    assert!(!files.workspace().needs_save(document).unwrap());
    files.commands(document).unwrap().redo().unwrap();
    assert!(owned_geometry(&files, document).get(first).is_err());
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_DELETE));
}

#[test]
fn typed_deletion_declines_invalid_indices_and_malformed_requests_before_mutation() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(
            delete_call(&mut files, &mut source, index),
            &index.to_string(),
        );
    }
    let host = FileHost::new(&mut files, &mut source, "test");
    expect_api_decline(host.call(METHOD_SKETCH_DELETE, b"{"), METHOD_SKETCH_DELETE);
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn typed_deletion_requires_an_active_part_and_a_writable_host() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    expect_api_decline(delete_call(&mut files, &mut source, 0), "active part");
    for kind in [DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PRESENTATION] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let before = owner_revision(&files, document);
        expect_api_decline(delete_call(&mut files, &mut source, 0), "kind");
        assert_eq!(before, owner_revision(&files, document));
    }
    let (files, _, _, _) = files_with_circle(Default::default());
    assert!(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .delete(&SketchArgs::default())
            .is_err()
    );
}

#[test]
fn deletion_veto_preserves_saved_sketches_and_history() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    expect_api_decline(delete_call(&mut files, &mut source, 0), "locked");
    let after = owner_revision(&files, document);
    assert_eq!((before.1, before.2), (after.1, after.2));
    assert!(after.0 > before.0);
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}

#[path = "delete/construction.rs"]
mod construction;

#[test]
fn typed_deletion_joins_a_group_and_abort_restores_the_sketch() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owned_geometry(&files, document).clone();
    {
        let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
        client
            .transactions()
            .begin(&TransactionBeginArgs {
                label: "Delete group".into(),
            })
            .unwrap();
        assert!(client.sketch().delete(&SketchArgs::default()).unwrap().ok);
        client.transactions().abort().unwrap();
    }
    assert!(before.same_evaluated_state(owned_geometry(&files, document)));
}
