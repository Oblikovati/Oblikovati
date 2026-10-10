// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{USER_PARAM, VariantValue};
use oblikovati_model::{
    history::*,
    identity::RecipeIdAllocator,
    parameter::{ParameterCollection, ParameterError},
    property::{DocumentProperties, USER_DEFINED},
    units::UnitsOfMeasure,
};

fn history() -> DocumentEditHistory {
    let mut ids = RecipeIdAllocator::default();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(ids.allocate().unwrap(), "width", USER_PARAM, "10 mm")
        .unwrap();
    DocumentEditHistory::new(
        DocumentSnapshot {
            parameters,
            properties: DocumentProperties::default(),
        },
        ids,
    )
    .unwrap()
}

fn set_width(history: &mut DocumentEditHistory, token: &TransactionToken, source: &str) {
    history
        .edit(token, |snapshot, _| {
            snapshot.parameters.set_expression(1, source)
        })
        .unwrap();
}

fn width(history: &DocumentEditHistory) -> f64 {
    history
        .snapshot()
        .parameters
        .get(1)
        .unwrap()
        .current_value()
        .unwrap()
        .value
}

#[test]
fn nested_commit_is_one_step_and_abort_restores_only_the_active_frame() {
    let mut history = history();
    let outer = history.begin("outer").unwrap();
    set_width(&mut history, &outer, "20 mm");
    let child = history.begin("child").unwrap();
    set_width(&mut history, &child, "30 mm");
    assert!(history.commit(&outer, false).is_err());
    history.abort(&child).unwrap();
    assert_eq!(width(&history), 2.0);
    let child = history.begin("child2").unwrap();
    set_width(&mut history, &child, "40 mm");
    history.commit(&child, true).unwrap();
    assert!(history.undo_labels().is_empty());
    history.commit(&outer, false).unwrap();
    assert_eq!(history.undo_labels(), ["outer"]);
    assert_eq!(history.undo().unwrap(), "outer");
    assert_eq!(width(&history), 1.0);
    assert_eq!(history.redo().unwrap(), "outer");
    assert_eq!(width(&history), 4.0);
}

#[test]
fn failed_mutation_is_atomic_but_reserved_ids_are_never_reused() {
    let mut history = history();
    let token = history.begin("allocation").unwrap();
    let generation = history.generation();
    let result = history.edit(&token, |snapshot, ids| {
        let id = ids.allocate().unwrap();
        snapshot
            .parameters
            .insert_expression(id, "temporary", USER_PARAM, "2")?;
        snapshot.parameters.set_expression(1, "(")
    });
    assert!(matches!(result, Err(EditError::Mutation(_))));
    let error = result.unwrap_err();
    assert!(std::error::Error::source(&error).is_some());
    assert!(!error.to_string().is_empty());
    assert_eq!(history.generation(), generation);
    assert_eq!(history.snapshot().parameters.ids(), [1]);
    assert_eq!(history.high_water(), 2);
    let id = history
        .edit(&token, |snapshot, ids| {
            let id = ids.allocate().unwrap();
            snapshot
                .parameters
                .insert_expression(id, "kept", USER_PARAM, "3")?;
            Ok::<_, ParameterError<u64>>(id)
        })
        .unwrap();
    assert_eq!(id, 3);
    history.commit(&token, false).unwrap();
    history.undo().unwrap();
    assert_eq!(history.high_water(), 3);
    assert!(history.snapshot().parameters.get(3).is_none());
    history.redo().unwrap();
    assert!(history.snapshot().parameters.get(3).is_some());
}

#[test]
fn merge_previous_keeps_its_label_and_new_commit_discards_redo() {
    let mut history = history();
    let first = history.begin("first").unwrap();
    set_width(&mut history, &first, "20 mm");
    history.commit(&first, false).unwrap();
    let second = history.begin("second").unwrap();
    set_width(&mut history, &second, "30 mm");
    history.commit(&second, true).unwrap();
    assert_eq!(history.undo_labels(), ["first"]);
    history.undo().unwrap();
    assert_eq!(width(&history), 1.0);
    assert_eq!(history.redo_labels(), ["first"]);
    let replacement = history.begin("replacement").unwrap();
    set_width(&mut history, &replacement, "40 mm");
    history.commit(&replacement, false).unwrap();
    assert!(history.redo_labels().is_empty());
    assert!(history.redo().is_err());
}

#[test]
fn property_and_parameter_snapshots_are_owned_and_generations_are_monotonic() {
    let mut history = history();
    let before = history.snapshot().clone();
    let token = history.begin("both").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm").unwrap();
            snapshot
                .properties
                .put(USER_DEFINED, "Title", VariantValue::String("part".into()))
        })
        .unwrap();
    assert_eq!(history.generation(), 1);
    assert!(before.properties.get(USER_DEFINED, "Title").is_none());
    assert_eq!(
        before
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1.0
    );
    history.commit(&token, false).unwrap();
    history.undo().unwrap();
    assert_eq!(history.generation(), 2);
    assert!(
        history
            .snapshot()
            .properties
            .get(USER_DEFINED, "Title")
            .is_none()
    );
    history.redo().unwrap();
    assert_eq!(history.generation(), 3);
    assert!(
        history
            .snapshot()
            .properties
            .get(USER_DEFINED, "Title")
            .is_some()
    );
    let aborted = history.begin("abort").unwrap();
    history.abort(&aborted).unwrap();
    assert_eq!(history.generation(), 4);
}

#[test]
fn stale_tokens_and_open_navigation_are_rejected_without_mutation() {
    let mut history = history();
    for error in [history.undo().unwrap_err(), history.redo().unwrap_err()] {
        assert!(error.to_string().contains("expected"));
    }
    let token = history.begin("open").unwrap();
    assert!(history.undo().unwrap_err().to_string().contains("open"));
    assert!(history.redo().is_err());
    history.abort(&token).unwrap();
    assert!(history.commit(&token, false).is_err());
    assert!(
        history
            .abort(&token)
            .unwrap_err()
            .to_string()
            .contains("active top")
    );
    let result = history.edit(&token, |_, _| Ok::<(), std::io::Error>(()));
    assert!(matches!(result, Err(EditError::History(_))));
    let error = result.unwrap_err();
    assert!(std::error::Error::source(&error).is_some());
    assert!(error.to_string().contains("active top"));
    assert_eq!(width(&history), 1.0);
}

#[test]
fn initial_allocator_must_cover_all_existing_recipe_ids() {
    let snapshot = history().snapshot().clone();
    assert!(DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).is_err());
}

#[test]
fn tokens_from_another_document_cannot_address_equal_sequence_numbers() {
    let mut first = history();
    let mut second = history();
    let foreign = first.begin("first").unwrap();
    let local = second.begin("second").unwrap();
    assert_ne!(foreign, local);
    assert_eq!(local, local.clone());
    let error = second.commit(&foreign, false).unwrap_err();
    assert_eq!(error, HistoryError::ForeignTransaction);
    assert!(error.to_string().contains("this history owner"));
    assert_eq!(
        second.abort(&foreign),
        Err(HistoryError::ForeignTransaction)
    );
    assert!(matches!(
        second.edit(&foreign, |_, _| Ok::<_, ()>(())),
        Err(EditError::History(HistoryError::ForeignTransaction))
    ));
    set_width(&mut second, &local, "20 mm");
    second.commit(&local, false).unwrap();
    assert_eq!(width(&first), 1.0);
    assert_eq!(width(&second), 2.0);
}

#[test]
fn edits_cannot_invent_identities_or_reuse_retired_reservations() {
    let mut history = history();
    let token = history.begin("insert").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        snapshot
            .parameters
            .insert_expression(99, "invented", USER_PARAM, "2")
    });
    let error = result.unwrap_err();
    assert!(error.to_string().contains("recipe ID 99"));
    assert_eq!(history.snapshot().parameters.ids(), [1]);
    history.edit(&token, |_, ids| ids.allocate()).unwrap();
    let generation = history.generation();
    let result = history.edit(&token, |snapshot, _| {
        snapshot
            .parameters
            .insert_expression(2, "reused", USER_PARAM, "2")
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::UnreservedIdentity(2)))
    ));
    assert_eq!(history.generation(), generation);
    assert_eq!(history.snapshot().parameters.ids(), [1]);
    assert_eq!(history.high_water(), 2);
}

#[test]
fn aborting_parent_discards_committed_children_and_preserves_redo_order() {
    let mut history = history();
    for (label, source) in [("first", "20 mm"), ("second", "30 mm"), ("third", "40 mm")] {
        let token = history.begin(label).unwrap();
        set_width(&mut history, &token, source);
        history.commit(&token, false).unwrap();
    }
    assert_eq!(history.undo_labels(), ["first", "second", "third"]);
    history.undo().unwrap();
    history.undo().unwrap();
    let parent = history.begin("discard").unwrap();
    let child = history.begin("child").unwrap();
    set_width(&mut history, &child, "50 mm");
    history.commit(&child, false).unwrap();
    history.abort(&parent).unwrap();
    assert_eq!(width(&history), 2.0);
    assert_eq!(history.redo_labels(), ["second", "third"]);
    assert_eq!(history.redo().unwrap(), "second");
    assert_eq!(width(&history), 3.0);
    assert_eq!(history.redo().unwrap(), "third");
    assert_eq!(width(&history), 4.0);
}

#[test]
fn unchanged_frame_discard_checks_ownership_and_keeps_changed_frames_open() {
    let mut history = history();
    let snapshot = history.snapshot().clone();
    let mut foreign =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    let token = history.begin("local").unwrap();
    let wrong = foreign.begin("foreign").unwrap();
    assert_eq!(
        history.discard_unchanged(&wrong),
        Err(HistoryError::ForeignTransaction)
    );
    set_width(&mut history, &token, "2 cm");
    assert!(!history.discard_unchanged(&token).unwrap());
    set_width(&mut history, &token, "10 mm");
    let generation = history.generation();
    assert!(history.discard_unchanged(&token).unwrap());
    assert_eq!(history.generation(), generation);
    assert!(history.commit(&token, false).is_err());
    assert!(history.undo_labels().is_empty());
}

#[test]
fn save_markers_follow_steps_across_navigation_and_branch_replacement() {
    let mut history = history();
    history.mark_saved().unwrap();
    assert!(history.entries().is_empty());
    for (label, source) in [("first", "2 cm"), ("second", "3 cm")] {
        let token = history.begin(label).unwrap();
        set_width(&mut history, &token, source);
        history.commit(&token, false).unwrap();
        history.mark_saved().unwrap();
    }
    history.undo().unwrap();
    let entries = history.entries();
    assert_eq!(
        entries
            .iter()
            .map(|entry| entry.label.as_str())
            .collect::<Vec<_>>(),
        ["first", "second"]
    );
    assert!(entries.iter().all(|entry| entry.saved));
    history.redo().unwrap();
    assert_eq!(history.entries(), entries);
    history.undo().unwrap();
    let token = history.begin("replacement").unwrap();
    set_width(&mut history, &token, "4 cm");
    history.commit(&token, false).unwrap();
    let entries = history.entries();
    assert_eq!(entries[1].label, "replacement");
    assert!(entries[0].saved && !entries[1].saved);
}

#[test]
fn merged_edits_invalidate_save_markers_and_open_groups_cannot_be_marked() {
    let mut history = history();
    let token = history.begin("first").unwrap();
    set_width(&mut history, &token, "2 cm");
    history.commit(&token, false).unwrap();
    history.mark_saved().unwrap();
    history.mark_saved().unwrap();
    let token = history.begin("merge").unwrap();
    set_width(&mut history, &token, "3 cm");
    assert!(history.mark_saved().is_err());
    assert!(history.entries()[0].saved);
    history.commit(&token, true).unwrap();
    assert_eq!(history.entries().len(), 1);
    assert!(!history.entries()[0].saved);
}
