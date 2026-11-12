// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_deletion_selection::{assert_mixed_cascade, selection_session};
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::*};
use oblikovati_session::selection::{SelectionClickMode, SketchPickTarget};
const SKETCH: DocumentSketchId = DocumentSketchId(1);
fn place_test_block(session: &mut DocumentSession<RecordingObserver>) -> SketchEntityId {
    use oblikovati_model::document_sketch::SketchBlockInstance;
    session
        .run("Place block", |snapshot, ids| {
            snapshot.sketches.define_empty_block("Rib").unwrap();
            let id = ids.allocate_sketch_entity().unwrap();
            let instance = SketchBlockInstance::new(id, "Rib", [0., 0.], 0., 1.).unwrap();
            snapshot
                .sketches
                .insert_block_instance(SKETCH, instance)
                .unwrap();
            Ok::<_, String>(id)
        })
        .unwrap()
}
#[test]
fn block_pick_deletes_only_the_instance_and_survives_undo_redo() {
    let mut session = selection_session(None);
    let id = place_test_block(&mut session);
    let target = SketchPickTarget::BlockInstance(id);
    assert!(session.sketch_pick(SKETCH, target).is_ok());
    assert!(
        session
            .sketch_pick(SKETCH, SketchPickTarget::BlockInstance(SketchEntityId(999)))
            .is_err()
    );
    session
        .set_sketch_selection(SKETCH, Some(target), SelectionClickMode::Replace, false)
        .unwrap();
    let before = session.snapshot().sketches.clone();
    let report = session
        .delete_editor_selection(Some(SKETCH), false)
        .unwrap();
    assert_eq!(report.entities, [id]);
    assert_eq!(session.snapshot().sketches.block_definitions().len(), 1);
    assert!(
        session
            .snapshot()
            .sketches
            .get(SKETCH)
            .unwrap()
            .block_instances()
            .is_empty()
    );
    assert!(session.selected_sketch_picks(SKETCH).is_empty());
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    session.redo().unwrap();
    assert!(
        session
            .snapshot()
            .sketches
            .get(SKETCH)
            .unwrap()
            .block_instances()
            .is_empty()
    );
}

#[test]
fn mixed_block_and_geometry_selection_is_one_atomic_history_step() {
    let mut session = selection_session(None);
    let id = place_test_block(&mut session);
    for target in [
        SketchPickTarget::Entity(SketchEntityId(2)),
        SketchPickTarget::BlockInstance(id),
    ] {
        session
            .set_sketch_selection(SKETCH, Some(target), SelectionClickMode::Toggle, false)
            .unwrap();
    }
    let before = session.snapshot().sketches.clone();
    let report = session
        .delete_editor_selection(Some(SKETCH), false)
        .unwrap();
    assert!(report.entities.contains(&SketchEntityId(2)) && report.entities.contains(&id));
    assert_eq!(
        session.undo_labels().last(),
        Some(&"Delete Sketch Entities")
    );
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
}

#[test]
fn vetoed_block_deletion_restores_the_instance_and_its_selection() {
    let mut seed = selection_session(None);
    let id = place_test_block(&mut seed);
    let mut session = super::sketch_dimension_delete::dimension_session_from_snapshot(
        seed.snapshot().clone(),
        Some(EVENT_TRANSACTION_COMMITTED),
    );
    let target = SketchPickTarget::BlockInstance(id);
    session
        .set_sketch_selection(SKETCH, Some(target), SelectionClickMode::Replace, false)
        .unwrap();
    let before = session.snapshot().sketches.clone();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.selected_sketch_picks(SKETCH)[0].target(), target);
}

#[test]
fn invalid_block_in_mixed_deletion_keeps_the_entire_document() {
    let session = selection_session(None);
    let mut snapshot = session.snapshot().clone();
    let before = snapshot.sketches.clone();
    let selection = SketchDeletionSelection {
        entities: vec![SketchEntityId(2)],
        ..Default::default()
    };
    assert!(
        snapshot
            .delete_sketch_selection_with_blocks(SKETCH, &selection, &[SketchEntityId(999)])
            .is_err()
    );
    assert!(snapshot.sketches.same_recipe(&before));
    assert!(
        snapshot
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
}
pub(super) fn select_owned_mixed(session: &mut DocumentSession<RecordingObserver>) {
    for target in [
        SketchPickTarget::Entity(SketchEntityId(2)),
        SketchPickTarget::Dimension(SketchConstraintId(1)),
        SketchPickTarget::Dimension(SketchConstraintId(3)),
        SketchPickTarget::GeometricConstraint(SketchConstraintId(2)),
    ] {
        session
            .set_sketch_selection(SKETCH, Some(target), SelectionClickMode::Toggle, false)
            .unwrap();
    }
}
#[test]
fn owned_deletion_clears_only_adopted_picks_and_has_one_undo_step() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let report = session
        .delete_editor_selection(Some(SKETCH), false)
        .unwrap();
    assert!(!report.entities.is_empty());
    assert_mixed_cascade(&session);
    assert!(session.selected_editor_picks().is_empty());
    session.undo().unwrap();
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert!(session.selected_editor_picks().is_empty());
    session.redo().unwrap();
    assert_mixed_cascade(&session);
}
#[test]
fn owned_deletion_veto_keeps_the_selection_and_recipe() {
    let mut session = selection_session(Some(EVENT_TRANSACTION_COMMITTED));
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let picks = session.selected_editor_picks();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .is_err()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    let retained = session.selected_editor_picks();
    assert_eq!(
        retained
            .iter()
            .map(|pick| pick.target())
            .collect::<Vec<_>>(),
        picks.iter().map(|pick| pick.target()).collect::<Vec<_>>()
    );
}
#[test]
fn owned_deletion_respects_editor_tool_and_empty_guards() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    for (sketch, active) in [(None, false), (Some(SKETCH), true)] {
        assert!(
            session
                .delete_editor_selection(sketch, active)
                .unwrap()
                .entities
                .is_empty()
        );
        assert_eq!(session.selected_editor_picks().len(), 4);
    }
    session.clear_transient_selection();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .unwrap()
            .entities
            .is_empty()
    );
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
}

#[test]
fn owned_deletion_does_not_rebind_stale_picks_after_an_unrelated_abort() {
    let mut session = selection_session(None);
    select_owned_mixed(&mut session);
    session.begin("Unrelated").unwrap();
    session.abort().unwrap();
    assert!(session.selected_editor_picks().is_empty());
    let before = session.snapshot().sketches.clone();
    let generation = session.generation();
    assert!(
        session
            .delete_editor_selection(Some(SKETCH), false)
            .is_err()
    );
    assert!(session.selected_editor_picks().is_empty());
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.generation(), generation);
}
