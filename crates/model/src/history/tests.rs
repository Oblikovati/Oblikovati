// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::units::UnitsOfMeasure;

fn empty_history() -> DocumentEditHistory {
    DocumentEditHistory::new(
        DocumentSnapshot {
            work_geometry: Default::default(),
            sketches: Default::default(),
            attributes: Default::default(),
            attachments: Default::default(),
            interests: Default::default(),
            resources: Default::default(),
            assets: Default::default(),
            model_states: None,
            drawing: None,
            body_names: Default::default(),
            design_views: Default::default(),
            sketch_settings: None,
            parameters: ParameterCollection::new(UnitsOfMeasure::default()),
            properties: DocumentProperties::default(),
        },
        RecipeIdAllocator::default(),
    )
    .unwrap()
}

#[test]
fn exhausted_tokens_do_not_open_another_frame() {
    let mut history = empty_history();
    history.next_token = u64::MAX - 1;
    let last = history.begin("last").unwrap();
    let error = history.begin("overflow").unwrap_err();
    assert_eq!(error, HistoryError::Exhausted("transaction tokens"));
    assert!(error.to_string().contains("u64::MAX"));
    history.commit(&last, false).unwrap();
    assert_eq!(history.undo_labels(), ["last"]);
}

#[test]
fn exhausted_generation_refuses_edit_and_abort_without_calling_mutation() {
    let mut history = empty_history();
    let token = history.begin("open").unwrap();
    history.generation.store(u64::MAX, Ordering::Release);
    let result = history.edit(&token, |_, _| -> Result<(), ()> {
        panic!("exhausted generation must not invoke command")
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::Exhausted(_)))
    ));
    assert!(matches!(
        history.abort(&token),
        Err(HistoryError::Exhausted(_))
    ));
    assert_eq!(history.open.len(), 1);
    assert_eq!(history.high_water(), 0);
    history.commit(&token, false).unwrap();
    assert_eq!(history.generation(), u64::MAX);
}

#[test]
fn exhausted_generation_preserves_both_navigation_stacks() {
    let mut history = empty_history();
    let token = history.begin("step").unwrap();
    history.commit(&token, true).unwrap();
    history.generation.store(u64::MAX, Ordering::Release);
    assert!(matches!(history.undo(), Err(HistoryError::Exhausted(_))));
    assert_eq!(history.undo_labels(), ["step"]);
    history.generation.store(u64::MAX - 1, Ordering::Release);
    history.undo().unwrap();
    assert!(matches!(history.redo(), Err(HistoryError::Exhausted(_))));
    assert_eq!(history.redo_labels(), ["step"]);
    assert!(history.undo_labels().is_empty());
}

#[test]
fn model_state_history_keeps_primary_and_never_reuses_undone_identity() {
    let mut history = empty_history();
    let initialize = history.begin("Initialize members").unwrap();
    history
        .edit(&initialize, |snapshot, _| {
            snapshot.model_states = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    history.commit(&initialize, false).unwrap();
    let token = history.begin("Create member").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            let id = ids
                .allocate_model_state()
                .map_err(|cause| cause.to_string())?;
            let members = snapshot.model_states.as_mut().unwrap();
            members.insert(id, "Large")?;
            members.activate(id)?;
            Ok::<_, String>(id)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    history.undo().unwrap();
    assert_eq!(
        history
            .snapshot()
            .model_states
            .as_ref()
            .unwrap()
            .list()
            .len(),
        1
    );
    history.redo().unwrap();
    assert_eq!(
        history
            .snapshot()
            .model_states
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .id,
        id
    );
    history.undo().unwrap();
    let token = history.begin("Replace undone member").unwrap();
    let next = history
        .edit(&token, |snapshot, ids| {
            let next = ids
                .allocate_model_state()
                .map_err(|cause| cause.to_string())?;
            snapshot
                .model_states
                .as_mut()
                .unwrap()
                .insert(next, "Other")?;
            Ok::<_, String>(next)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(next > id);
}

#[test]
fn unreserved_model_state_identity_is_refused_without_adopting_candidate() {
    let mut history = empty_history();
    let token = history.begin("Unreserved member").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        let mut members = crate::model_state::ModelStateMembers::default();
        members.insert(2, "Invalid")?;
        snapshot.model_states = Some(members);
        Ok::<_, String>(())
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::UnreservedIdentity(2)))
    ));
    assert!(history.snapshot().model_states.is_none());
}

#[test]
fn owned_part_members_cannot_be_dropped_from_history() {
    let mut history = empty_history();
    let token = history.begin("Initialize part").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.model_states = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let token = history.begin("Drop members").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        snapshot.model_states = None;
        Ok::<_, String>(())
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::ModelStateMembers(_)))
    ));
    assert!(history.snapshot().model_states.is_some());
}
