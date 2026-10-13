// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchReadinessCause,
    feature::{FeatureEvaluationStatus, part::PartFeatureFailure, sketch_prism::SketchPrismCause},
    history::DocumentSketchPlacementError,
};
#[test]
fn native_copy_publishes_references_and_reports_free_profile_points() {
    verify_native_copy_case("copy", translated_native_point);
}
#[test]
fn native_mirror_publishes_reflected_keys_and_reports_free_profile_points() {
    verify_native_copy_case("mirror", reflected_native_point);
}
fn translated_native_point(point: &[f64]) -> [f64; 2] {
    [point[0] + 1.0, point[1] + 2.0]
}
fn reflected_native_point(point: &[f64]) -> [f64; 2] {
    [point[0], -point[1]]
}
fn verify_native_copy_case(op: &str, expected: fn(&[f64]) -> [f64; 2]) {
    let directory = tempfile::tempdir().unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("copy.opd"));
    let before = publish_native_scene(&mut owner, &mut window);
    assert!(solid_pixels(&before) > 0);
    verify_native_profile_copy(&mut owner, op, expected);
    verify_copy_readiness_and_undo(&mut owner, &mut window, &before);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_native_profile_copy(
    owner: &mut NativeDesktopOwner,
    op: &str,
    expected: fn(&[f64]) -> [f64; 2],
) {
    let client = Client::new(owner.document_host());
    let before = native_sketch_entities(&client);
    let created = copy_native_profile(&client, op);
    assert_eq!(created.len(), 4);
    let after = native_sketch_entities(&client);
    assert_eq!(&after[..before.len()], before);
    assert_eq!(after.len(), before.len() + 4);
    for (source, copy) in (101..105).zip(created) {
        let original = before.iter().find(|row| row.id == source).unwrap();
        let copied = after.iter().find(|row| row.id == copy).unwrap();
        verify_native_copied_line(&client, original, copied, expected);
    }
}
fn native_sketch_entities<T: oblikovati_api::client::Caller>(
    client: &Client<T>,
) -> Vec<SketchEntityInfo> {
    client
        .sketch()
        .entities(&SketchArgs::default())
        .unwrap()
        .entities
        .unwrap()
}
fn copy_native_profile<T: oblikovati_api::client::Caller>(
    client: &Client<T>,
    op: &str,
) -> Vec<u64> {
    client
        .sketch()
        .transform(&TransformSketchArgs {
            op: op.into(),
            mirror_line: 101,
            entities: Some((101..105).collect()),
            vector: Some(vec![1.0, 2.0]),
            ..Default::default()
        })
        .unwrap()
        .created
        .unwrap()
}
fn verify_native_copied_line<T: oblikovati_api::client::Caller>(
    client: &Client<T>,
    original: &SketchEntityInfo,
    copied: &SketchEntityInfo,
    expected: fn(&[f64]) -> [f64; 2],
) {
    assert_ne!(original.reference_key, copied.reference_key);
    verify_native_copy_coordinates(original, copied, expected);
    let resolved = client
        .sketch()
        .resolve_reference(&ResolveSketchReferenceArgs {
            reference_key: copied.reference_key.clone(),
        })
        .unwrap();
    assert!(resolved.found);
    assert_eq!(resolved.entity_id, copied.id);
}
fn verify_native_copy_coordinates(
    original: &SketchEntityInfo,
    copied: &SketchEntityInfo,
    expected: fn(&[f64]) -> [f64; 2],
) {
    let original_points = original.points.as_ref().unwrap();
    let copied_points = copied.points.as_ref().unwrap();
    assert_eq!(copied_points.len(), original_points.len());
    for (old, new) in original_points.iter().zip(copied_points) {
        let old = old.as_ref().unwrap();
        assert_eq!(new.as_ref().unwrap(), &expected(old));
    }
}

fn verify_copy_readiness_and_undo(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    before: &ViewportCapture,
) {
    let after = publish_native_scene(owner, window);
    assert_eq!(solid_pixels(&after), 0);
    verify_copy_free_variables(window.scene_status().unwrap());
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(publish_native_scene(owner, window).rgba == before.rgba);
}
fn verify_copy_free_variables(status: &crate::HeadSceneStatus) {
    let FeatureEvaluationStatus::Failed(PartFeatureFailure::Inputs(cause)) =
        &status.health[0].status
    else {
        panic!(
            "copied free points: expected input decline, got {:?}",
            status.health
        );
    };
    assert_eq!(cause.sketch, fixture::SKETCH);
    let SketchPrismCause::Placement(DocumentSketchPlacementError::Verification(readiness)) =
        cause.cause.as_ref()
    else {
        panic!("copied free points: expected readiness decline, got {cause:?}");
    };
    assert_eq!(readiness.cause, SketchReadinessCause::FreeVariables(8));
    assert!(status.title().contains("1 unavailable features"));
}
