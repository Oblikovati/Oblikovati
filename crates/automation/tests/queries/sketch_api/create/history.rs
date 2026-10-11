// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn grouped_creations(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, abort: bool) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Create two sketches".into(),
        })
        .unwrap();
    client.sketch().create(&origin_request("XY")).unwrap();
    client.sketch().create(&origin_request("YZ")).unwrap();
    if abort {
        client.transactions().abort().unwrap();
        return;
    }
    client.transactions().end().unwrap();
}
#[test]
fn typed_creation_joins_history_groups_and_preserves_saved_anchor_identities() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    grouped_creations(&mut files, &mut source, false);
    let authored = snapshot(&files, document).sketches.clone();
    assert_eq!(owner_revision(&files, document).1, ["Create two sketches"]);
    files.commands(document).unwrap().undo().unwrap();
    assert!(snapshot(&files, document).sketches.sketches().is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert!(authored.same_recipe(&snapshot(&files, document).sketches));
    let reopened =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert!(authored.same_recipe(&snapshot(&files, reopened).sketches));
    assert_created(&files, reopened, 0, true);
    assert_created(&files, reopened, 1, true);
    assert!(!files.workspace().needs_save(reopened).unwrap());
}
#[test]
fn an_aborted_creation_group_cannot_reuse_either_namespace() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    grouped_creations(&mut files, &mut source, true);
    assert!(snapshot(&files, document).sketches.sketches().is_empty());
    assert!(owner_revision(&files, document).1.is_empty());
    let reply = create_call(&mut files, &mut source, &Default::default()).unwrap();
    assert_eq!(reply.sketch_index, 0);
    let placed = &snapshot(&files, document).sketches.sketches()[0];
    assert_eq!(placed.id().0, 3);
    assert_eq!(placed.geometry().points()[0].id.0, 3);
}
#[test]
fn a_transaction_veto_restores_the_saved_creation_baseline() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let document = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let before = owner_revision(&files, document);
    expect_api_decline(
        create_call(&mut files, &mut source, &Default::default()),
        "locked",
    );
    assert!(snapshot(&files, document).sketches.sketches().is_empty());
    assert_eq!(owner_revision(&files, document).1, before.1);
}
#[test]
fn malformed_creation_json_and_query_only_hosts_do_not_mutate_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    {
        let host = FileHost::new(&mut files, &mut source, "test");
        expect_api_decline(host.call(METHOD_SKETCH_CREATE, b"{"), METHOD_SKETCH_CREATE);
        expect_api_decline(
            host.call(METHOD_SKETCH_CREATE, br#"{"workPlaneIndex":"bad"}"#),
            METHOD_SKETCH_CREATE,
        );
    }
    assert!(
        Client::new(FileQueryHost::new(&files))
            .sketch()
            .create(&Default::default())
            .is_err()
    );
    assert_eq!(before, owner_revision(&files, document));
}
