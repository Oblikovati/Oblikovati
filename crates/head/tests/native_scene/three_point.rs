// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_three_point_curves_use_the_kernel_and_restore_the_original_solid_on_undo() {
    for (kind, free) in [("circle", 3), ("arc", 5)] {
        let directory = tempfile::tempdir().unwrap();
        let mut window = NativeHeadWindow::open().unwrap();
        let mut owner = native_owner(window.wake.clone()).unwrap();
        open_native_part(&mut owner, &directory.path().join("three-point.opd"));
        let original = publish_native_scene(&mut owner, &mut window);
        assert!(solid_pixels(&original) > 0);
        verify_native_three_point(&mut owner, kind);
        verify_copy_readiness_and_undo(&mut owner, &mut window, &original, free);
        owner.finish().unwrap();
        window.finish().unwrap();
    }
}
fn verify_native_three_point(owner: &mut NativeDesktopOwner, kind: &str) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let mut request = AddSketchEntityArgs {
        kind: kind.into(),
        variant: "threePoint".into(),
        points: Some(vec![
            Some(vec![0.0, 0.0]),
            Some(vec![1.0, 0.0]),
            Some(vec![2.0, 0.0]),
        ]),
        ..Default::default()
    };
    assert!(client.sketch().add_entity(&request).is_err());
    assert_eq!(native_sketch_entities(&client), before);
    request.points = Some(vec![
        Some(vec![2.0, 0.0]),
        Some(vec![0.0, 2.0]),
        Some(vec![-2.0, 0.0]),
    ]);
    let reply = client.sketch().add_entity(&request).unwrap();
    let (entity, points) = if kind == "circle" {
        (106, vec![105])
    } else {
        (108, vec![105, 106, 107])
    };
    assert_eq!(reply.entity_id, entity);
    assert_eq!(reply.point_i_ds, Some(points));
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    let curve = after.iter().find(|row| row.id == entity).unwrap();
    assert!((curve.radius - 2.0).abs() < 1e-9);
    verify_native_entity_reference(&client, curve);
}
