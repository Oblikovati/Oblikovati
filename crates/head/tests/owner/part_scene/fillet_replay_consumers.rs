// SPDX-License-Identifier: GPL-2.0-only
//! Consumers must replay the same named solid history as the viewport.
use super::*;
use oblikovati_automation::{DrawingViewProjectionAccess, PartUpdateAccess};
use oblikovati_model::drawing::DrawingBaseViewDefinition;

fn fillet_owner() -> (TestOwner, u64) {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let path = std::env::temp_dir().join("head-fillet-consumers.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), super::fillet_scene::fillet_package());
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    set_parameter(&mut owner, "width", "50 mm");
    let scene = owner.take_scene().unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 1, "edited fillet must replay");
    (owner, document)
}

#[test]
fn feature_properties_publishes_the_checkpointed_suppression_tail() {
    let (mut owner, document) = fillet_owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = oblikovati_automation::PartReplayService::with_policy(policy).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let fillet = FeatureId(2);
    let edit = oblikovati_gui::GuiFeatureEdit {
        feature: fillet,
        name: "Conditional fillet".into(),
        always_suppressed: false,
        condition: Some(oblikovati_gui::GuiFeatureCondition {
            parameter: "width".into(),
            comparison: 4,
            value: "40 mm".into(),
        }),
    };
    assert!(owner.edit_feature_from_gui(document, &edit).unwrap());
    assert!(
        replay
            .checkpoint_prefix(&owner.files, document)
            .unwrap()
            .is_some()
    );
    assert_published_suppression(&owner, document, &replay, fillet);

    owner.files.commands(document).unwrap().undo().unwrap();
    owner.take_scene().unwrap().unwrap();
    assert!(
        replay
            .current_part(&owner.files, document)
            .unwrap()
            .body(fillet)
            .is_some()
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    owner.take_scene().unwrap().unwrap();
    assert_published_suppression(&owner, document, &replay, fillet);
}

fn assert_published_suppression(
    owner: &TestOwner,
    document: u64,
    replay: &oblikovati_automation::PartReplayService,
    fillet: FeatureId,
) {
    use oblikovati_model::feature::FeatureEvaluationStatus;
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let current = replay.current_part(&owner.files, document).unwrap();
    assert!(current.body(fillet).is_none());
    assert!(current.health().iter().any(|record| {
        record.feature == fillet && matches!(record.status, FeatureEvaluationStatus::Suppressed)
    }));
    assert_eq!(
        session.snapshot().work_geometry.get(fillet).unwrap().name(),
        "Conditional fillet"
    );
    let full = crate::drawing_fixture_replay::evaluate(session.history(), guid).unwrap();
    let body = current.body(FeatureId(1)).unwrap();
    let expected = full.body(FeatureId(1)).unwrap();
    let keys = body.face_reference_keys(session.history(), guid).unwrap();
    assert_eq!(
        keys,
        expected
            .face_reference_keys(session.history(), guid)
            .unwrap()
    );
    for key in keys {
        let area = body.face_area(session.history(), guid, &key).unwrap();
        let expected = expected.face_area(session.history(), guid, &key).unwrap();
        assert!((area - expected).abs() < 1e-10);
    }
}

#[test]
fn base_view_projects_fillet_cube() {
    let (owner, document) = fillet_owner();
    let session = owner.files.workspace().session(document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let view = DrawingBaseViewDefinition::new(
        "Front",
        oblikovati_api::types::BASE_VIEW_FRONT,
        1.,
        [100., 100.],
    )
    .unwrap();
    let fresh = crate::drawing_fixture_replay::evaluate(session.history(), guid).unwrap();
    let counts = oblikovati_drawing::DrawingProjectionService
        .counts_current(session.history(), guid, &view, &fresh)
        .unwrap();
    let current = owner
        .part_scenes
        .current_part(&owner.files, document)
        .unwrap();
    let cached_counts = oblikovati_drawing::DrawingProjectionService
        .counts_current(session.history(), guid, &view, current)
        .unwrap();
    assert_eq!(cached_counts, counts);
    let cached_lines = oblikovati_drawing::DrawingProjectionService
        .saved_lines_current(session.history(), guid, &view, current)
        .unwrap();
    let fresh_lines = oblikovati_drawing::DrawingProjectionService
        .saved_lines_current(session.history(), guid, &view, &fresh)
        .unwrap();
    assert_eq!(cached_lines, fresh_lines);
    assert!(counts[0] > 0, "fillet cube must project visible curves");
}

#[test]
fn drawing_projection_reuses_current_result_and_rejects_stale_replay() {
    let (mut owner, document) = fillet_owner();
    let cached = owner
        .part_scenes
        .current_part(&owner.files, document)
        .unwrap() as *const _;
    let retained = owner
        .with_drawing_part(document, |part, _, _| Ok(std::ptr::eq(part, cached)))
        .unwrap();
    assert!(retained);
    set_parameter(&mut owner, "width", "60 mm");
    let retained = owner
        .with_drawing_part(document, |part, _, _| Ok(std::ptr::eq(part, cached)))
        .unwrap();
    assert!(!retained);
}

#[test]
fn unsuppressed_fillet_reports_healthy() {
    let (mut owner, _) = fillet_owner();
    for suppressed in [true, false] {
        let reply = Client::new(owner.document_host())
            .features()
            .set_suppressed(&oblikovati_api::wire::SetFeatureSuppressedArgs { id: 2, suppressed })
            .unwrap();
        assert_eq!(reply.feature.feature_info.suppressed, suppressed);
        assert!(
            reply.feature.feature_info.health.is_empty(),
            "{:?}",
            reply.feature.feature_info.health
        );
    }
}

#[test]
fn typed_suppression_publishes_current_ordered_health_and_checkpoints() {
    let (mut owner, document) = fillet_owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = oblikovati_automation::PartReplayService::with_policy(policy).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    for suppressed in [true, false] {
        let detail = Client::new(owner.document_host())
            .features()
            .set_suppressed(&oblikovati_api::wire::SetFeatureSuppressedArgs { id: 2, suppressed })
            .unwrap()
            .feature
            .feature_info;
        assert_eq!(detail.suppressed, suppressed);
        assert!(detail.health.is_empty());
        let current = replay.current_part(&owner.files, document).unwrap();
        assert_eq!(current.body(FeatureId(2)).is_none(), suppressed);
        assert!(
            replay
                .checkpoint_prefix(&owner.files, document)
                .unwrap()
                .is_some()
        );
    }
}

#[test]
fn typed_rename_of_later_feature_reuses_document_checkpoints() {
    let (mut owner, document) = fillet_owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = oblikovati_automation::PartReplayService::with_policy(policy).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let previous_keys = fillet_face_keys(&owner, document, &replay);
    let (full_replays, _) = owner.part_scenes.checkpoint_preview_metrics();

    let renamed = Client::new(owner.document_host())
        .features()
        .rename(&oblikovati_api::wire::RenameFeatureArgs {
            id: 2,
            name: "Finish round".into(),
        })
        .unwrap();

    assert_eq!(renamed.feature.feature_info.name, "Finish round");
    assert!(renamed.feature.feature_info.health.is_empty());
    assert_eq!(
        owner.part_scenes.checkpoint_preview_metrics().0,
        full_replays
    );
    assert!(
        replay
            .checkpoint_prefix(&owner.files, document)
            .unwrap()
            .is_some()
    );
    assert_eq!(fillet_face_keys(&owner, document, &replay), previous_keys);
}

fn fillet_face_keys(
    owner: &TestOwner,
    document: u64,
    replay: &oblikovati_automation::PartReplayService,
) -> Vec<Vec<u8>> {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    replay
        .current_part(&owner.files, document)
        .unwrap()
        .body(FeatureId(2))
        .unwrap()
        .face_reference_keys(session.history(), guid)
        .unwrap()
}

#[test]
fn typed_suppression_declines_checkpoints_from_another_policy() {
    use oblikovati_martensite::PrismResolutionPolicy;
    let (mut owner, document) = fillet_owner();
    let PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size,
        minimum_size,
    } = crate::part_scene::resolution_policy::desktop_policy()
    else {
        panic!("expected a model-relative desktop policy");
    };
    let alternate = PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size: reference_size * 2.,
        minimum_size,
    };
    let mut replay = oblikovati_automation::PartReplayService::with_policy(alternate).unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let reply = Client::new(owner.document_host())
        .features()
        .set_suppressed(&oblikovati_api::wire::SetFeatureSuppressedArgs {
            id: 2,
            suppressed: true,
        })
        .unwrap();
    assert!(reply.feature.feature_info.suppressed);
    assert!(reply.feature.feature_info.health.is_empty());
    assert!(replay.current_part(&owner.files, document).is_err());
}

#[test]
fn extrusion_preview_replays_existing_fillet() {
    let (owner, document) = fillet_owner();
    let request = oblikovati_api::wire::featureargs::Extrude {
        distance: "10 mm".into(),
        operation: "new".into(),
        ..Default::default()
    };
    let preview = owner
        .preview_extrusion_from_gui(document, &request)
        .unwrap();
    assert!(!preview.faces.is_empty());
}

#[test]
fn fillet_health_replay_accepts_later_model_state_identities() {
    let (mut owner, document) = fillet_owner();
    let mut last = 0;
    for index in 0..4 {
        last = Client::new(owner.document_host())
            .model_states()
            .create(&CreateModelStateArgs {
                name: format!("Variant {index}"),
                ..Default::default()
            })
            .unwrap()
            .model_state
            .id;
    }
    let params = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .ids();
    assert!(params.iter().all(|id| *id < last));
    let reply = Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id: 2,
            suppressed: false,
        })
        .unwrap();
    assert!(reply.feature.feature_info.health.is_empty());
}
