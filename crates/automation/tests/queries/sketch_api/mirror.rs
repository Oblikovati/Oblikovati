// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchCopyOperation;
fn mirror_request(axis: u64) -> TransformSketchArgs {
    TransformSketchArgs {
        op: "mirror".into(),
        entities: Some(vec![5]),
        mirror_line: axis,
        ..Default::default()
    }
}
fn mirror_point(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> u64 {
    let ids = move_call(files, source, &mirror_request(3))
        .unwrap()
        .created
        .unwrap();
    assert_eq!(ids.len(), 1);
    ids[0]
}
fn mirrored_point_row(files: &DiscoveryWorkspace, id: u64) -> SketchEntityInfo {
    rows(files).into_iter().find(|row| row.id == id).unwrap()
}
#[test]
fn canonical_mirror_uses_the_owned_axis_in_every_working_scale() {
    for scale in [0.1, 2.0] {
        let (mut files, document, mut source) = configured_scaled(scale);
        install_mixed(&mut files, document);
        let before = rows(&files);
        let copy = mirror_point(&mut files, &mut source);
        assert_eq!(&rows(&files)[..before.len()], before);
        let original = before.iter().find(|row| row.id == 5).unwrap();
        let y = original.points.as_ref().unwrap()[0].as_ref().unwrap()[1];
        assert_eq!(
            mirrored_point_row(&files, copy).points,
            Some(vec![Some(vec![0.0, -y])])
        );
        verify_keys(&files, &rows(&files));
        assert_eq!(
            owner_revision(&files, document).1.last().unwrap(),
            "Mirror sketch entities"
        );
    }
}
#[test]
fn mirror_history_reload_and_abort_keep_geometry_and_never_reuse_ids() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    let first = mirror_point(&mut files, &mut source);
    let copied = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), copied);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), copied);
    begin_move_group(&mut files, document);
    let discarded = mirror_point(&mut files, &mut source);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), copied);
    assert!(mirror_point(&mut files, &mut source) > discarded && discarded > first);
}
#[test]
fn invalid_axis_declines_before_reservation_and_read_only_queries_cannot_mirror() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    let original = rows(&files);
    for axis in [0, 4, 999] {
        assert!(move_call(&mut files, &mut source, &mirror_request(axis)).is_err());
        assert_eq!(rows(&files), original);
    }
    assert_query_move_refused(&files, &mirror_request(3));
    assert_eq!(mirror_point(&mut files, &mut source), 7);
}
#[test]
fn mirror_veto_keeps_source_and_burns_saved_reservations() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let original = rows(&files);
    expect_api_decline(
        move_call(&mut files, &mut source, &mirror_request(3)),
        "locked",
    );
    assert_eq!(rows(&files), original);
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!(mirror_point(&mut files, &mut source) > 7);
}
#[test]
fn mirror_reply_rejection_and_current_axis_edits_use_the_shared_command() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    reject_mirror_reply(&mut files, document);
    move_call(&mut files, &mut source, &movement(vec![3], vec![0.0, 5.0])).unwrap();
    let expected = current_axis_reflection(&files);
    let mirrored = mirror_point(&mut files, &mut source);
    assert!(mirrored > 7);
    let copied = mirrored_point_row(&files, mirrored).points.unwrap();
    for (actual, expected) in copied[0].as_ref().unwrap().iter().zip(expected) {
        assert!((actual - expected).abs() < 1e-12, "{actual} != {expected}");
    }
}
fn reject_mirror_reply(files: &mut DiscoveryWorkspace, document: u64) {
    let sketch = owned_geometry(files, document).sketches()[0].id();
    let original = rows(files);
    let result = files
        .commands(document)
        .unwrap()
        .copy_selected_sketch_entities_projected(
            sketch,
            &[SketchEntityId(5)],
            SketchCopyOperation::Mirror(SketchEntityId(3)),
            |result| {
                assert_eq!(result.unwrap().copies.len(), 1);
                Err::<(), _>("reply rejected")
            },
        );
    assert!(result.is_err());
    assert_eq!(rows(files), original);
}
#[test]
fn assembly_mirror_uses_the_same_source_and_identity_path() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    install_mixed(&mut files, document);
    let copy = mirror_point(&mut files, &mut source);
    assert_eq!(
        mirrored_point_row(&files, copy).points,
        Some(vec![Some(vec![0.0, -20.0])])
    );
}

fn current_axis_reflection(files: &DiscoveryWorkspace) -> [f64; 2] {
    let axis = mirrored_point_row(files, 3).points.unwrap();
    let selected = mirrored_point_row(files, 5).points.unwrap();
    let (a, b, p) = (
        axis[0].as_ref().unwrap(),
        axis[1].as_ref().unwrap(),
        selected[0].as_ref().unwrap(),
    );
    let d = [b[0] - a[0], b[1] - a[1]];
    let t = ((p[0] - a[0]) * d[0] + (p[1] - a[1]) * d[1]) / (d[0] * d[0] + d[1] * d[1]);
    [
        2.0 * (a[0] + t * d[0]) - p[0],
        2.0 * (a[1] + t * d[1]) - p[1],
    ]
}
