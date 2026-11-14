// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{MeasureArgs, MeasureResult};

fn measure(
    owner: &mut TestOwner,
    index: i64,
    key: &str,
    kind: &str,
) -> Result<MeasureResult, oblikovati_api::Error> {
    Client::new(owner.document_host())
        .analysis()
        .measure(&MeasureArgs {
            body_index: index,
            type_: kind.into(),
            key_a: key.into(),
            ..Default::default()
        })
}
fn faces(owner: &mut TestOwner, body: usize) -> Vec<String> {
    Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()[body]
        .faces
        .as_ref()
        .unwrap()
        .iter()
        .map(|face| face.key.clone())
        .collect()
}
fn areas(owner: &mut TestOwner, keys: &[String]) -> Vec<i64> {
    let mut values: Vec<_> = keys
        .iter()
        .map(|key| {
            let result = measure(owner, 0, key, "area").unwrap();
            assert_eq!(
                (result.type_.as_str(), result.unit.as_str()),
                ("area", "mm²")
            );
            (result.value * 1_000_000.).round() as i64
        })
        .collect();
    values.sort_unstable();
    values
}
#[test]
fn named_face_area_tracks_parameter_edit_and_reopen() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let keys = faces(&mut owner, 0);
    assert_eq!(
        areas(&mut owner, &keys),
        vec![
            600_000_000,
            600_000_000,
            800_000_000,
            800_000_000,
            1_200_000_000,
            1_200_000_000
        ]
    );
    set_parameter(&mut owner, "depth", "3 cm");
    let edited = areas(&mut owner, &keys);
    assert_eq!(
        edited,
        vec![
            900_000_000,
            900_000_000,
            1_200_000_000,
            1_200_000_000,
            1_200_000_000,
            1_200_000_000
        ]
    );
    let path = "/memory/face-area.opd";
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
    assert_eq!(areas(&mut owner, &keys), edited);
}
#[test]
fn face_area_rejects_other_bodies_bad_keys_and_unsupported_types() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    open_prism(&mut owner, true);
    let first = faces(&mut owner, 0);
    let second = faces(&mut owner, 1);
    assert!(measure(&mut owner, 0, &second[0], "area").is_err());
    assert!(measure(&mut owner, -1, &first[0], "area").is_err());
    assert!(measure(&mut owner, 2, &first[0], "area").is_err());
    assert!(measure(&mut owner, 0, "invalid", "area").is_err());
    for kind in [
        "length",
        "distance",
        "minDistance",
        "angle",
        "loopLength",
        "other",
    ] {
        assert!(
            measure(&mut owner, 0, &first[0], kind)
                .unwrap_err()
                .to_string()
                .contains("measurement type")
        );
    }
}

#[test]
fn deleted_body_face_is_refused_and_undo_restores_its_area() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let face = faces(&mut owner, 0).remove(0);
    let expected = measure(&mut owner, 0, &face, "area").unwrap();
    Client::new(owner.document_host())
        .body()
        .delete(&BodyIndexArgs { body_index: 0 })
        .unwrap();
    assert!(measure(&mut owner, 0, &face, "area").is_err());
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(measure(&mut owner, 0, &face, "area").unwrap(), expected);
}
