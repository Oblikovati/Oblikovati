// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        DocumentEditHistory, DocumentSnapshot, EditError, HistoryError, SketchReservationMarks,
    },
    identity::RecipeIdAllocator,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
fn history() -> DocumentEditHistory {
    DocumentEditHistory::new(
        document_snapshot::empty_snapshot(),
        RecipeIdAllocator::default(),
    )
    .unwrap()
}

fn create_tagged_sketch(history: &mut DocumentEditHistory) -> DocumentSketchId {
    let token = history.begin("Create sketch").unwrap();
    let sketch = history
        .edit(&token, |snapshot, ids| {
            let sketch = ids.allocate_sketch().unwrap();
            snapshot
                .sketches
                .add(sketch, "First", SketchPlane::xy(), Sketch2D::default())
                .unwrap();
            Ok::<_, String>(sketch)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    history.tag_sketch_creation(sketch).unwrap();
    sketch
}

#[test]
fn sketch_editor_history_replays_without_changing_the_recipe() {
    let mut history = history();
    let sketch = create_tagged_sketch(&mut history);
    let before = history.snapshot().sketches.clone();
    history.record_sketch_finish(sketch).unwrap();
    assert_eq!(
        history.undo_sketch_transition().unwrap().before,
        Some(sketch)
    );
    history.undo().unwrap();
    assert_eq!(history.snapshot().sketches, before);
    assert_eq!(history.undo_sketch_transition().unwrap().before, None);
    history.undo().unwrap();
    assert!(history.snapshot().sketches.get(sketch).is_err());
    assert_eq!(
        history.redo_sketch_transition().unwrap().after,
        Some(sketch)
    );
    history.redo().unwrap();
    assert_eq!(history.redo_sketch_transition().unwrap().after, None);
    history.redo().unwrap();
    assert_eq!(history.snapshot().sketches, before);
}

#[test]
fn a_recipe_merge_does_not_absorb_a_sketch_editor_transition() {
    let mut history = history();
    let sketch = create_tagged_sketch(&mut history);
    history.record_sketch_finish(sketch).unwrap();
    history.undo().unwrap();
    let edit = history.begin("Hide sketch").unwrap();
    history
        .edit(&edit, |snapshot, _| {
            snapshot.sketches.set_visible(sketch, false)
        })
        .unwrap();
    history.commit(&edit, true).unwrap();
    assert_eq!(history.undo_labels(), ["Create sketch", "Hide sketch"]);
    assert!(history.redo_labels().is_empty());
    history.undo().unwrap();
    assert!(history.snapshot().sketches.get(sketch).unwrap().visible());
    assert_eq!(
        history.undo_sketch_transition().unwrap().after,
        Some(sketch)
    );
}

#[test]
fn sketch_editor_history_rejects_missing_sketches_and_open_transactions() {
    let mut history = history();
    let missing = DocumentSketchId(7);
    assert!(
        history
            .record_sketch_finish(missing)
            .unwrap_err()
            .to_string()
            .contains('7')
    );
    assert!(history.tag_sketch_creation(missing).is_err());
    let group = history.begin("Open").unwrap();
    assert!(history.record_sketch_finish(missing).is_err());
    assert!(history.tag_sketch_creation(missing).is_err());
    history.abort(&group).unwrap();
    assert!(history.undo_labels().is_empty());
}
#[test]
fn undo_redo_abort_and_failed_edits_never_release_sketch_or_entity_ids() {
    let mut history = history();
    let token = history.begin("create").unwrap();
    history
        .edit(&token, |snapshot, ids| -> Result<(), String> {
            let sketch = ids.allocate_sketch().map_err(|e| e.to_string())?;
            let point = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
            let mut geometry = Sketch2D::default();
            geometry.add_point(point, [1.0, 2.0]).unwrap();
            snapshot
                .sketches
                .add(sketch, "First", SketchPlane::xy(), geometry)
                .map_err(|e| e.to_string())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let first = history.snapshot().sketches.clone();
    history.undo().unwrap();
    assert!(history.snapshot().sketches.sketches().is_empty());
    assert_eq!(
        history.sketch_reservations(),
        SketchReservationMarks {
            sketch: 1,
            entity: 1,
            constraints: Default::default(),
            dimension_names: Default::default(),
        }
    );
    history.redo().unwrap();
    assert_eq!(history.snapshot().sketches, first);
    let token = history.begin("refused").unwrap();
    let result = history.edit(&token, |_, ids| -> Result<(), String> {
        assert_eq!(ids.allocate_sketch().unwrap().0, 2);
        assert_eq!(ids.allocate_sketch_entity().unwrap().0, 2);
        Err("refused".into())
    });
    assert!(matches!(result, Err(EditError::Mutation(_))));
    history.abort(&token).unwrap();
    assert_eq!(history.snapshot().sketches, first);
    assert_eq!(
        history.sketch_reservations(),
        SketchReservationMarks {
            sketch: 2,
            entity: 2,
            constraints: Default::default(),
            dimension_names: Default::default(),
        }
    );
}
#[test]
fn unreserved_identities_and_reuse_of_deleted_reservations_are_refused() {
    let mut history = history()
        .with_sketch_reservations(SketchReservationMarks {
            sketch: 5,
            entity: 8,
            constraints: Default::default(),
            dimension_names: Default::default(),
        })
        .unwrap();
    let token = history.begin("invalid").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        snapshot.sketches.add(
            DocumentSketchId(1),
            "Reused",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::UnreservedSketchId(1)))
    ));
    let result = history.edit(&token, |snapshot, ids| {
        let sketch = ids.allocate_sketch().unwrap();
        let mut geometry = Sketch2D::default();
        geometry.add_point(SketchEntityId(8), [0.0, 0.0]).unwrap();
        snapshot
            .sketches
            .add(sketch, "Bad entity", SketchPlane::xy(), geometry)
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::UnreservedSketchEntityId(
            8
        )))
    ));
    assert!(history.snapshot().sketches.sketches().is_empty());
    assert_eq!(history.sketch_reservations().sketch, 6);
    history.abort(&token).unwrap();
    assert!(
        history
            .with_sketch_reservations(SketchReservationMarks {
                sketch: 5,
                entity: 8,
                constraints: Default::default(),
                dimension_names: Default::default(),
            })
            .is_err()
    );
}
#[test]
fn refresh_detects_sketch_changes_but_cannot_create_unreserved_identities() {
    let mut history = history();
    let token = history.begin("create").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            snapshot.sketches.add(
                ids.allocate_sketch().unwrap(),
                "First",
                SketchPlane::xy(),
                Sketch2D::default(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let generation = history.generation();
    assert!(
        history
            .refresh(|snapshot| snapshot.sketches.rename(DocumentSketchId(1), "Updated"))
            .unwrap()
    );
    assert_eq!(history.generation(), generation + 1);
    assert!(
        !history
            .refresh(|snapshot| snapshot.sketches.rename(DocumentSketchId(1), "Updated"))
            .unwrap()
    );
    let error = history
        .refresh(|snapshot| {
            snapshot.sketches.add(
                DocumentSketchId(2),
                "Unreserved",
                SketchPlane::xy(),
                Sketch2D::default(),
            )
        })
        .unwrap_err();
    assert!(matches!(
        error,
        EditError::History(HistoryError::UnreservedSketchId(2))
    ));
    assert_eq!(history.snapshot().sketches.sketch_ids(), [1]);
}
#[test]
fn saved_marks_cover_live_ids_and_maximum_namespaces_refuse_allocation() {
    let mut history = history()
        .with_sketch_reservations(SketchReservationMarks {
            sketch: u64::MAX,
            entity: u64::MAX,
            constraints: Default::default(),
            dimension_names: Default::default(),
        })
        .unwrap();
    let token = history.begin("exhausted").unwrap();
    history
        .edit(&token, |_, ids| -> Result<(), String> {
            assert!(ids.allocate_sketch().is_err());
            assert!(ids.allocate_sketch_entity().is_err());
            Ok(())
        })
        .unwrap();
    history.abort(&token).unwrap();
    assert_eq!(history.sketch_reservations().entity, u64::MAX);
}

#[test]
fn source_refresh_adopts_health_only_verification_without_an_authored_recipe_change() {
    let mut history = history();
    let token = history.begin("Sketch").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            snapshot.sketches.add(
                ids.allocate_sketch().unwrap(),
                "Empty",
                SketchPlane::xy(),
                Sketch2D::default(),
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let before = history.snapshot().sketches.clone();
    assert!(history.refresh(verify_sketches).unwrap());
    assert!(history.snapshot().sketches.same_recipe(&before));
    assert!(!history.snapshot().sketches.same_evaluated_state(&before));
    assert!(!history.refresh(verify_sketches).unwrap());
}
fn verify_sketches(snapshot: &mut DocumentSnapshot) -> Result<(), String> {
    snapshot.sketches = snapshot
        .sketches
        .solve_pending(&snapshot.parameters, Default::default(), || false)
        .candidate;
    Ok(())
}
#[path = "history_sketches/placement.rs"]
mod placement;
#[path = "history_sketches/polygon.rs"]
mod polygon;
#[path = "history_sketches/polyline.rs"]
mod polyline;
#[path = "history_sketches/slots.rs"]
mod slots;

#[path = "history_sketches/rectangles.rs"]
mod rectangles;

#[path = "history_sketches/circles.rs"]
mod circles;
