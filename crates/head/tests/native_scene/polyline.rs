// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_polyline_creation_retains_keys_reports_free_vertices_and_undo_restores_image() {
    verify_native_chain("polyline", &[[5.0, 0.0], [7.0, 0.0], [7.0, 2.0]]);
}
#[test]
fn native_rectangle_creation_retains_keys_and_undo_restores_image() {
    verify_native_chain(
        "rectangle",
        &[[5.0, 0.0], [7.0, 0.0], [7.0, 2.0], [5.0, 2.0]],
    );
}
fn verify_native_chain(kind: &str, points: &[[f64; 2]]) {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("polyline.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&original) > 0);
    verify_native_chain_creation(&mut owner, kind, points);
    verify_copy_readiness_and_undo(&mut owner, &mut window, &original, points.len() * 2);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_chain_creation(owner: &mut NativeDesktopOwner, kind: &str, points: &[[f64; 2]]) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let input = if kind == "rectangle" {
        vec![points[0], points[2]]
    } else {
        points.to_vec()
    };
    let report = client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: kind.into(),
            closed: true,
            points: Some(input.iter().map(|point| Some(point.to_vec())).collect()),
            ..Default::default()
        })
        .unwrap();
    let count = points.len() as u64;
    assert_eq!(report.point_i_ds, Some((105..105 + count).collect()));
    assert_eq!(
        report.entity_i_ds,
        Some((105 + count..105 + count * 2).collect())
    );
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    assert_eq!(after.len(), before.len() + points.len());
    verify_native_chain_lines(&client, &after, points);
}
fn verify_native_chain_lines<T: oblikovati_api::client::Caller>(
    client: &Client<T>,
    rows: &[SketchEntityInfo],
    points: &[[f64; 2]],
) {
    for index in 0..points.len() {
        let row = rows
            .iter()
            .find(|row| row.id == 105 + points.len() as u64 + index as u64)
            .unwrap();
        assert_eq!(
            row.points,
            Some(vec![
                Some(points[index].to_vec()),
                Some(points[(index + 1) % points.len()].to_vec())
            ])
        );
        verify_native_entity_reference(client, row);
    }
}
