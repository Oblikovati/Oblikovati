// SPDX-License-Identifier: GPL-2.0-only
//! Prism distance edits preserve a proven clean feature prefix.
use super::*;
use oblikovati_automation::PartUpdateAccess;

fn edit_distance(
    owner: &mut TestOwner,
    id: u64,
    value: &str,
) -> Result<FeatureDetailResult, oblikovati_api::Error> {
    Client::new(owner.document_host())
        .features()
        .edit(&EditFeatureArgs {
            id,
            scalars: Some(vec![ScalarEdit {
                index: 0,
                value: value.into(),
            }]),
            ..Default::default()
        })
}

fn assert_matches_full_replay(owner: &TestOwner, document: u64) {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let expected = session
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let actual = owner
        .part_scenes
        .current_part(&owner.files, document)
        .unwrap();
    let expected_body = expected.body(FeatureId(2)).unwrap();
    let actual_body = actual.body(FeatureId(2)).unwrap();
    let keys = expected_body
        .face_reference_keys(session.history(), guid)
        .unwrap();
    assert_eq!(
        keys,
        actual_body
            .face_reference_keys(session.history(), guid)
            .unwrap()
    );
    for key in keys {
        let expected_area = expected_body
            .face_area(session.history(), guid, &key)
            .unwrap();
        let actual_area = actual_body
            .face_area(session.history(), guid, &key)
            .unwrap();
        assert!((expected_area - actual_area).abs() < 1e-10);
    }
}

#[test]
fn edit_later_prism_reuses_prefix_and_matches_full_replay() {
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
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    assert!(edit_distance(&mut owner, 2, "-5 mm").is_err());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    let detail = edit_distance(&mut owner, 2, "30 mm").unwrap();
    assert!(detail.feature.feature_info.health.is_empty());
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    assert_matches_full_replay(&owner, document);
    let edited = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    assert_eq!(
        edited.bodies.as_ref().unwrap()[0].faces,
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
        edited
    );
    super::cutoff_checkpoint::save_close_reopen(&mut owner, document);
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .reference_keys()
            .unwrap(),
        edited
    );
}

#[test]
fn edit_first_prism_uses_full_replay() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, true);
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    edit_distance(&mut owner, 1, "30 mm").unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before + 1);
}
