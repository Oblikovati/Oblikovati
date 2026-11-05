// SPDX-License-Identifier: GPL-2.0-only
use super::entities::{activate_copy, install_mixed, rows, verify_keys};
use super::*;
use oblikovati_api::wire::SketchEntityInfo;
use oblikovati_model::document_sketch::{DocumentSketches, SketchConstructionReport};
fn selection(files: &DiscoveryWorkspace) -> Vec<SketchEntityId> {
    rows(files)
        .iter()
        .filter(|row| matches!(row.kind.as_str(), "line" | "circle" | "arc"))
        .map(|row| SketchEntityId(row.id))
        .collect()
}
fn owned_sketch(files: &DiscoveryWorkspace, document: u64) -> DocumentSketchId {
    owned_geometry(files, document).sketches()[0].id()
}
fn set_roles(
    files: &mut DiscoveryWorkspace,
    document: u64,
    ids: &[SketchEntityId],
    construction: bool,
) -> SketchConstructionReport {
    let sketch = owned_sketch(files, document);
    files
        .commands(document)
        .unwrap()
        .set_sketch_construction(sketch, ids, construction)
        .unwrap()
}
fn numerical_signature(sketches: &DocumentSketches) -> Vec<u64> {
    let geometry = sketches.sketches()[0].geometry();
    geometry
        .points()
        .iter()
        .flat_map(|point| point.position.map(f64::to_bits))
        .chain(
            geometry
                .circles()
                .iter()
                .map(|circle| circle.radius.to_bits()),
        )
        .collect()
}
fn verification(files: &DiscoveryWorkspace, document: u64) -> SketchEvaluation {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    match snapshot.sketches.sketches()[0].health(&snapshot.parameters) {
        PlacedSketchHealth::Current(evaluation) => evaluation.clone(),
        unexpected => panic!("verification {unexpected:?}: expected current evidence"),
    }
}
fn assert_flagged_rows(original: &[SketchEntityInfo], current: &[SketchEntityInfo]) {
    let mut expected = original.to_vec();
    for row in &mut expected {
        row.construction = matches!(row.kind.as_str(), "line" | "circle" | "arc");
    }
    assert_eq!(current, expected);
}
#[test]
fn typed_metadata_command_flags_curves_without_changing_numerics_health_or_keys() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let before = rows(&files);
    let ids = selection(&files);
    let report = set_roles(&mut files, document, &ids, true);
    assert_eq!(report.changed, ids);
    assert_flagged_rows(&before, &rows(&files));
    assert!(files.workspace().needs_save(document).unwrap());
    verify_keys(&files, &rows(&files));
}
#[test]
fn metadata_updates_retain_exact_numeric_bits_and_current_verification() {
    let (mut files, document, _, _) = files_with_circle(Default::default());
    let numeric = numerical_signature(owned_geometry(&files, document));
    let evidence = verification(&files, document);
    let ids = selection(&files);
    set_roles(&mut files, document, &ids, true);
    assert_eq!(
        numeric,
        numerical_signature(owned_geometry(&files, document))
    );
    assert_eq!(evidence, verification(&files, document));
}
#[test]
fn undo_redo_restore_construction_as_one_normal_history_step() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    files.save(document, "test", &mut source).unwrap();
    let before = rows(&files);
    let ids = selection(&files);
    set_roles(&mut files, document, &ids, true);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Set sketch construction"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    assert!(!files.workspace().needs_save(document).unwrap());
    files.commands(document).unwrap().redo().unwrap();
    assert_flagged_rows(&before, &rows(&files));
}
#[test]
fn no_op_and_empty_metadata_commands_keep_committed_history_and_saved_state() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let ids = selection(&files);
    let before = owner_revision(&files, document);
    assert!(
        set_roles(&mut files, document, &ids, false)
            .changed
            .is_empty()
    );
    assert!(
        set_roles(&mut files, document, &[], true)
            .changed
            .is_empty()
    );
    let after = owner_revision(&files, document);
    // #3768: history assigns a generation to each accepted candidate, including equal recipes.
    assert_eq!(after.0, before.0 + 2);
    assert_eq!((after.1, after.2), (before.1, before.2));
}
#[test]
fn refused_selections_keep_state_and_no_metadata_command_consumes_entity_ids() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let ids = selection(&files);
    set_roles(&mut files, document, &[], true);
    let before = owner_revision(&files, document);
    refuse_invalid_selections(&mut files, document, sketch, &ids);
    assert_eq!(before, owner_revision(&files, document));
    create_next_origin(&mut files, &mut source);
    assert_eq!(next_anchor(&files, document), 3);
}
fn refuse_invalid_selections(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
    ids: &[SketchEntityId],
) {
    assert!(
        files
            .commands(document)
            .unwrap()
            .set_sketch_construction(sketch, &[ids[0], SketchEntityId(99)], true)
            .is_err()
    );
    assert!(
        files
            .commands(document)
            .unwrap()
            .set_sketch_construction(DocumentSketchId(99), ids, true)
            .is_err()
    );
}
fn create_next_origin(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
}
fn next_anchor(files: &DiscoveryWorkspace, document: u64) -> u64 {
    owned_geometry(files, document).sketches()[1]
        .geometry()
        .points()[0]
        .id
        .0
}
#[test]
fn saved_reopened_and_copied_construction_flags_keep_profile_membership_and_local_ids() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let ids = selection(&files);
    set_roles(&mut files, document, &ids, true);
    let saved = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), saved);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/construction-copy.obk"),
    );
    let copied = rows(&files);
    assert_copied_flags(&saved, &copied);
    verify_keys(&files, &copied);
}
fn assert_copied_flags(saved: &[SketchEntityInfo], copied: &[SketchEntityInfo]) {
    for (first, second) in saved.iter().zip(copied) {
        assert_eq!(
            (first.id, first.construction, &first.points),
            (second.id, second.construction, &second.points)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
}
#[test]
fn response_preparation_failure_discards_construction_before_owner_adoption() {
    let (mut files, document, sketch, _) = files_with_circle(Default::default());
    let ids = selection(&files);
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    let result = files
        .commands(document)
        .unwrap()
        .set_sketch_construction_projected(sketch, &ids, true, |report| {
            assert_eq!(report.unwrap().changed, ids);
            Err::<(), _>("serialization refused")
        });
    let cause = result.unwrap_err().to_string();
    assert!(cause.contains("serialization refused"));
    assert_eq!(before, owner_revision(&files, document));
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}
#[test]
fn vetoed_metadata_keeps_the_saved_flag_state_and_evidence() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let sketch = owned_sketch(&files, document);
    let ids = selection(&files);
    let original = rows(&files);
    let cause = files
        .commands(document)
        .unwrap()
        .set_sketch_construction(sketch, &ids, true)
        .unwrap_err()
        .to_string();
    assert!(cause.contains("locked"), "{cause}");
    assert_eq!(rows(&files), original);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn metadata_changes_keep_failed_numerical_evidence_without_retrying_the_solver() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let entity = circle_entity(&files, document, sketch);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .parameters()
        .set(&numeric("width", "missing"))
        .unwrap();
    let before = verification(&files, document);
    assert!(matches!(before, SketchEvaluation::Failed(_)));
    set_roles(&mut files, document, &[entity], true);
    assert_eq!(before, verification(&files, document));
    assert!(
        owned_geometry(&files, document)
            .get(sketch)
            .unwrap()
            .geometry()
            .is_construction(entity)
            .unwrap()
    );
}

fn circle_entity(
    files: &DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> SketchEntityId {
    owned_geometry(files, document)
        .get(sketch)
        .unwrap()
        .geometry()
        .circles()[0]
        .id
}

#[test]
fn grouped_flag_edits_abort_without_recording_a_step_or_changing_saved_geometry() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = rows(&files);
    let labels = owner_revision(&files, document).1;
    let ids = selection(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Construction flags")
        .unwrap();
    set_roles(&mut files, document, &ids, true);
    assert!(rows(&files)[1].construction);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), before);
    assert_eq!(owner_revision(&files, document).1, labels);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn grouped_set_and_clear_cancel_as_content_and_create_no_committed_step() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = rows(&files);
    let labels = owner_revision(&files, document).1;
    let ids = selection(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Construction flags")
        .unwrap();
    set_roles(&mut files, document, &ids, true);
    set_roles(&mut files, document, &ids, false);
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(rows(&files), before);
    assert_eq!(owner_revision(&files, document).1, labels);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn grouped_metadata_commit_preserves_the_callers_label_for_undo_and_redo() {
    let (mut files, document, _, _) = files_with_circle(Default::default());
    let ids = selection(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Construction flags")
        .unwrap();
    set_roles(&mut files, document, &ids, true);
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Construction flags"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert!(!rows(&files)[1].construction);
    files.commands(document).unwrap().redo().unwrap();
    assert!(rows(&files)[1].construction);
}
#[test]
fn assembly_metadata_commands_use_the_same_owned_curve_boundary() {
    let (mut files, document, _) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    install_mixed(&mut files, document);
    let before = rows(&files);
    let ids = selection(&files);
    set_roles(&mut files, document, &ids, true);
    assert_flagged_rows(&before, &rows(&files));
}
