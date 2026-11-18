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
    let counts = oblikovati_drawing::DrawingProjectionService
        .counts(session.history(), guid, &view)
        .unwrap();
    let current = owner
        .part_scenes
        .current_part(&owner.files, document)
        .unwrap();
    let cached_counts = oblikovati_drawing::DrawingProjectionService
        .counts_current(session.history(), guid, &view, current)
        .unwrap();
    assert_eq!(cached_counts, counts);
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
