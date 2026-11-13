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
            assembly: None,
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
fn assembly_occurrence_changes_follow_undo_and_redo() {
    let mut history = empty_history();
    let before = history.snapshot().clone();
    let token = history.begin("Place occurrence").unwrap();
    history
        .edit(&token, |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "Part:1".into(),
                component: "part.opd".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    history.commit(&token, true).unwrap();
    assert!(!before.same_metadata_recipe(history.snapshot()));
    assert_eq!(
        history.snapshot().assembly.as_ref().unwrap()[0].name,
        "Part:1"
    );
    history.undo().unwrap();
    assert!(history.snapshot().assembly.is_none());
    history.redo().unwrap();
    assert_eq!(
        history.snapshot().assembly.as_ref().unwrap()[0].component,
        "part.opd"
    );
}

#[test]
fn occurrence_ids_are_reserved_across_abort_preview_and_undo() {
    let mut history = empty_history();
    let first = history.begin("Rejected placement").unwrap();
    history
        .edit(&first, |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "First".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    history.abort(&first).unwrap();
    assert_eq!(history.occurrence_high_water(), 1);
    let preview = history
        .preview(|_, ids| -> Result<u64, ()> { Ok(ids.allocate_occurrence().unwrap()) })
        .unwrap();
    assert_eq!(preview, 2);
    assert_eq!(history.occurrence_high_water(), 1);
    let second = history.begin("Place").unwrap();
    history
        .edit(&second, |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "Second".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    history.commit(&second, false).unwrap();
    assert_eq!(
        history.snapshot().assembly.as_ref().unwrap()[0].session_id,
        2
    );
    history.undo().unwrap();
    let retry = history.begin("Retry").unwrap();
    let rejected = history.edit(&retry, |snapshot, _| -> Result<(), ()> {
        snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
            session_id: 1,
            ..Default::default()
        }]);
        Ok(())
    });
    assert!(rejected.is_err());
    history
        .edit(&retry, |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "Third".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    history.commit(&retry, false).unwrap();
    assert_eq!(
        history.snapshot().assembly.as_ref().unwrap()[0].session_id,
        3
    );
}

#[test]
fn duplicate_and_unreserved_occurrence_ids_cannot_enter_history() {
    let mut history = empty_history();
    let token = history.begin("Place").unwrap();
    history
        .edit(&token, |snapshot, ids| -> Result<(), ()> {
            snapshot.assembly = Some(vec![crate::assembly::AssemblyOccurrenceRecipe {
                session_id: ids.allocate_occurrence().unwrap(),
                name: "A".into(),
                ..Default::default()
            }]);
            Ok(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let before = history.snapshot().clone();
    let duplicate = history.begin("Duplicate ID").unwrap();
    let result = history.edit(&duplicate, |snapshot, _| -> Result<(), ()> {
        let rows = snapshot.assembly.as_mut().unwrap();
        rows.push(rows[0].clone());
        Ok(())
    });
    assert!(result.is_err());
    history.abort(&duplicate).unwrap();
    let result = history.refresh(|snapshot| -> Result<(), ()> {
        snapshot.assembly.as_mut().unwrap()[0].session_id = 99;
        Ok(())
    });
    assert!(result.is_err());
    assert_eq!(history.snapshot().assembly, before.assembly);
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
fn model_state_override_cannot_reference_a_missing_feature() {
    let mut history = empty_history();
    let token = history.begin("Invalid feature override").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        let mut members = crate::model_state::ModelStateMembers::default();
        members.set_suppressed(42, false, true);
        snapshot.model_states = Some(members);
        Ok::<_, String>(())
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::ModelStateMembers(_)))
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
