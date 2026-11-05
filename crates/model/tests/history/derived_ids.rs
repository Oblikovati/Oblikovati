// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn add_table(history: &mut DocumentEditHistory) -> i64 {
    let token = history.begin("Link").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            let id = ids
                .allocate_derived_table()
                .map_err(|error| error.to_string())?;
            snapshot
                .parameters
                .restore_derived_table(id, "source.opd", &[], false)
                .map_err(|error| error.to_string())?;
            Ok::<_, String>(id)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    id
}

#[test]
fn table_reservations_survive_undo_abort_and_failed_edits() {
    let mut history = history();
    assert_eq!(add_table(&mut history), 1);
    history.undo().unwrap();
    assert_eq!(add_table(&mut history), 2);
    let token = history.begin("Discard").unwrap();
    history
        .edit(&token, |_, ids| {
            assert_eq!(ids.allocate_derived_table().unwrap(), 3);
            Ok::<_, String>(())
        })
        .unwrap();
    history.abort(&token).unwrap();
    let token = history.begin("Fail").unwrap();
    assert!(
        history
            .edit(&token, |_, ids| {
                assert_eq!(ids.allocate_derived_table().unwrap(), 4);
                Err::<(), _>("rejected")
            })
            .is_err()
    );
    history.abort(&token).unwrap();
    assert_eq!(add_table(&mut history), 5);
    assert_eq!(history.high_water(), 1);
}

#[test]
fn edits_and_evaluation_cannot_invent_table_identities() {
    let mut history = history();
    let token = history.begin("Unreserved").unwrap();
    let error = history
        .edit(&token, |snapshot, _| {
            snapshot
                .parameters
                .restore_derived_table(7, "source.opd", &[], false)
        })
        .unwrap_err()
        .to_string();
    assert!(error.contains("derived table ID 7"), "{error}");
    assert!(history.snapshot().parameters.derived_tables().is_empty());
    history.abort(&token).unwrap();
    assert!(
        history
            .refresh(|snapshot| {
                snapshot
                    .parameters
                    .restore_derived_table(1, "source.opd", &[], false)
            })
            .is_err()
    );
    assert_eq!(add_table(&mut history), 1);
    history.undo().unwrap();
    let token = history.begin("Reuse").unwrap();
    assert!(
        history
            .edit(&token, |snapshot, _| {
                snapshot
                    .parameters
                    .restore_derived_table(1, "source.opd", &[], false)
            })
            .is_err()
    );
    history.abort(&token).unwrap();
}

#[test]
fn restored_table_namespace_is_independent_and_signed_exhaustion_is_named() {
    let mut snapshot = history().snapshot().clone();
    snapshot
        .parameters
        .restore_derived_table(i64::MAX, "source.opd", &[], false)
        .unwrap();
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let token = history.begin("Exhausted").unwrap();
    let error = history
        .edit(&token, |_, ids| ids.allocate_derived_table())
        .unwrap_err()
        .to_string();
    assert!(error.contains("below i64::MAX"), "{error}");
    history.abort(&token).unwrap();
    assert_eq!(history.high_water(), 1);
    assert_eq!(
        history.snapshot().parameters.derived_tables()[0].id,
        i64::MAX
    );
}

#[test]
fn restored_tables_seed_reservations_and_redo_retains_existing_ids() {
    let mut snapshot = history().snapshot().clone();
    snapshot
        .parameters
        .restore_derived_table(7, "source.opd", &[], false)
        .unwrap();
    let mut history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    assert_eq!(add_table(&mut history), 8);
    history.undo().unwrap();
    history.redo().unwrap();
    assert!(history.snapshot().parameters.derived_table(8).is_some());
    assert_eq!(add_table(&mut history), 9);
}

#[test]
fn saved_reservations_restore_without_rewinding_live_tables() {
    let mut history = history().with_derived_reservations(9).unwrap();
    assert_eq!(history.derived_high_water(), 9);
    assert_eq!(add_table(&mut history), 10);
    let error = history
        .with_derived_reservations(8)
        .unwrap_err()
        .to_string();
    assert!(error.contains("expected at least 10"), "{error}");
}
