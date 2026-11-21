// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn new_extrusion_is_confined_to_the_active_member() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let variant = Client::new(owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Variant".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: variant })
        .unwrap();
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: 1 })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: variant })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, document, 1.);
}

#[test]
fn model_state_switch_replays_its_own_extrusion_suppression() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    let id = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[0]
        .id()
        .0;
    let variant = Client::new(owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Variant".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id,
            suppressed: true,
        })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: variant })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, document, 1.);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    let path = "/memory/state-suppression.opd";
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
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: 1 })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .model_states()
        .activate(&RepRef { id: variant })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, reopened, 1.);
}

#[test]
fn typed_suppression_replays_scene_and_undo_redo() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    let id = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[0]
        .id()
        .0;
    let args = SetFeatureSuppressedArgs {
        id,
        suppressed: true,
    };
    let reply = Client::new(owner.document_host())
        .features()
        .set_suppressed(&args)
        .unwrap();
    assert_eq!(reply.feature.feature_info.id, id);
    assert_eq!(reply.feature.feature_info.kind, "extrude");
    assert!(reply.feature.feature_info.suppressed);
    assert!(reply.feature.feature_info.health.is_empty());
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    let reply = Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id,
            suppressed: false,
        })
        .unwrap();
    assert!(!reply.feature.feature_info.suppressed);
    assert!(reply.feature.feature_info.health.is_empty());
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, document, 1.);
}

#[test]
fn unknown_feature_is_rejected_without_changing_the_scene() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    open_sketch(&mut owner);
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    let error = Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id: u64::MAX,
            suppressed: true,
        })
        .unwrap_err();
    assert!(error.to_string().contains("18446744073709551615"));
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[test]
fn suppression_requires_replay_and_persists_after_reopen() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    let id = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[0]
        .id()
        .0;
    let args = SetFeatureSuppressedArgs {
        id,
        suppressed: true,
    };
    let error = Client::new(
        owner
            .document_host()
            .with_extrusion(&RejectExtrusionCandidate),
    )
    .features()
    .set_suppressed(&args)
    .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected a feature replay service")
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    Client::new(owner.document_host())
        .features()
        .set_suppressed(&args)
        .unwrap();
    let path = "/memory/suppressed-extrusion.opd";
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
    let reopened = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert!(
        owner
            .documents()
            .workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .feature_suppressed(oblikovati_model::feature::FeatureId(id))
            .unwrap()
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        0
    );
    Client::new(owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id,
            suppressed: false,
        })
        .unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_cube(&owner, reopened, 1.);
}
