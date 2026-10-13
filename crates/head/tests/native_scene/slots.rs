// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_slot_variants_use_owned_boundaries_and_undo_restores_the_visible_solid() {
    for variant in [
        "",
        "overall",
        "slotCenter",
        "arc",
        "arcCenterPoint",
        "arcThreePoint",
    ] {
        let directory = tempfile::tempdir().unwrap();
        let mut window = NativeHeadWindow::open().unwrap();
        let mut owner = native_owner(window.wake.clone()).unwrap();
        open_native_part(&mut owner, &directory.path().join("slot.opd"));
        let before = publish_native_scene(&mut owner, &mut window);
        assert!(solid_pixels(&before) > 0);
        verify_native_slot(&mut owner, variant);
        verify_copy_readiness_and_undo(
            &mut owner,
            &mut window,
            &before,
            if variant.starts_with("arc") { 12 } else { 10 },
        );
        owner.finish().unwrap();
        window.finish().unwrap();
    }
}
fn verify_native_slot(owner: &mut NativeDesktopOwner, variant: &str) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let points = match variant {
        "arc" => vec![vec![0.0, 0.0], vec![4.0, 0.0], vec![0.0, 4.0]],
        "arcThreePoint" => vec![vec![4.0, 0.0], vec![0.0, 4.0], vec![-4.0, 0.0]],
        _ => vec![vec![0.0, 0.0], vec![4.0, 0.0]],
    };
    let reply = client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "slot".into(),
            variant: variant.into(),
            points: Some(points.into_iter().map(Some).collect()),
            width: "2 cm".into(),
            end_angle: "90 deg".into(),
            ccw: true,
            ..Default::default()
        })
        .unwrap();
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    assert_eq!(after.len(), before.len() + 4);
    let first = if variant.starts_with("arc") { 113 } else { 111 };
    assert_eq!(reply.entity_i_ds, Some((first..first + 4).collect()));
    for row in &after[before.len()..] {
        verify_native_entity_reference(&client, row);
    }
}
