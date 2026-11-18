// SPDX-License-Identifier: GPL-2.0-only
//! Edited blends remain resolvable through later body replacements.
use super::*;

fn edge_key(guid: DocumentGuid, role: &str) -> String {
    SemanticReferenceName::new(guid, 1, 2, "sketch-prism", role, &[1, 101])
        .unwrap()
        .encode()
        .unwrap()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}
fn edit_radius(id: u64, value: &str) -> oblikovati_api::wire::EditFeatureArgs {
    oblikovati_api::wire::EditFeatureArgs {
        id,
        scalars: Some(vec![oblikovati_api::wire::ScalarEdit {
            index: 0,
            value: value.into(),
        }]),
        ..Default::default()
    }
}
#[test]
fn editing_a_fillet_consumed_by_a_chamfer_preserves_the_final_body() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let id = Client::new(owner.document_host())
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(vec![edge_key(guid, "start-edge-curve")]),
            radius: "1 mm".into(),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap();
    Client::new(owner.document_host())
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Chamfer {
            edge_refs: Some(vec![edge_key(guid, "end-edge-curve")]),
            distance: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
    Client::new(owner.document_host())
        .features()
        .edit(&edit_radius(id, "1.2 mm"))
        .unwrap();
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    let validation_before = owner.fillet_access.as_ref().unwrap().replay_counts();
    Client::new(owner.document_host())
        .features()
        .edit(&edit_radius(id, "1.5 mm"))
        .unwrap();
    assert_eq!(owner.part_scenes.checkpoint_preview_metrics().0, before);
    assert_eq!(
        owner.fillet_access.as_ref().unwrap().replay_counts(),
        validation_before
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    let session = owner.files.workspace().session(document).unwrap();
    let body = owner
        .part_scenes
        .current(session)
        .unwrap()
        .body_from_source(FeatureId(id))
        .unwrap();
    assert!(
        !body
            .face_reference_keys(session.history(), guid)
            .unwrap()
            .is_empty()
    );
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let full = session
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let expected = full.body_from_source(FeatureId(id)).unwrap();
    let faces = body.face_reference_keys(session.history(), guid).unwrap();
    assert_eq!(
        faces,
        expected
            .face_reference_keys(session.history(), guid)
            .unwrap()
    );
    for face in faces {
        assert!(
            (body.face_area(session.history(), guid, &face).unwrap()
                - expected.face_area(session.history(), guid, &face).unwrap())
            .abs()
                < 1e-10
        );
    }
}

#[test]
fn associative_sketch_forces_ordered_recompute_after_blend_edit() {
    use oblikovati_model::{
        document_sketch::SketchOrientationDefinition, feature::work_axis::WorkAxisReference,
        sketch::SketchFrameAxis,
    };
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let id = Client::new(owner.document_host())
        .features()
        .add_feature(&oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(vec![edge_key(guid, "start-edge-curve")]),
            radius: "1 mm".into(),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap()["id"]
        .as_u64()
        .unwrap();
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Orient sketch", |snapshot, _| {
            snapshot.sketches.set_orientation(
                oblikovati_model::document_sketch::DocumentSketchId(1),
                Some(SketchOrientationDefinition {
                    reference: WorkAxisReference::OriginX,
                    sketch_axis: SketchFrameAxis::X,
                    reverse: false,
                    origin: None,
                }),
            )
        })
        .unwrap();
    owner.take_scene().unwrap().unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let draft = oblikovati_gui::GuiFilletEditRequest {
        document,
        generation,
        feature: FeatureId(id),
        radius: "1.5 mm".into(),
    };
    let preview_before = owner.part_scenes.checkpoint_preview_metrics();
    assert!(owner.preview_fillet_edit(&draft).unwrap());
    assert_eq!(
        owner.part_scenes.checkpoint_preview_metrics(),
        preview_before
    );
    assert!(owner.preview_fillet_from_gui(None).unwrap());
    owner
        .part_scenes
        .ensure_edit_checkpoints(&mut owner.files, document)
        .unwrap();
    let before = owner.part_scenes.checkpoint_preview_metrics().0;
    let validation_before = owner.fillet_access.as_ref().unwrap().replay_counts();
    Client::new(owner.document_host())
        .features()
        .edit(&edit_radius(id, "1.5 mm"))
        .unwrap();
    assert!(owner.part_scenes.checkpoint_preview_metrics().0 > before);
    assert_eq!(
        owner.fillet_access.as_ref().unwrap().replay_counts().1,
        validation_before.1 + 1
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}
