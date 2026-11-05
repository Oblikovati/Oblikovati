// SPDX-License-Identifier: GPL-2.0-only
use super::rotation::{created_line_row, free_created_line};
use super::*;
use oblikovati_model::{history::EditError, sketch::SketchCopyError};
use oblikovati_session::{CommandError, MutationError};
fn copying(entities: Vec<u64>, vector: Vec<f64>) -> TransformSketchArgs {
    let mut input = movement(entities, vector);
    input.op = "copy".into();
    input
}
fn copy_line(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, line: u64) -> u64 {
    let reply = move_call(files, source, &copying(vec![line], vec![3.0, 4.0])).unwrap();
    let copies = reply.created.unwrap();
    assert_eq!(copies.len(), 1);
    copies[0]
}
fn assert_line_copy(original: &SketchEntityInfo, copy: &SketchEntityInfo, vector: [f64; 2]) {
    assert_ne!(original.id, copy.id);
    assert_ne!(original.reference_key, copy.reference_key);
    let mut same_identity = copy.clone();
    same_identity.id = original.id;
    same_identity.index = original.index;
    same_identity.reference_key = original.reference_key.clone();
    assert_translated(std::slice::from_ref(original), &[same_identity], vector);
}
#[test]
fn canonical_copy_creates_one_line_with_shared_hidden_points_and_keeps_the_source() {
    let (mut files, document, mut source, line) = free_created_line();
    let original = rows(&files);
    let copy = copy_line(&mut files, &mut source, line);
    assert_eq!(&rows(&files)[..original.len()], original);
    assert_eq!(rows(&files).len(), original.len() + 1);
    assert_line_copy(
        &created_line_row(&files, line),
        &created_line_row(&files, copy),
        [3.0, 4.0],
    );
    verify_keys(&files, &rows(&files));
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Copy sketch entities"
    );
}
#[test]
fn copy_undo_redo_and_reload_keep_the_copy_and_burn_undone_ids() {
    let (mut files, document, mut source, line) = free_created_line();
    let copy = copy_line(&mut files, &mut source, line);
    let copied = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files).len(), 2);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), copied);
    files.commands(document).unwrap().undo().unwrap();
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files).len(), 2);
    let next = copy_line(&mut files, &mut source, line);
    assert!(next > copy);
}
#[test]
fn aborted_copy_group_does_not_reuse_its_reserved_identities() {
    let (mut files, document, mut source, line) = free_created_line();
    let original = rows(&files);
    begin_move_group(&mut files, document);
    let discarded = copy_line(&mut files, &mut source, line);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), original);
    let accepted = copy_line(&mut files, &mut source, line);
    assert!(accepted > discarded);
    files.commands(document).unwrap().undo().unwrap();
    let after_undo = copy_line(&mut files, &mut source, line);
    assert!(after_undo > accepted);
}
#[test]
fn rejected_copy_response_discards_geometry_but_preserves_reservations() {
    let (mut files, document, mut source, line) = free_created_line();
    let original = rows(&files);
    let before = owner_revision(&files, document);
    let rejected = rejected_copy(&mut files, document, line);
    assert_eq!(rows(&files), original);
    assert_eq!(owner_revision(&files, document), before);
    let next = copy_line(&mut files, &mut source, line);
    assert!(next > rejected);
}
fn rejected_copy(files: &mut DiscoveryWorkspace, document: u64, line: u64) -> u64 {
    let sketch = owned_geometry(files, document).sketches()[0].id();
    let mut rejected = 0;
    let result = files
        .commands(document)
        .unwrap()
        .copy_sketch_entities_projected(sketch, &[SketchEntityId(line)], [3.0, 4.0], |result| {
            let report = result.unwrap();
            assert_eq!(report.relations.len(), 3);
            rejected = report.copies[0].0;
            Err::<(), _>("reply refused")
        });
    assert!(result.is_err());
    rejected
}
#[test]
fn invalid_copy_batches_and_vectors_keep_owner_state_and_reserve_nothing() {
    let (mut files, document, mut source, line) = free_created_line();
    for input in [
        copying(vec![line, 999], vec![1.0; 2]),
        copying(vec![], vec![1.0; 2]),
        copying(vec![line], vec![]),
    ] {
        let before = rows(&files);
        assert!(move_call(&mut files, &mut source, &input).is_err());
        assert_eq!(rows(&files), before);
    }
    assert_eq!(copy_line(&mut files, &mut source, line), line + 3);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Copy sketch entities"
    );
}
#[test]
fn copied_selection_is_unique_and_vectors_use_document_working_units() {
    for scale in [0.1, 2.0] {
        let (mut files, document, mut source) = configured_scaled(scale);
        install_mixed(&mut files, document);
        let original = created_line_row(&files, 3);
        let input = copying(vec![3, 3], vec![1.0, 2.0]);
        let ids = move_call(&mut files, &mut source, &input)
            .unwrap()
            .created
            .unwrap();
        assert_eq!(ids.len(), 1);
        assert_line_copy(&original, &created_line_row(&files, ids[0]), [1.0, 2.0]);
    }
}
#[test]
fn read_only_queries_and_commit_veto_refuse_copy_without_geometry_changes() {
    let (mut files, document, mut source, line) = free_created_line();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let original = rows(&files);
    let input = copying(vec![line], vec![1.0; 2]);
    expect_api_decline(move_call(&mut files, &mut source, &input), "locked");
    assert_eq!(rows(&files), original);
    assert_query_move_refused(&files, &input);
    assert!(files.workspace().needs_save(document).unwrap());
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert!(copy_line(&mut files, &mut source, line) > line + 3);
}
#[test]
fn saved_copy_reloads_with_its_new_reference_keys_and_geometry() {
    let (mut files, document, mut source, line) = free_created_line();
    copy_line(&mut files, &mut source, line);
    let copied = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), copied);
    verify_keys(&files, &copied);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn assembly_copy_uses_the_same_owner_command_with_a_fresh_circle_center() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    let sketch = create_owner(&mut files, document);
    let circle = create_circle(&mut files, document, sketch);
    let before = rows(&files);
    let input = copying(vec![circle.entity.0], vec![1.0, 2.0]);
    let copies = move_call(&mut files, &mut source, &input)
        .unwrap()
        .created
        .unwrap();
    assert_eq!(copies.len(), 1);
    assert_eq!(&rows(&files)[..before.len()], before);
    assert_line_copy(&before[1], rows(&files).last().unwrap(), [1.0, 2.0]);
}
#[test]
fn direct_copy_command_keeps_typed_missing_sketch_errors_and_complete_relations() {
    let (mut files, document, _, line) = free_created_line();
    assert_missing_copy_owner(&mut files, document, line);
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let report = files
        .commands(document)
        .unwrap()
        .copy_sketch_entities(sketch, &[SketchEntityId(line)], [1.0; 2])
        .unwrap();
    assert_eq!(report.relations.len(), 3);
    assert_eq!(report.copies, [SketchEntityId(line + 3)]);
    assert_line_copy(
        &created_line_row(&files, line),
        &created_line_row(&files, line + 3),
        [1.0; 2],
    );
}
fn assert_missing_copy_owner(files: &mut DiscoveryWorkspace, document: u64, line: u64) {
    let missing = oblikovati_model::document_sketch::DocumentSketchId(u64::MAX);
    let error = files
        .commands(document)
        .unwrap()
        .copy_sketch_entities(missing, &[SketchEntityId(line)], [0.0; 2])
        .unwrap_err();
    assert!(error.to_string().contains(&u64::MAX.to_string()));
    assert!(matches!(
        error,
        CommandError::Edit(EditError::Mutation(MutationError::Command(
            SketchCopyError::Sketch(_)
        )))
    ));
}
