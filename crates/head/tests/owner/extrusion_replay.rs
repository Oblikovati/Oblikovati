// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn append_candidate_adoption_checks_document_policy_and_transition() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let session = owner.files.workspace().session(document).unwrap();
    let prepared = oblikovati_automation::prepare_extrusion(
        session.snapshot(),
        owner.extrusion_access.as_ref().unwrap(),
        &request(),
    )
    .unwrap();
    let replay = owner
        .files
        .workspace()
        .owned_part_replay(document, Some(policy))
        .unwrap();
    let foreign = Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Other part".into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let ((wrong_document, wrong_policy, accepted, stale), transition) = owner
        .files
        .commands(document)
        .unwrap()
        .run_projected_with_transition(
            "Second extrusion",
            |snapshot, ids| prepared.create(snapshot, ids),
            |candidate, feature, history| {
                let capture = || {
                    replay
                        .prepare(history, candidate, feature)
                        .unwrap()
                        .unwrap()
                        .1
                };
                Ok::<_, oblikovati_api::Error>((capture(), capture(), capture(), capture()))
            },
        )
        .unwrap();
    assert!(
        owner
            .files
            .adopt_owned_part_candidate(foreign, Some(policy), wrong_document, &transition)
            .is_err()
    );
    let mut other = resolution();
    other.weld *= 2.;
    assert!(
        owner
            .files
            .adopt_owned_part_candidate(
                document,
                Some(oblikovati_martensite::PrismResolutionPolicy::Fixed(other)),
                wrong_policy,
                &transition
            )
            .is_err()
    );
    owner
        .files
        .adopt_owned_part_candidate(document, Some(policy), accepted, &transition)
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(
        session
            .owned_part_slot(Some(policy))
            .unwrap()
            .current(session)
            .unwrap()
            .ordered_bodies()
            .count(),
        2
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!(
        owner
            .files
            .adopt_owned_part_candidate(document, Some(policy), stale, &transition)
            .is_err()
    );
}

#[test]
fn certified_extrusion_rejects_foreign_identity_and_policy() {
    use oblikovati_automation::ExtrudeRegionAccess;
    use oblikovati_model::identity::DocumentGuid;
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let mut other_resolution = resolution();
    other_resolution.weld *= 2.;
    let other_policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(other_resolution);
    owner
        .files
        .recompute_owned_part(document, Some(other_policy), &mut 0)
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let foreign: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let replay = workspace.owned_part_replay(document, Some(policy)).unwrap();
    let other = workspace
        .owned_part_replay(document, Some(other_policy))
        .unwrap();
    let backend = owner.extrusion_access.as_ref().unwrap();
    let prepared =
        oblikovati_automation::prepare_extrusion(session.snapshot(), backend, &request()).unwrap();
    session
        .preview_projected(
            |candidate, ids| prepared.create(candidate, ids),
            |candidate, feature| {
                assert!(
                    backend
                        .capture_with_prefix(
                            session.history(),
                            foreign,
                            candidate,
                            feature,
                            &replay
                        )
                        .is_err()
                );
                assert!(
                    backend
                        .capture_with_prefix(session.history(), guid, candidate, feature, &other)
                        .is_err()
                );
                assert!(
                    backend
                        .capture_with_prefix(session.history(), guid, candidate, feature, &replay)
                        .unwrap()
                        .is_some()
                );
                Ok::<_, oblikovati_api::Error>(())
            },
        )
        .unwrap();
}

#[test]
fn appended_preview_matches_full_replay_faces() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let prepared = oblikovati_automation::prepare_extrusion(
        session.snapshot(),
        owner.extrusion_access.as_ref().unwrap(),
        &request(),
    )
    .unwrap();
    session
        .preview_projected(
            |candidate, ids| prepared.create(candidate, ids),
            |candidate, feature| {
                let mut evaluator =
                    oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
                        session.history(),
                        guid,
                        policy,
                    );
                let full = candidate.evaluate_part_features(&mut evaluator).unwrap();
                let expected = full.body(feature).unwrap();
                let keys = expected
                    .face_reference_keys(session.history(), guid)
                    .unwrap();
                assert!(!keys.is_empty());
                let (_, reused) = workspace
                    .inspect_owned_append(document, Some(policy), candidate, feature, |actual| {
                        assert_eq!(
                            actual.face_reference_keys(session.history(), guid).unwrap(),
                            keys
                        );
                        for key in &keys {
                            assert_eq!(
                                actual.face_area(session.history(), guid, key).unwrap(),
                                expected.face_area(session.history(), guid, key).unwrap()
                            );
                        }
                    })
                    .unwrap();
                assert!(reused);
                Ok::<_, oblikovati_api::Error>(())
            },
        )
        .unwrap();
}

#[test]
fn extrusion_commit_publishes_the_certified_document_result_before_display() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let session = owner.files.workspace().session(document).unwrap();
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let result = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 1);
}

#[test]
fn second_extrusion_reuses_the_current_document_prefix() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    let session = owner.files.workspace().session(document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let first = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    let keys = first
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap()
        .face_reference_keys(session.history(), guid)
        .unwrap();
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let session = owner.files.workspace().session(document).unwrap();
    let result = session
        .owned_part_slot(Some(policy))
        .unwrap()
        .current(session)
        .unwrap();
    assert_eq!(result.ordered_bodies().count(), 2);
    assert_eq!(
        result
            .body(oblikovati_model::feature::FeatureId(1))
            .unwrap()
            .face_reference_keys(session.history(), guid)
            .unwrap(),
        keys
    );
}

#[test]
fn repeated_extrusion_previews_reuse_the_current_part_and_cancel_cleanly() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let initial = owner.part_scenes.extrusion_prefix_previews();
    let mut preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "1 cm".into(),
    };
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    preview.distance = "2 cm".into();
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert_eq!(owner.part_scenes.extrusion_prefix_previews(), initial + 2);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(owner.extrusion_preview_from_gui(None).unwrap());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[test]
fn extrusion_preview_does_not_reuse_a_different_policy() {
    use oblikovati_automation::PartUpdateAccess;
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    assert!(
        owner
            .extrude_from_gui(document, crate::scene_fixture::SKETCH, 0, "width")
            .unwrap()
    );
    let mut alternate = resolution();
    alternate.weld *= 2.;
    let mut replay = oblikovati_automation::PartReplayService::with_policy(
        oblikovati_martensite::PrismResolutionPolicy::Fixed(alternate),
    )
    .unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    for factor in [3., 4., 5.] {
        let mut other = resolution();
        other.weld *= factor;
        let mut other_replay = oblikovati_automation::PartReplayService::with_policy(
            oblikovati_martensite::PrismResolutionPolicy::Fixed(other),
        )
        .unwrap();
        other_replay
            .ensure_part(&mut owner.files, document)
            .unwrap();
    }
    let original_policy = oblikovati_martensite::PrismResolutionPolicy::Fixed(resolution());
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .owned_part_prefix(Some(original_policy))
            .is_none()
    );
    let before = owner.part_scenes.extrusion_prefix_previews();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let preview = oblikovati_gui::GuiExtrudePreview {
        generation: Some(generation),
        document,
        sketch: crate::scene_fixture::SKETCH,
        profile: 0,
        distance: "1 cm".into(),
    };
    assert!(owner.extrusion_preview_from_gui(Some(&preview)).unwrap());
    assert_eq!(owner.part_scenes.extrusion_prefix_previews(), before);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(replay.current_part(&owner.files, document).is_ok());
    assert!(owner.extrusion_preview_from_gui(None).unwrap());
}
