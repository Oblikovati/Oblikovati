// SPDX-License-Identifier: GPL-2.0-only
use super::entities::{author_large_point, configured_scaled, install_mixed, rows, verify_keys};
use super::primitive::{create_circle, create_owner};
use super::*;
use oblikovati_api::wire::SketchEntityInfo;
#[path = "translation/spline.rs"]
mod spline;
fn movement(entities: Vec<u64>, vector: Vec<f64>) -> TransformSketchArgs {
    TransformSketchArgs {
        op: "move".into(),
        entities: Some(entities),
        vector: Some(vector),
        ..Default::default()
    }
}
fn move_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &TransformSketchArgs,
) -> Result<TransformSketchResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .transform(input)
}
fn all_ids(files: &DiscoveryWorkspace) -> Vec<u64> {
    rows(files).iter().map(|row| row.id).collect()
}
fn assert_translated(before: &[SketchEntityInfo], after: &[SketchEntityInfo], vector: [f64; 2]) {
    let mut expected = before.to_vec();
    for row in &mut expected {
        for point in row.points.as_mut().unwrap() {
            let coordinates = point.as_mut().unwrap();
            coordinates[0] += vector[0];
            coordinates[1] += vector[1];
        }
    }
    assert_eq!(after, expected);
}
#[test]
fn canonical_move_translates_each_shared_point_once_and_preserves_current_keys() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let before = rows(&files);
    let ids = all_ids(&files);
    let reply = move_call(&mut files, &mut source, &movement(ids, vec![3.0, 4.0])).unwrap();
    assert!(reply.created.is_none());
    assert_translated(&before, &rows(&files), [3.0, 4.0]);
    verify_keys(&files, &before);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Move sketch entities"
    );
}
#[test]
fn centimetre_translation_is_converted_to_each_documents_working_units() {
    for scale in [0.1, 2.0] {
        let (mut files, document, mut source) = configured_scaled(scale);
        install_mixed(&mut files, document);
        let before = rows(&files);
        let ids = all_ids(&files);
        move_call(&mut files, &mut source, &movement(ids, vec![1.0, 2.0])).unwrap();
        assert_translated(&before, &rows(&files), [1.0, 2.0]);
    }
}
#[test]
fn duplicate_selection_and_selected_shared_point_follow_unique_point_semantics() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let before = rows(&files);
    let ids = all_ids(&files);
    let repeated = ids.iter().copied().chain(ids.iter().copied()).collect();
    move_call(&mut files, &mut source, &movement(repeated, vec![3.0, 4.0])).unwrap();
    assert_translated(&before, &rows(&files), [3.0, 4.0]);
}
fn assert_move_refused(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    input: &TransformSketchArgs,
    expected: &str,
) {
    let before = owner_revision(files, document);
    let geometry = owned_geometry(files, document).clone();
    expect_api_decline(move_call(files, source, input), expected);
    assert_eq!(before, owner_revision(files, document));
    assert!(geometry.same_evaluated_state(owned_geometry(files, document)));
}
#[test]
fn invalid_batches_vectors_indices_and_unported_operations_preserve_owner_state() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    refuse_invalid_vectors(&mut files, document, &mut source);
    assert_move_refused(
        &mut files,
        document,
        &mut source,
        &movement(vec![1, 99], vec![1.0; 2]),
        "99",
    );
    let mut input = movement(vec![1], vec![1.0; 2]);
    input.sketch_index = -1;
    assert_move_refused(&mut files, document, &mut source, &input, "ordered index");
    input.sketch_index = 0;
    input.op = "trim".into();
    assert_move_refused(&mut files, document, &mut source, &input, "not ported");
}
#[test]
fn working_unit_conversion_overflow_declines_before_a_command_runs() {
    let (mut files, document, mut source) = configured_scaled(0.1);
    install_mixed(&mut files, document);
    assert_move_refused(
        &mut files,
        document,
        &mut source,
        &movement(vec![1], vec![f64::MAX, 0.0]),
        "representable in working units",
    );
}
#[test]
fn output_coordinate_overflow_discards_every_selected_coordinate() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Large point", author_large_point)
        .unwrap();
    assert_move_refused(
        &mut files,
        document,
        &mut source,
        &movement(vec![1], vec![f64::MAX, 0.0]),
        "transform point",
    );
}
#[test]
fn empty_api_selection_is_declined_and_zero_move_creates_no_committed_step() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    assert_move_refused(
        &mut files,
        document,
        &mut source,
        &movement(vec![], vec![1.0; 2]),
        "at least one owned entity",
    );
    let ids = all_ids(&files);
    move_call(&mut files, &mut source, &movement(ids, vec![0.0; 2])).unwrap();
    let after = owner_revision(&files, document);
    assert_eq!((after.1, after.2), (before.1, before.2));
}
#[test]
fn undo_redo_and_save_reopen_restore_the_moved_recipe() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let before = rows(&files);
    let ids = all_ids(&files);
    move_call(&mut files, &mut source, &movement(ids, vec![3.0, 4.0])).unwrap();
    let moved = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), moved);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), moved);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn grounded_centers_are_restored_by_current_source_recompute_in_both_document_kinds() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let sketch = create_owner(&mut files, document);
        let circle = create_circle(&mut files, document, sketch);
        let before = rows(&files);
        move_call(
            &mut files,
            &mut source,
            &movement(vec![circle.entity.0], vec![3.0, 4.0]),
        )
        .unwrap();
        assert_eq!(rows(&files), before);
        verify_keys(&files, &before);
    }
}
#[test]
fn reply_failure_discards_the_translated_snapshot_and_current_evidence() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let original = owned_geometry(&files, document).clone();
    let before = owner_revision(&files, document);
    let result = files
        .commands(document)
        .unwrap()
        .translate_sketch_entities_projected(sketch, &[SketchEntityId(1)], [3.0, 4.0], |result| {
            assert_eq!(result.unwrap(), [SketchEntityId(1)]);
            Err::<(), _>("reply refused")
        });
    assert!(result.is_err());
    assert!(original.same_evaluated_state(owned_geometry(&files, document)));
    assert_eq!(owner_revision(&files, document), before);
}

fn refuse_invalid_vectors(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    for vector in [vec![], vec![1.0], vec![1.0, 2.0, 3.0]] {
        assert_move_refused(
            files,
            document,
            source,
            &movement(vec![1], vector),
            "two finite",
        );
    }
}
#[test]
fn typed_move_reports_authored_defining_points_and_keeps_curve_roles() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    mark_construction(&mut files, document, sketch);
    let before = rows(&files);
    let moved = files
        .commands(document)
        .unwrap()
        .translate_sketch_entities(sketch, &[SketchEntityId(6)], [3.0, 4.0])
        .unwrap();
    assert_eq!(
        moved,
        [SketchEntityId(1), SketchEntityId(2), SketchEntityId(5)]
    );
    assert_translated(&before, &rows(&files), [3.0, 4.0]);
}
#[test]
fn grouped_move_abort_and_commit_use_the_existing_history_boundary() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let original = rows(&files);
    let ids = all_ids(&files);
    start_moved_group(&mut files, document, &mut source, ids.clone());
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), original);
    start_moved_group(&mut files, document, &mut source, ids);
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Move selection"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), original);
}
#[test]
fn vetoed_move_restores_the_saved_recipe_and_query_host_cannot_move() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let original = rows(&files);
    let input = movement(all_ids(&files), vec![3.0, 4.0]);
    expect_api_decline(move_call(&mut files, &mut source, &input), "locked");
    assert_eq!(rows(&files), original);
    assert!(!files.workspace().needs_save(document).unwrap());
    assert_query_move_refused(&files, &input);
    assert_eq!(rows(&files), original);
}

fn mark_construction(files: &mut DiscoveryWorkspace, document: u64, sketch: DocumentSketchId) {
    files
        .commands(document)
        .unwrap()
        .set_sketch_construction(sketch, &[SketchEntityId(4), SketchEntityId(6)], true)
        .unwrap();
}
fn begin_move_group(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .begin("Move selection")
        .unwrap();
}
fn assert_query_move_refused(files: &DiscoveryWorkspace, input: &TransformSketchArgs) {
    expect_api_decline(
        Client::new(FileQueryHost::new(files))
            .sketch()
            .transform(input),
        "sketch.transform",
    );
}

fn start_moved_group(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    ids: Vec<u64>,
) {
    begin_move_group(files, document);
    move_call(files, source, &movement(ids, vec![3.0, 4.0])).unwrap();
}
#[test]
fn fixed_dimensions_still_control_the_shape_after_a_translation_seed_edit() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let radius = circle_radius(files.workspace(), document, sketch);
    move_call(&mut files, &mut source, &movement(vec![2], vec![3.0, 4.0])).unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let view = snapshot.require_resolved_sketch(sketch).unwrap();
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    assert_eq!(view.sketch.geometry().circles()[0].radius, radius);
    let bound = SolveOptions::default().relative_tolerance * 3.0_f64.hypot(4.0);
    assert!(
        view.sketch.geometry().points()[0]
            .position
            .iter()
            .all(|value| value.abs() <= bound)
    );
}
#[test]
fn translation_consumes_no_new_entity_reservation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let ids = all_ids(&files);
    move_call(&mut files, &mut source, &movement(ids, vec![3.0, 4.0])).unwrap();
    let sketch = owned_geometry(&files, document).sketches()[0].id();
    let report = files
        .commands(document)
        .unwrap()
        .create_sketch_primitive(
            sketch,
            &oblikovati_model::sketch::SketchPrimitiveCreation::Point { position: [1.0; 2] },
        )
        .unwrap();
    assert_eq!(report.entity, SketchEntityId(7));
}
#[test]
fn unsupported_document_kinds_decline_the_canonical_edit() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    assert_move_refused(
        &mut files,
        document,
        &mut source,
        &movement(vec![1], vec![3.0, 4.0]),
        "part or assembly document",
    );
}
#[path = "rotation.rs"]
mod rotation;

#[path = "copy.rs"]
mod copy;
#[path = "mirror.rs"]
mod mirror;
#[path = "split.rs"]
mod split;
