// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn typed_cut_replaces_the_named_body_and_replays_after_reopen() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let client = Client::new(owner.document_host());
    client.features().add_feature(&request()).unwrap();
    let sketch_index = client
        .sketch()
        .create(&Default::default())
        .unwrap()
        .sketch_index;
    client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            sketch_index,
            kind: "rectangle".into(),
            points: Some(vec![Some(vec![0.3, 0.3]), Some(vec![0.7, 0.7])]),
            ..Default::default()
        })
        .unwrap();
    let reply = client
        .features()
        .add_feature(&Extrude {
            sketch_index,
            operation: "cut".into(),
            distance: "15 mm".into(),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap();
    assert_eq!(reply["bodies"], 1);
    let key = check_boolean_body(&mut owner, document, 0.84);
    set_size(&mut owner, "12 mm");
    assert_eq!(check_boolean_body(&mut owner, document, 1.536), key);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(check_boolean_body(&mut owner, document, 0.84), key);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    let reopened = reopen_boolean_part(&mut owner, document);
    assert_eq!(check_boolean_body(&mut owner, reopened, 1.536), key);
}

fn check_boolean_body(owner: &mut TestOwner, document: u64, volume: f64) -> Vec<u8> {
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let replay = owner.part_scenes.current(session).unwrap();
    let body = replay
        .body_from_source(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    assert!(body.as_prism().is_none());
    let key = body.body_reference_key(session.history(), guid).unwrap();
    let properties = body
        .physical_properties(session.history(), guid, key)
        .unwrap();
    assert!(
        (properties.geometry.volume - volume).abs() < 1e-8,
        "{properties:?}"
    );
    key.to_vec()
}

fn reopen_boolean_part(owner: &mut TestOwner, document: u64) -> u64 {
    let path = "/memory/certified-cut.opd";
    let client = Client::new(owner.document_host());
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

#[test]
fn through_all_cut_follows_the_target_after_it_grows() {
    for direction in ["positive", "negative", "symmetric"] {
        let (owner, _) = owner();
        let mut owner = owner
            .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
        let document = open_sketch(&mut owner);
        let client = Client::new(owner.document_host());
        client
            .features()
            .add_feature(&Extrude {
                direction: direction.into(),
                ..request()
            })
            .unwrap();
        let sketch_index = client
            .sketch()
            .create(&Default::default())
            .unwrap()
            .sketch_index;
        client
            .sketch()
            .add_entity(&AddSketchEntityArgs {
                sketch_index,
                kind: "rectangle".into(),
                points: Some(vec![Some(vec![0.3, 0.3]), Some(vec![0.7, 0.7])]),
                ..Default::default()
            })
            .unwrap();
        client
            .features()
            .add_feature(&Extrude {
                sketch_index,
                operation: "cut".into(),
                extent: "through-all".into(),
                direction: direction.into(),
                ..Default::default()
            })
            .unwrap();
        let key = check_boolean_body(&mut owner, document, 0.84);
        set_size(&mut owner, "20 mm");
        assert_eq!(check_boolean_body(&mut owner, document, 7.68), key);
        let reopened = reopen_boolean_part(&mut owner, document);
        assert_eq!(check_boolean_body(&mut owner, reopened, 7.68), key);
    }
}

#[test]
fn join_adds_a_boss_and_preserves_the_target_through_reopen() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let client = Client::new(owner.document_host());
    client.features().add_feature(&request()).unwrap();
    let join = boss_request(&client, "width");
    let reply = client.features().add_feature(&join).unwrap().0.unwrap();
    assert_eq!(reply["bodies"], 1);
    let body = check_boolean_body(&mut owner, document, 1.08);
    set_size(&mut owner, "12 mm");
    assert_eq!(check_boolean_body(&mut owner, document, 1.808), body);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(check_boolean_body(&mut owner, document, 1.08), body);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    let reopened = reopen_boolean_part(&mut owner, document);
    assert_eq!(check_boolean_body(&mut owner, reopened, 1.808), body);
}

fn boss_request(client: &Client<impl oblikovati_api::client::Caller>, offset: &str) -> Extrude {
    let plane = client
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec!["origin/plane/xy".into()]),
            offset: offset.into(),
            ..Default::default()
        })
        .unwrap();
    let sketch_index = client
        .sketch()
        .create(&CreateSketchArgs {
            work_plane_index: Some(Box::new(plane.index)),
            ..Default::default()
        })
        .unwrap()
        .sketch_index;
    client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            sketch_index,
            kind: "rectangle".into(),
            points: Some(vec![Some(vec![0.3, 0.3]), Some(vec![0.7, 0.7])]),
            ..Default::default()
        })
        .unwrap();
    Extrude {
        sketch_index,
        operation: "join".into(),
        distance: "5 mm".into(),
        ..Default::default()
    }
}

#[test]
fn disconnected_join_leaves_the_target_and_tool_sketch_unchanged() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let client = Client::new(owner.document_host());
    client.features().add_feature(&request()).unwrap();
    let join = boss_request(&client, "20 mm");
    let error = client
        .features()
        .add_feature(&join)
        .unwrap_err()
        .to_string();
    assert!(error.contains("expected exactly one"), "{error}");
    drop(client);
    owner.take_scene().unwrap().unwrap();
    assert_cube(&owner, document, 1.);
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(snapshot.work_geometry.features().len(), 2);
    assert!(snapshot.sketches.sketches()[1].visible());
}
