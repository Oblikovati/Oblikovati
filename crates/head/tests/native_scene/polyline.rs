// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_polyline_creation_retains_keys_reports_free_vertices_and_undo_restores_image() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("polyline.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&original) > 0);
    verify_native_polyline(&mut owner);
    verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 6);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_polyline(owner: &mut NativeDesktopOwner) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let report = client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "polyline".into(),
            closed: true,
            points: Some(vec![
                Some(vec![5.0, 0.0]),
                Some(vec![7.0, 0.0]),
                Some(vec![7.0, 2.0]),
            ]),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(report.point_i_ds, Some(vec![105, 106, 107]));
    assert_eq!(report.entity_i_ds, Some(vec![108, 109, 110]));
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    assert_eq!(after.len(), before.len() + 3);
    verify_native_polyline_lines(&client, &after);
}
fn verify_native_polyline_lines<T: oblikovati_api::client::Caller>(
    client: &Client<T>,
    rows: &[SketchEntityInfo],
) {
    let points = [[5.0, 0.0], [7.0, 0.0], [7.0, 2.0]];
    for index in 0..3 {
        let row = rows
            .iter()
            .find(|row| row.id == 108 + index as u64)
            .unwrap();
        assert_eq!(
            row.points,
            Some(vec![
                Some(points[index].to_vec()),
                Some(points[(index + 1) % 3].to_vec())
            ])
        );
        verify_native_entity_reference(client, row);
    }
}
