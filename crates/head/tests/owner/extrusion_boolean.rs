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

#[test]
fn overlapping_join_replays_split_face_names_without_rebinding_the_parent() {
    use oblikovati_model::{
        feature::FeatureId,
        identity::{SketchPrismReferenceRole, sketch_prism_reference_name},
    };
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let guid = owner
        .documents()
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let parent = sketch_prism_reference_name(
        guid,
        FeatureId(1),
        crate::scene_fixture::SKETCH,
        SketchPrismReferenceRole::StartCap,
    )
    .unwrap();
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
            kind: "circle".into(),
            points: Some(vec![Some(vec![0.5, 0.5])]),
            radius: "2 mm".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .features()
        .add_feature(&Extrude {
            sketch_index,
            operation: "join".into(),
            distance: "20 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let area = std::f64::consts::PI * 0.04;
    check_boolean_body(&mut owner, document, 1. + area);
    let first = split_cap_keys(&owner, document, &parent);
    set_size(&mut owner, "12 mm");
    check_boolean_body(&mut owner, document, 1.728 + area * 0.8);
    let reopened = reopen_boolean_part(&mut owner, document);
    check_boolean_body(&mut owner, reopened, 1.728 + area * 0.8);
    assert_eq!(split_cap_keys(&owner, reopened, &parent), first);
}

fn split_cap_keys(owner: &TestOwner, document: u64, parent: &[u8]) -> Vec<Vec<u8>> {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let replay = owner.part_scenes.current(session).unwrap();
    let body = replay
        .body_from_source(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let oblikovati_martensite::NamedSolid::Derived(body) = body else {
        panic!("expected joined body")
    };
    let descendants = body.split_keys().get(parent).unwrap();
    assert_eq!(descendants.len(), 2);
    assert!(body.contains(session.history(), guid, parent, 1).is_err());
    assert!(!body.deleted_keys().iter().any(|key| key == parent));
    for key in descendants {
        assert!(body.contains(session.history(), guid, key, 1).unwrap());
    }
    descendants.clone()
}

#[test]
fn rectangular_pattern_replays_joined_bosses_on_the_same_body() {
    use oblikovati_api::wire::featureargs::PatternRectangular;
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_sketch(&mut owner);
    let client = Client::new(owner.document_host());
    client.features().add_feature(&request()).unwrap();
    let plane = client
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec!["origin/plane/xy".into()]),
            offset: "width".into(),
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
            kind: "circle".into(),
            points: Some(vec![Some(vec![0.2, 0.2])]),
            radius: "1 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let join = client
        .features()
        .add_feature(&Extrude {
            sketch_index,
            operation: "join".into(),
            distance: "5 mm".into(),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap();
    let seed = join["name"].as_str().unwrap().to_owned();
    let result = client
        .features()
        .add_feature(&PatternRectangular {
            source_features: Some(vec![seed]),
            count_x: 2,
            count_y: 2,
            step_x: Some(vec![0.56, 0., 0.]),
            step_y: Some(vec![0., 0.56, 0.]),
            ..Default::default()
        })
        .unwrap()
        .0
        .unwrap();
    assert_eq!(result["kind"], "rectangular-pattern");
    assert_eq!(result["bodies"], 1);
    let boss = std::f64::consts::PI * 0.005;
    let primary = check_boolean_body(&mut owner, document, 1. + 4. * boss);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(check_boolean_body(&mut owner, document, 1. + boss), primary);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        check_boolean_body(&mut owner, document, 1. + 4. * boss),
        primary
    );
    set_size(&mut owner, "12 mm");
    let reopened = reopen_boolean_part(&mut owner, document);
    assert_eq!(
        check_boolean_body(&mut owner, reopened, 1.728 + 4. * boss),
        primary
    );
}
