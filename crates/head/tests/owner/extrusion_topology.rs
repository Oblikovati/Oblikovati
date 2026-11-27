// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{AssignAppearanceArgs, BodyTopology};
use std::collections::BTreeSet;

fn cube_topology(owner: &mut TestOwner, size: f64) -> BodyTopology {
    let response = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let [body] = response.bodies.unwrap().try_into().unwrap();
    let body: BodyTopology = body;
    let groups = [
        body.faces.as_ref().unwrap(),
        body.edges.as_ref().unwrap(),
        body.vertices.as_ref().unwrap(),
    ];
    assert_eq!(groups.map(Vec::len), [6, 12, 8]);
    let mut keys = BTreeSet::new();
    for (index, group) in groups.into_iter().enumerate() {
        for entity in group {
            assert!(keys.insert(entity.key.clone()));
            assert_eq!(entity.kind, if index == 0 { "plane" } else { "" });
            let point = entity.point.as_ref().unwrap();
            assert_eq!(point.len(), 3);
            assert!(
                point
                    .iter()
                    .all(|value| *value >= -1e-8 && *value <= size + 1e-8)
            );
            let boundaries = point
                .iter()
                .filter(|value| value.abs() < 1e-8 || (**value - size).abs() < 1e-8)
                .count();
            assert_eq!(
                boundaries,
                index + 1,
                "unexpected representative point {point:?}"
            );
        }
    }
    body
}
fn keys(body: &BodyTopology) -> BTreeSet<String> {
    body.faces
        .iter()
        .chain(&body.edges)
        .chain(&body.vertices)
        .flatten()
        .map(|entity| entity.key.clone())
        .collect()
}
#[test]
fn typed_topology_keys_survive_edit_undo_redo_and_reopen() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    Client::new(owner.document_host())
        .features()
        .add_feature(&request())
        .unwrap();
    let initial = cube_topology(&mut owner, 1.);
    let expected = keys(&initial);
    set_size(&mut owner, "20 mm");
    assert_eq!(keys(&cube_topology(&mut owner, 2.)), expected);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(keys(&cube_topology(&mut owner, 1.)), expected);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(keys(&cube_topology(&mut owner, 2.)), expected);
    let face = &initial.faces.as_ref().unwrap()[0].key;
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "face".into(),
            key: face.clone(),
            appearance_id: "default".into(),
        })
        .unwrap();
    let path = document_fixture_path!("/memory/topology-cube.opd");
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
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(keys(&cube_topology(&mut owner, 2.)), expected);
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "face".into(),
            key: face.clone(),
            appearance_id: "default".into(),
        })
        .unwrap();
}
