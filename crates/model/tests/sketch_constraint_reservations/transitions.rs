// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::{EditError, HistoryError};
fn replace_constraint(
    snapshot: &mut DocumentSnapshot,
    sketch: u64,
    constraint: u64,
) -> Result<(), String> {
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(sketch), fixed_geometry(sketch, constraint))
        .map_err(|cause| cause.to_string())
}
fn reserve_replacement(history: &mut DocumentEditHistory, label: &str) -> u64 {
    let token = history.begin(label).unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            let constraint = ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap();
            replace_constraint(snapshot, 1, constraint.0)?;
            Ok::<_, String>(constraint.0)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    id
}
fn live_constraint(history: &DocumentEditHistory) -> u64 {
    history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .constraints()[0]
        .id
        .0
}
fn assert_unreserved(result: Result<(), EditError<String>>, sketch: u64, constraint: u64) {
    match result {
        Err(EditError::History(HistoryError::UnreservedSketchConstraintId {
            sketch: actual,
            constraint: relation,
        })) => {
            assert_eq!((actual, relation), (sketch, constraint));
        }
        other => panic!(
            "constraint {constraint} in sketch {sketch}: expected unreserved-ID error, got {other:?}"
        ),
    }
}
#[test]
fn undo_redo_and_aborted_replacements_preserve_permanent_marks() {
    let mut history = history();
    assert_eq!(reserve_replacement(&mut history, "Replace"), 8);
    history.undo().unwrap();
    assert_eq!(live_constraint(&history), 7);
    assert_eq!(history.sketch_reservations().constraints[&1], 8);
    history.redo().unwrap();
    assert_eq!(live_constraint(&history), 8);
    let token = history.begin("Abort").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap();
            replace_constraint(snapshot, 1, id.0)
        })
        .unwrap();
    history.abort(&token).unwrap();
    assert_eq!(live_constraint(&history), 8);
    assert_eq!(reserve_replacement(&mut history, "Next"), 10);
}
#[test]
fn mutation_and_validation_failures_reserve_ids_without_publishing_geometry() {
    let mut history = history();
    let token = history.begin("Refused").unwrap();
    let result = history.edit(&token, |snapshot, ids| -> Result<(), String> {
        let id = ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap();
        replace_constraint(snapshot, 1, id.0)?;
        Err("mutation refused".into())
    });
    assert!(matches!(result, Err(EditError::Mutation(_))));
    let failed = history.edit(&token, |snapshot, ids| {
        let allocated = ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap();
        assert_eq!(allocated.0, 9);
        replace_constraint(snapshot, 1, 8)
    });
    assert_unreserved(failed, 1, 8);
    assert_eq!(live_constraint(&history), 7);
    history.abort(&token).unwrap();
    assert_eq!(reserve_replacement(&mut history, "Next"), 10);
}
#[test]
fn another_namespace_reservation_cannot_authorize_a_constraint() {
    let mut history = history();
    let token = history.begin("Wrong owner").unwrap();
    assert_unreserved(
        history.edit(&token, |snapshot, ids| {
            let constraint = ids.allocate_sketch_constraint(DocumentSketchId(2)).unwrap();
            replace_constraint(snapshot, 1, constraint.0)
        }),
        1,
        8,
    );
    history.abort(&token).unwrap();
    assert_eq!(
        history.sketch_reservations().constraints,
        BTreeMap::from([(1, 7), (2, 8)])
    );
}
#[test]
fn retired_relations_cannot_be_resurrected_by_edits_or_refresh() {
    let mut history = history();
    reserve_replacement(&mut history, "Replace");
    for constraint in [7, 9] {
        let result = history.refresh(|snapshot| replace_constraint(snapshot, 1, constraint));
        assert_unreserved(result.map(|_| ()), 1, constraint);
    }
    let token = history.begin("Resurrect").unwrap();
    assert_unreserved(
        history.edit(&token, |snapshot, _| replace_constraint(snapshot, 1, 7)),
        1,
        7,
    );
    history.abort(&token).unwrap();
    assert_eq!(live_constraint(&history), 8);
}
#[test]
fn deleting_a_sketch_keeps_its_local_marks_through_undo_and_redo() {
    let mut history = history();
    let token = history.begin("Delete").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.sketches.remove(DocumentSketchId(1))
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(history.sketch_reservations().constraints[&1], 7);
    history.undo().unwrap();
    assert_eq!(live_constraint(&history), 7);
    history.redo().unwrap();
    assert_eq!(history.sketch_reservations().constraints[&1], 7);
}
