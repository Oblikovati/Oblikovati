// SPDX-License-Identifier: GPL-2.0-only
//! Unchanged sketch recomputation can publish its matching part result.
use super::*;

fn settled_replay(session: &mut PartSession) -> CompletedPartSketchRecompute<f64, &'static str> {
    let initial = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut FakePrismEvaluator::default(), Default::default())
        .unwrap();
    session.adopt_recomputed_part_sketches(initial).unwrap();
    session
        .prepare_part_evaluation()
        .recompute_sketches(&mut FakePrismEvaluator::default(), Default::default())
        .unwrap()
}

#[test]
fn unchanged_sketch_replay_publishes_the_current_part() {
    let mut session = session(7);
    let replay = settled_replay(&mut session);
    let generation = session.generation();
    let mut slot = PartSlot::default();
    assert!(session.adopt_sketch_replay(replay, &mut slot).unwrap());
    assert_eq!(session.generation(), generation);
    assert_eq!(current_depth(&slot, &session), 2.);
}

#[test]
fn stale_sketch_replay_never_replaces_the_current_slot() {
    let mut session = session(7);
    let replay = settled_replay(&mut session);
    let mut slot = PartSlot::default();
    publish(&mut session, &mut slot);
    session
        .create_sketch_prism("Second extrusion", fixture::selected_prism_definition())
        .unwrap();
    assert!(matches!(
        session.adopt_sketch_replay(replay, &mut slot),
        Err(PartSketchAdoptionError::Evaluation(
            PartEvaluationError::Superseded { .. }
        ))
    ));
    assert!(!slot.latest().unwrap().is_current);
}

#[test]
fn unchanged_replay_cannot_replace_a_foreign_document_slot() {
    let mut first = session(7);
    let mut second = session(8);
    let mut slot = PartSlot::default();
    publish(&mut first, &mut slot);
    let replay = settled_replay(&mut second);
    assert!(matches!(
        second.adopt_sketch_replay(replay, &mut slot),
        Err(PartSketchAdoptionError::Evaluation(
            PartEvaluationError::ForeignDocument { .. }
        ))
    ));
    assert_eq!(current_depth(&slot, &first), 2.);
}

#[test]
fn retained_recipe_reports_dirty_changes_only_for_its_history() {
    let mut owner = session(7);
    let mut slot = PartSlot::default();
    assert!(matches!(
        slot.replay_change(&owner),
        Err(PartEvaluationError::Unavailable { .. })
    ));
    publish(&mut owner, &mut slot);
    assert_eq!(slot.replay_change(&owner).unwrap(), None);
    edit_depth(&mut owner, "3 cm");
    assert_eq!(
        slot.replay_change(&owner).unwrap(),
        Some(oblikovati_model::feature::FeatureReplayRange { start: 0, end: 1 })
    );
    assert!(slot.current(&owner).is_err());
    owner.undo().unwrap();
    assert_eq!(slot.replay_change(&owner).unwrap(), None);
    assert!(slot.current(&owner).is_err());
    assert!(matches!(
        slot.replay_change(&session(8)),
        Err(PartEvaluationError::ForeignDocument { .. })
    ));
    assert!(matches!(
        slot.replay_change(&session(7)),
        Err(PartEvaluationError::ForeignHistory { .. })
    ));
}

#[test]
fn retained_recipe_reports_a_later_feature_change() {
    use oblikovati_model::feature::{
        AuthoredFeature, FeatureReplayRange, work_plane::WorkPlaneDefinition,
    };
    let mut document = fixture::selected_prism_document();
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            FeatureId(2),
            "Later plane",
            WorkPlaneDefinition::Fixed(oblikovati_model::sketch::SketchPlane::xy()),
        ))
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    let mut owner = DocumentSession::new(
        7,
        DocumentEditHistory::new(document, ids).unwrap(),
        RecordingPartObserver::default(),
    );
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    owner
        .run("Suppress later plane", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(FeatureId(2), true)
        })
        .unwrap();
    assert_eq!(
        slot.replay_change(&owner).unwrap(),
        Some(FeatureReplayRange { start: 1, end: 2 })
    );
    assert!(slot.current(&owner).is_err());
}

#[test]
fn fillet_creation_marks_only_its_new_feature_as_dirty() {
    use oblikovati_model::feature::FeatureReplayRange;
    let mut owner = session(7);
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let feature = owner.create_fillet("Fillet", fillet_input()).unwrap();
    assert_eq!(feature, FeatureId(2));
    assert_eq!(
        slot.replay_change(&owner).unwrap(),
        Some(FeatureReplayRange { start: 1, end: 2 })
    );
    assert!(slot.current(&owner).is_err());
    owner.undo().unwrap();
    assert_eq!(slot.replay_change(&owner).unwrap(), None);
    owner.redo().unwrap();
    assert_eq!(
        slot.replay_change(&owner).unwrap(),
        Some(FeatureReplayRange { start: 1, end: 2 })
    );
}

#[test]
fn candidate_tail_requires_the_current_document_history() {
    use oblikovati_model::feature::{AuthoredFeature, part::PartFeatureDefinition};
    let mut owner = session(7);
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let mut candidate = owner.snapshot().clone();
    let definition = fillet_input();
    let reads = definition.reads();
    candidate
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    assert!(
        slot.preview_candidate_tail(&owner, &candidate, &mut FakePrismEvaluator::default())
            .is_ok()
    );
    assert!(
        slot.preview_candidate_tail(&session(7), &candidate, &mut FakePrismEvaluator::default())
            .is_err()
    );
    edit_depth(&mut owner, "3 cm");
    assert!(
        slot.preview_candidate_tail(&owner, &candidate, &mut FakePrismEvaluator::default())
            .is_err()
    );
}
