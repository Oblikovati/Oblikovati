// SPDX-License-Identifier: GPL-2.0-only
//! End-of-part edits reuse only a proven current feature prefix.
use super::*;
use oblikovati_automation::PartUpdateAccess;

#[test]
fn cutoff_reuses_checkpoints_and_preserves_names_across_restore_and_history() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, true);
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    let original = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(original.bodies.as_ref().unwrap().len(), 2);
    let reply = Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: 1 })
        .unwrap();
    assert_eq!(reply.position, 1);
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    assert!(
        owner
            .part_scenes
            .current_part(&owner.files, document)
            .is_ok()
    );
    let rolled = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(rolled.bodies.as_ref().unwrap().len(), 1);
    assert_eq!(
        rolled.bodies.as_ref().unwrap()[0].faces,
        original.bodies.as_ref().unwrap()[0].faces
    );
    Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: -1 })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    let restored = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(restored, original);
    owner.files.commands(document).unwrap().undo().unwrap();
    let undone = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(undone, rolled);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        original
    );
    save_close_reopen(&mut owner, document);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        original
    );
}

pub(super) fn save_close_reopen(owner: &mut TestOwner, document: u64) {
    let path = document_fixture_path!("/memory/checkpoint-replay.opd");
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
}

#[test]
fn cutoff_in_open_transaction_uses_full_replay_and_abort_restores_part() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, true);
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let original = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Temporary cutoff".into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: 1 })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before + 1);
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        original
    );
}

#[test]
fn cutoff_does_not_use_checkpoints_from_another_policy() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut other = oblikovati_automation::PartReplayService::with_policy(policy).unwrap();
    other
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    Client::new(owner.document_host())
        .documents()
        .set_end_of_part(&SetEndOfPartArgs { position: 1 })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before + 1);
    assert!(other.part_requires_update(&owner.files, document).unwrap());
}
