// SPDX-License-Identifier: GPL-2.0-only
//! Feature deletion retains only independently certified earlier features.
use super::*;
use oblikovati_automation::PartUpdateAccess;

#[test]
fn deleting_later_feature_reuses_prefix_and_preserves_names() {
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
    Client::new(owner.document_host())
        .features()
        .delete(&FeatureRefArgs { id: 2 })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    assert!(
        owner
            .part_scenes
            .current_part(&owner.files, document)
            .is_ok()
    );
    let remaining = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(remaining.bodies.as_ref().unwrap().len(), 1);
    assert_eq!(
        remaining.bodies.as_ref().unwrap()[0].faces,
        original.bodies.as_ref().unwrap()[0].faces
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        original
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        remaining
    );
    super::cutoff_checkpoint::save_close_reopen(&mut owner, document);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        remaining
    );
}

#[test]
fn deleting_first_feature_uses_full_replay() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, true);
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    Client::new(owner.document_host())
        .features()
        .delete(&FeatureRefArgs { id: 1 })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before + 1);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap()
            .bodies
            .unwrap()
            .len(),
        1
    );
}
