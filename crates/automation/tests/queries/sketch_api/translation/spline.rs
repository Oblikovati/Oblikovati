// SPDX-License-Identifier: GPL-2.0-only
use super::super::spline_requests::{fit_request, offset_request};
use super::*;
use oblikovati_model::sketch::SketchEntityId;

fn create_family(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> [u64; 2] {
    let client = Client::new(FileHost::new(files, source, "test"));
    client.sketch().create(&Default::default()).unwrap();
    let fit = client
        .sketch()
        .add_entity(&fit_request("sweet", false))
        .unwrap();
    let offset = client
        .sketch()
        .add_entity(&offset_request(fit.entity_id, "-5 mm"))
        .unwrap();
    [fit.entity_id, offset.entity_id]
}

fn entity_row(files: &DiscoveryWorkspace, id: u64) -> SketchEntityInfo {
    rows(files).into_iter().find(|row| row.id == id).unwrap()
}

fn assert_offset_parent(files: &DiscoveryWorkspace, document: u64, ids: [u64; 2]) {
    let sketch = owned_geometry(files, document).sketches()[0].geometry();
    let offset = sketch.offset_spline(SketchEntityId(ids[1])).unwrap();
    assert_eq!(offset.parent(), SketchEntityId(ids[0]));
    assert_eq!(offset.distance(), -0.5);
}

#[test]
fn public_move_translates_fit_points_and_retains_derived_offset_identity() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_spline_move(owner);
    }
}

fn verify_spline_move(owner: DocumentType) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    let ids = create_family(&mut files, &mut source);
    let before = entity_row(&files, ids[0]);
    let offset = entity_row(&files, ids[1]);
    move_call(
        &mut files,
        &mut source,
        &movement(vec![ids[0]], vec![1., 2.]),
    )
    .unwrap();
    assert_translated(
        std::slice::from_ref(&before),
        &[entity_row(&files, ids[0])],
        [1., 2.],
    );
    assert_eq!(entity_row(&files, ids[1]), offset);
    assert_offset_parent(&files, document, ids);
    verify_keys(&files, &[before, offset]);
}

#[test]
fn public_copy_preserves_fit_intent_without_copying_the_derived_offset() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let ids = create_family(&mut files, &mut source);
    let before = entity_row(&files, ids[0]);
    let original_count = rows(&files).len();
    let copy = copy_fit(&mut files, &mut source, ids[0]);
    let copied = entity_row(&files, copy);
    assert_copied_fit(&files, &before, &copied, original_count);
    assert_offset_parent(&files, document, ids);
    verify_keys(&files, &[before, copied]);
}

fn assert_copied_fit(
    files: &DiscoveryWorkspace,
    before: &SketchEntityInfo,
    copied: &SketchEntityInfo,
    original_count: usize,
) {
    assert_eq!(copied.kind, before.kind);
    assert_eq!(copied.fit_method, "sweet");
    assert_copy_points(before, copied);
    let after = rows(files);
    assert_eq!(after.len(), original_count + 1);
    assert_eq!(
        after
            .iter()
            .filter(|row| row.kind == "offsetSpline")
            .count(),
        1
    );
}

fn assert_copy_points(before: &SketchEntityInfo, copied: &SketchEntityInfo) {
    let mut expected = before.points.clone().unwrap();
    for point in &mut expected {
        let coordinates = point.as_mut().unwrap();
        coordinates[0] += 1.;
        coordinates[1] += 2.;
    }
    assert_eq!(copied.points, Some(expected));
}

#[test]
fn public_rotate_preserves_offset_source_through_history_and_reopen() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let ids = create_family(&mut files, &mut source);
    let before = entity_row(&files, ids[0]);
    let request = rotation(ids[0]);
    move_call(&mut files, &mut source, &request).unwrap();
    let rotated = entity_row(&files, ids[0]);
    assert_rotated_points(&before, &rotated);
    assert_offset_parent(&files, document, ids);
    verify_rotation_history(&mut files, document, &mut source, ids, before, rotated);
}

fn rotation(id: u64) -> TransformSketchArgs {
    TransformSketchArgs {
        op: "rotate".into(),
        entities: Some(vec![id]),
        center: Some(vec![0., 0.]),
        angle: "90 deg".into(),
        ..Default::default()
    }
}

fn assert_rotated_points(before: &SketchEntityInfo, rotated: &SketchEntityInfo) {
    let original = before.points.as_ref().unwrap();
    let changed = rotated.points.as_ref().unwrap();
    for (source, target) in original.iter().zip(changed) {
        let [x, y] = source.as_ref().unwrap().as_slice() else {
            panic!("source point: expected XY")
        };
        let [rx, ry] = target.as_ref().unwrap().as_slice() else {
            panic!("rotated point: expected XY")
        };
        assert!((rx + y).abs() < 1e-10 && (ry - x).abs() < 1e-10);
    }
}

fn verify_rotation_history(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    ids: [u64; 2],
    before: SketchEntityInfo,
    rotated: SketchEntityInfo,
) {
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(entity_row(files, ids[0]), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(entity_row(files, ids[0]), rotated);
    let reopened = reopen_with_observer(files, document, source, Default::default());
    assert_eq!(entity_row(files, ids[0]), rotated);
    assert_offset_parent(files, reopened, ids);
}

fn copy_fit(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, id: u64) -> u64 {
    let mut request = movement(vec![id], vec![1., 2.]);
    request.op = "copy".into();
    let created = move_call(files, source, &request).unwrap().created.unwrap();
    let [copy] = created.as_slice() else {
        panic!("copy reply: expected one fit spline identity");
    };
    *copy
}
