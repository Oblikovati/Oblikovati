// SPDX-License-Identifier: GPL-2.0-only
//! Feature reordering replays from the earliest changed feature.
use super::*;
use oblikovati_automation::PartUpdateAccess;

fn add_third_prism(owner: &mut TestOwner, document: u64) {
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Add independent feature", |snapshot, ids| {
            let mut recipe = fixture::selected_prism_definition().recipe();
            recipe.displacement[2] = DatumLength::Constant(3.);
            let definition =
                oblikovati_model::feature::sketch_prism::SketchPrismDefinition::from_recipe(
                    &recipe,
                )
                .unwrap();
            assert_eq!(
                snapshot
                    .create_sketch_prism("Third", definition, ids)
                    .unwrap(),
                FeatureId(3)
            );
            Ok::<_, String>(())
        })
        .unwrap();
}

#[test]
fn reorder_later_features_reuses_prefix_and_preserves_names() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, true);
    add_third_prism(&mut owner, document);
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    let original = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    assert!(
        Client::new(owner.document_host())
            .features()
            .reorder(&ReorderFeatureArgs {
                id: 3,
                new_index: 99,
            })
            .is_err()
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    let result = Client::new(owner.document_host())
        .features()
        .reorder(&ReorderFeatureArgs {
            id: 3,
            new_index: 1,
        })
        .unwrap();
    assert_eq!(result.feature.index, 1);
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    assert!(
        owner
            .part_scenes
            .current_part(&owner.files, document)
            .is_ok()
    );
    let reordered = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let old = original.bodies.as_ref().unwrap();
    let new = reordered.bodies.as_ref().unwrap();
    assert_eq!(new.len(), 3);
    assert_eq!(new[0].faces, old[0].faces);
    assert_eq!(new[1].faces, old[2].faces);
    assert_eq!(new[2].faces, old[1].faces);
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
        reordered
    );
    super::cutoff_checkpoint::save_close_reopen(&mut owner, document);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        reordered
    );
}

#[test]
fn reorder_first_feature_uses_full_replay() {
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
        .reorder(&ReorderFeatureArgs {
            id: 2,
            new_index: 0,
        })
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before + 1);
}
