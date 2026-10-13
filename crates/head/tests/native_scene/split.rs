// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{
    native_sketch_entities, verify_copy_readiness_and_undo, verify_native_entity_reference,
};
use super::*;
#[test]
fn native_split_keeps_the_source_key_and_reports_the_new_free_junction() {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("split.opd"));
    let original = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&original) > 0);
    verify_native_split_geometry(&mut owner);
    verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 2);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_split_geometry(owner: &mut NativeDesktopOwner) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let request = native_split_request();
    let parts = client
        .sketch()
        .transform(&request)
        .unwrap()
        .created
        .unwrap();
    assert_eq!(parts, [101, 106]);
    let after = native_sketch_entities(&client);
    assert_eq!(after.len(), before.len() + 1);
    verify_native_split_parts(&before, &after);
    for id in parts {
        verify_native_entity_reference(&client, after.iter().find(|row| row.id == id).unwrap());
    }
}
fn native_split_request() -> TransformSketchArgs {
    TransformSketchArgs {
        op: "split".into(),
        entities: Some(vec![101]),
        vector: Some(vec![2.0, 1.0]),
        ..Default::default()
    }
}
fn verify_native_split_parts(before: &[SketchEntityInfo], after: &[SketchEntityInfo]) {
    let original = before.iter().find(|row| row.id == 101).unwrap();
    let first = after.iter().find(|row| row.id == 101).unwrap();
    let tail = after.iter().find(|row| row.id == 106).unwrap();
    assert_eq!(first.reference_key, original.reference_key);
    assert_ne!(first.reference_key, tail.reference_key);
    assert_eq!(
        first.points,
        Some(vec![Some(vec![0.0, 0.0]), Some(vec![2.0, 0.0])])
    );
    assert_eq!(
        tail.points,
        Some(vec![Some(vec![2.0, 0.0]), Some(vec![4.0, 0.0])])
    );
    verify_native_unsplit_rows(before, after);
}
fn verify_native_unsplit_rows(before: &[SketchEntityInfo], after: &[SketchEntityInfo]) {
    for row in before.iter().filter(|row| row.id != 101) {
        assert_eq!(
            after.iter().find(|current| current.id == row.id).unwrap(),
            row
        );
    }
}
