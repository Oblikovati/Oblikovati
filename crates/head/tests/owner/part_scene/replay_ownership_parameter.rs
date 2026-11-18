// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn parameter_edit_adopts_a_checkpointed_named_fillet_and_keeps_undo() {
    use oblikovati_model::identity::SemanticReferenceName;
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_named_prism(&mut owner, "parameter-checkpoint.opd");
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "radius".into(),
            expression: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    let edge_text: String = edge.iter().map(|byte| format!("{byte:02x}")).collect();
    Client::new(owner.document_host())
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(vec![edge_text]),
            radius: "radius".into(),
            ..Default::default()
        })
        .unwrap();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = replay.replay_count();
    set_parameter(&mut owner, "radius", "1.5 mm");
    assert!(replay.current_part(&owner.files, document).is_ok());
    set_parameter(&mut owner, "radius", "1000 mm");
    assert!(
        replay
            .current_part(&owner.files, document)
            .unwrap()
            .body(FeatureId(2))
            .is_none()
    );
    set_parameter(&mut owner, "radius", "2 mm");
    let session = owner.files.workspace().session(document).unwrap();
    let actual = replay.current_part(&owner.files, document).unwrap();
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let expected = session
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let actual_body = actual.body(FeatureId(2)).unwrap();
    let expected_body = expected.body(FeatureId(2)).unwrap();
    let keys = actual_body
        .face_reference_keys(session.history(), guid)
        .unwrap();
    assert_eq!(
        keys,
        expected_body
            .face_reference_keys(session.history(), guid)
            .unwrap()
    );
    let saved_keys = keys.clone();
    for key in keys {
        let actual_area = actual_body
            .face_area(session.history(), guid, &key)
            .unwrap();
        let expected_area = expected_body
            .face_area(session.history(), guid, &key)
            .unwrap();
        assert!((actual_area - expected_area).abs() < 1e-10);
    }
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!(replay.current_part(&owner.files, document).is_err());
    replay.ensure_part(&mut owner.files, document).unwrap();
    owner.files.commands(document).unwrap().redo().unwrap();
    assert!(replay.current_part(&owner.files, document).is_err());
    replay.ensure_part(&mut owner.files, document).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    Client::new(owner.document_host())
        .parameters()
        .update(&ParameterUpdateArgs {
            name: "radius".into(),
            comment: Some(Box::new("Fillet control".into())),
            ..Default::default()
        })
        .unwrap();
    assert!(replay.current_part(&owner.files, document).is_ok());
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: "/memory/checkpoint-parameter.opd".into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: "/memory/checkpoint-parameter.opd".into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    replay.ensure_part(&mut owner.files, reopened).unwrap();
    let session = owner.files.workspace().session(reopened).unwrap();
    let reopened_guid = owner
        .files
        .workspace()
        .identity(reopened)
        .unwrap()
        .internal_name;
    let body = replay
        .current_part(&owner.files, reopened)
        .unwrap()
        .body(FeatureId(2))
        .unwrap();
    assert_eq!(
        body.face_reference_keys(session.history(), reopened_guid)
            .unwrap(),
        saved_keys
    );
    for key in saved_keys {
        assert!(
            body.face_area(session.history(), reopened_guid, &key)
                .unwrap()
                > 0.
        );
    }
}

#[test]
fn first_feature_parameter_change_uses_full_replay_and_bad_edit_is_atomic() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "parameter-full-replay.opd");
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let result = Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "not a valid expression !".into(),
            ..Default::default()
        });
    assert!(result.is_err());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
    );
    assert!(replay.current_part(&owner.files, document).is_ok());
    let count = replay.replay_count();
    set_parameter(&mut owner, "width", "5 cm");
    assert!(replay.current_part(&owner.files, document).is_err());
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), count + 1);
}
