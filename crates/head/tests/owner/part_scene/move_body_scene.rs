// SPDX-License-Identifier: GPL-2.0-only
//! Typed body motion through the desktop replay owner (#4097).
use super::*;
use oblikovati_api::wire::{
    MassPropertiesArgs,
    featureargs::{MoveBody, MoveBodyOp},
};

fn centroid_x(owner: &mut TestOwner) -> f64 {
    Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs::default())
        .unwrap()
        .centroid_x_mm
}
fn expect_centroid(owner: &mut TestOwner, expected: f64) {
    let actual = centroid_x(owner);
    assert!(
        (actual - expected).abs() < 1e-6,
        "centroid {actual}: expected {expected}"
    );
}
#[test]
fn typed_move_recomputes_undoes_redoes_and_reopens() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "offset".into(),
            expression: "10 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let face = super::hole_scene::face_key(&owner, document);
    let input = MoveBody {
        operations: Some(vec![MoveBodyOp {
            type_: "freeDrag".into(),
            x: "offset".into(),
            ..Default::default()
        }]),
        ..Default::default()
    };
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "move-body");
    expect_centroid(&mut owner, 30.);
    set_parameter(&mut owner, "offset", "20 mm");
    expect_centroid(&mut owner, 40.);
    owner.files.commands(document).unwrap().undo().unwrap();
    expect_centroid(&mut owner, 30.);
    owner.files.commands(document).unwrap().redo().unwrap();
    expect_centroid(&mut owner, 40.);
    super::cutoff_checkpoint::save_close_reopen(&mut owner, document);
    expect_centroid(&mut owner, 40.);
    assert!((super::hole_scene::face_area(&mut owner, &face) - 1200.).abs() < 1e-5);
}
#[test]
fn invalid_motion_does_not_change_the_document() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for input in [
        MoveBody {
            body_index: -1,
            translation: Some(vec![1., 0., 0.]),
            ..Default::default()
        },
        MoveBody {
            body_index: 1,
            translation: Some(vec![1., 0., 0.]),
            ..Default::default()
        },
        MoveBody {
            translation: Some(vec![1., 0.]),
            ..Default::default()
        },
        MoveBody {
            operations: Some(vec![MoveBodyOp {
                type_: "alongRay".into(),
                dir: Some(vec![0., 0., 0.]),
                dist: "1 mm".into(),
                ..Default::default()
            }]),
            ..Default::default()
        },
    ] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .add_feature(&input)
                .is_err(),
            "accepted {input:?}"
        );
    }
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
    );
    expect_centroid(&mut owner, 20.);
}
