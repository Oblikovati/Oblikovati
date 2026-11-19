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
            part_design_views: None,
            sketch_settings: None,
            parameters: ParameterCollection::new(UnitsOfMeasure::default()),
            properties: DocumentProperties::default(),
        },
        RecipeIdAllocator::default(),
    )
    .unwrap()
}

#[test]
fn edit_view_tracks_source_generation_and_rejection_keeps_the_owner() {
    let mut history = empty_history();
    let scope = history.evaluation_scope();
    let token = history.begin("Rejected candidate").unwrap();
    let result: Result<(), _> = history.edit_with_view(&token, |candidate, _, view| {
        assert!(scope.is_current(view));
        candidate.design_views.push("Candidate".into());
        assert!(view.snapshot().design_views.is_empty());
        Err("rejected")
    });
    assert!(result.is_err());
    assert!(scope.is_current(&history));
    assert!(history.snapshot().design_views.is_empty());
    history
        .edit_with_view(&token, |candidate, _, view| {
            assert!(scope.is_current(view));
            candidate.design_views.push("Accepted".into());
            Ok::<(), ()>(())
        })
        .unwrap();
    assert!(scope.is_superseded());
    assert_eq!(history.snapshot().design_views, ["Accepted"]);
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
fn part_view_edits_replay_and_do_not_reuse_undone_ids() {
    let mut history = empty_history();
    let token = history.begin("Create view").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            let views = snapshot
                .part_design_views
                .get_or_insert_with(Default::default);
            let id = ids
                .allocate_part_design_view()
                .map_err(|cause| cause.to_string())?;
            views.copy(1, id, "Review")?;
            Ok::<_, String>(id)
        })
        .unwrap();
    history.commit(&token, true).unwrap();
    history.undo().unwrap();
    history.redo().unwrap();
    assert!(
        history
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .get(id)
            .is_ok()
    );
    history.undo().unwrap();
    let token = history.begin("Create replacement view").unwrap();
    let next = history
        .edit(&token, |snapshot, ids| {
            let views = snapshot
                .part_design_views
                .get_or_insert_with(Default::default);
            let next = ids
                .allocate_part_design_view()
                .map_err(|cause| cause.to_string())?;
            views.copy(1, next, "Replacement")?;
            Ok::<_, String>(next)
        })
        .unwrap();
    history.commit(&token, true).unwrap();
    assert!(next > id);
}

#[test]
fn unreserved_part_view_identity_is_refused() {
    let mut history = empty_history();
    let token = history.begin("Unreserved view").unwrap();
    let result = history.edit(&token, |snapshot, _| {
        let mut views = crate::part_design_view::PartDesignViews::default();
        views.copy(1, 6, "Invalid")?;
        snapshot.part_design_views = Some(views);
        Ok::<_, String>(())
    });
    assert!(matches!(
        result,
        Err(EditError::History(HistoryError::UnreservedIdentity(6)))
    ));
    assert!(history.snapshot().part_design_views.is_none());
}

#[test]
fn view_display_history_stays_on_its_own_step() {
    let mut history = empty_history();
    let transition = PartViewDisplayTransition {
        before: PartViewDisplayState {
            hidden_body_keys: Default::default(),
            camera: None,
        },
        after: PartViewDisplayState {
            hidden_body_keys: std::collections::BTreeSet::from(["aa".into()]),
            camera: None,
        },
    };
    let token = history.begin("Display").unwrap();
    history
        .tag_part_view_display(&token, transition.clone())
        .unwrap();
    assert!(!history.discard_unchanged(&token).unwrap());
    history.commit(&token, false).unwrap();
    assert_eq!(history.undo_part_view_display(), Some(&transition));
    history.undo().unwrap();
    assert_eq!(history.redo_part_view_display(), Some(&transition));
    history.redo().unwrap();
    let ordinary = history.begin("Ordinary").unwrap();
    history.commit(&ordinary, true).unwrap();
    assert!(history.undo_part_view_display().is_none());
    assert!(!history.jump_crosses_part_view(1));
    assert!(history.jump_crosses_part_view(0));
    history.undo().unwrap();
    assert_eq!(history.undo_part_view_display(), Some(&transition));
    history.undo().unwrap();
    assert!(history.jump_crosses_part_view(1));
    assert!(history.jump_crosses_part_view(2));
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

#[test]
fn snapshot_replay_reserves_model_state_ids_without_changing_snapshot() {
    let mut snapshot = empty_history().snapshot().clone();
    let mut members = crate::model_state::ModelStateMembers::default();
    members.insert(20, "Variant").unwrap();
    snapshot.model_states = Some(members);
    let replay = SnapshotReplay::new(&snapshot).unwrap();
    let replay = replay.history();
    assert_eq!(replay.high_water(), 20);
    assert_eq!(replay.snapshot().model_states, snapshot.model_states);
    assert_eq!(
        replay.snapshot().parameters.ids(),
        snapshot.parameters.ids()
    );
}
