// SPDX-License-Identifier: GPL-2.0-only
use super::rotation::{created_line_row, free_created_line};
use super::*;
fn split_request(line: u64, pick: [f64; 2]) -> TransformSketchArgs {
    TransformSketchArgs {
        op: "split".into(),
        entities: Some(vec![line]),
        vector: Some(pick.to_vec()),
        ..Default::default()
    }
}
fn split_free_line(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, line: u64) -> u64 {
    let parts = move_call(files, source, &split_request(line, [3.0, 1.0]))
        .unwrap()
        .created
        .unwrap();
    assert_eq!(parts.len(), 2);
    assert_eq!(parts[0], line);
    parts[1]
}
#[test]
fn canonical_split_keeps_the_first_reference_and_creates_a_shared_hidden_junction() {
    let (mut files, document, mut source, line) = free_created_line();
    let original = created_line_row(&files, line);
    let tail = split_free_line(&mut files, &mut source, line);
    assert_eq!(tail, line + 2);
    assert_eq!(rows(&files).len(), 3);
    let first = created_line_row(&files, line);
    assert_eq!(first.reference_key, original.reference_key);
    assert_split_line_coordinates(&first, &created_line_row(&files, tail));
    verify_keys(&files, &rows(&files));
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Split sketch line"
    );
}
fn assert_split_line_coordinates(first: &SketchEntityInfo, tail: &SketchEntityInfo) {
    assert_eq!(
        first.points,
        Some(vec![Some(vec![2.0, 0.0]), Some(vec![3.0, 0.0])])
    );
    assert_eq!(
        tail.points,
        Some(vec![Some(vec![3.0, 0.0]), Some(vec![4.0, 0.0])])
    );
}
#[test]
fn split_undo_redo_reload_and_abort_preserve_geometry_and_consumed_ids() {
    let (mut files, document, mut source, line) = free_created_line();
    let original = rows(&files);
    let tail = split_free_line(&mut files, &mut source, line);
    let split = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), original);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), split);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), split);
    verify_split_group_reservations(&mut files, document, &mut source, tail);
}
fn verify_split_group_reservations(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    tail: u64,
) {
    let before = rows(files);
    begin_move_group(files, document);
    let request = split_request(tail, [3.5, 1.0]);
    let discarded = move_call(files, source, &request).unwrap().created.unwrap()[1];
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(files), before);
    let accepted = move_call(files, source, &request).unwrap().created.unwrap()[1];
    assert!(accepted > discarded);
}
#[test]
fn invalid_splits_validate_the_whole_selection_before_reserving() {
    let (mut files, _, mut source, line) = free_created_line();
    let original = rows(&files);
    let mut invalid_batch = split_request(line, [3.0, 0.0]);
    invalid_batch.entities = Some(vec![line, 999]);
    for request in [
        invalid_batch,
        split_request(1, [3.0, 0.0]),
        split_request(line, [2.0, 9.0]),
        split_request(line, [5.0, 0.0]),
    ] {
        assert!(move_call(&mut files, &mut source, &request).is_err());
        assert_eq!(rows(&files), original);
    }
    assert_query_move_refused(&files, &split_request(line, [3.0, 0.0]));
    assert_eq!(split_free_line(&mut files, &mut source, line), line + 2);
}
#[test]
fn split_veto_retains_original_geometry_and_saves_reserved_ids() {
    let (mut files, document, mut source, line) = free_created_line();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let original = rows(&files);
    expect_api_decline(
        move_call(&mut files, &mut source, &split_request(line, [3.0, 0.0])),
        "locked",
    );
    assert_eq!(rows(&files), original);
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!(split_free_line(&mut files, &mut source, line) > line + 2);
}
#[test]
fn split_reply_rejection_keeps_original_geometry_and_burns_both_ids() {
    let (mut files, document, mut source, line) = free_created_line();
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let before = rows(&files);
    let rejected = files
        .commands(document)
        .unwrap()
        .split_sketch_line_projected(sketch, &[SketchEntityId(line)], [3.0, 1.0], |result| {
            assert_eq!(result.unwrap().parts[0], SketchEntityId(line));
            Err::<(), _>("reply refused")
        });
    assert!(rejected.is_err());
    assert_eq!(rows(&files), before);
    assert!(split_free_line(&mut files, &mut source, line) > line + 2);
}
#[test]
fn split_converts_pick_units_and_only_splits_the_first_valid_selected_entity() {
    for scale in [0.1, 2.0] {
        verify_split_scale(scale);
    }
}
fn verify_split_scale(scale: f64) {
    let (mut files, document, mut source) = configured_scaled(scale);
    install_mixed(&mut files, document);
    let original = created_line_row(&files, 3);
    let end = original.points.unwrap()[1].as_ref().unwrap()[0];
    let circle = created_line_row(&files, 4);
    let mut request = split_request(3, [end / 2.0, 1.0]);
    request.entities = Some(vec![3, 4]);
    let parts = move_call(&mut files, &mut source, &request)
        .unwrap()
        .created
        .unwrap();
    assert_eq!(parts[0], 3);
    assert_eq!(
        created_line_row(&files, 3).points.unwrap()[1],
        Some(vec![end / 2.0, 0.0])
    );
    assert_eq!(created_line_row(&files, 4), circle);
}

#[test]
fn assembly_split_uses_the_same_document_command() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    install_mixed(&mut files, document);
    let parts = move_call(&mut files, &mut source, &split_request(3, [10.0, 0.0]))
        .unwrap()
        .created
        .unwrap();
    assert_eq!(parts, [3, 8]);
    assert_eq!(
        created_line_row(&files, 3).points.unwrap()[1],
        Some(vec![10.0, 0.0])
    );
}
#[test]
fn typed_split_refuses_empty_selections_and_missing_owners_without_changes() {
    let (mut files, document, _, line) = free_created_line();
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let missing = oblikovati_model::document_sketch::DocumentSketchId(u64::MAX);
    let before = rows(&files);
    for (owner, selected) in [(sketch, vec![]), (missing, vec![SketchEntityId(line)])] {
        let result = files
            .commands(document)
            .unwrap()
            .split_sketch_line_projected(owner, &selected, [3.0; 2], |result| result);
        assert!(result.is_err());
        assert_eq!(rows(&files), before);
    }
}
