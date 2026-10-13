// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_polygon_keeps_regular_constraints_and_undo_restores_the_original_solid() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("polygon.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&original) > 0);
    verify_native_polygon_creation(&mut owner);
    verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 4);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_polygon_creation(owner: &mut NativeDesktopOwner) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let reply = client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "polygon".into(),
            sides: 4,
            points: Some(vec![Some(vec![6.0, 1.0]), Some(vec![8.0, 1.0])]),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(reply.entity_i_ds, Some(vec![109, 110, 111, 112]));
    assert_eq!(reply.point_i_ds, Some(vec![105, 106, 107, 108, 113]));
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    assert_eq!(after.len(), before.len() + 5);
    for row in &after[before.len()..] {
        verify_native_entity_reference(&client, row);
    }
    let circle = after.iter().find(|row| row.id == 114).unwrap();
    assert!(circle.construction);
    assert!((circle.radius - 2.0).abs() < 1e-9);
}
