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

#[test]
fn transaction_validates_tail_against_source_and_rejects_stale_prefix() {
    let mut owner = session(7);
    owner
        .run("Hide source sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let feature = owner
        .run_projected_with_view(
            "Second prism",
            |candidate, ids| {
                candidate
                    .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                    .map_err(|error| error.to_string())
            },
            |candidate, feature, source| {
                let count = prefix.validate_tail(
                    source,
                    candidate,
                    feature,
                    &mut FakePrismEvaluator::default(),
                )?;
                assert_eq!(count, Some(2));
                Ok::<_, String>(feature)
            },
        )
        .unwrap();
    assert_eq!(feature, FeatureId(2));
    let generation = owner.generation();
    let result = owner.run_projected_with_view(
        "Stale prefix",
        |candidate, ids| {
            candidate
                .create_sketch_prism("Third", fixture::selected_prism_definition(), ids)
                .map_err(|error| error.to_string())
        },
        |candidate, feature, source| {
            prefix
                .validate_tail(
                    source,
                    candidate,
                    feature,
                    &mut FakePrismEvaluator::default(),
                )
                .map(|_| ())
        },
    );
    assert!(result.is_err());
    assert_eq!(owner.generation(), generation);
    assert_eq!(owner.snapshot().work_geometry.features().len(), 2);
}

#[test]
fn candidate_prefix_declines_reuse_after_a_prefix_input_change() {
    use oblikovati_model::feature::{AuthoredFeature, part::PartFeatureDefinition};
    let owner = session(7);
    let mut slot = PartSlot::default();
    let completed = complete_fake(&owner);
    let mut owner = owner;
    owner.adopt_part_evaluation(completed, &mut slot).unwrap();
    let prefix = slot.candidate_prefix(&owner).unwrap();
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
    candidate.parameters.set_expression(3, "3 cm").unwrap();
    assert_eq!(
        prefix
            .validate_tail(
                owner.history(),
                &candidate,
                FeatureId(2),
                &mut FakePrismEvaluator::default()
            )
            .unwrap(),
        None
    );
}

#[test]
fn accepted_edit_transition_matches_only_its_source_and_current_history() {
    let mut owner = session(7);
    let source = owner.history().evaluation_scope();
    let (value, transition) = owner
        .run_projected_with_transition(
            "Depth",
            |candidate, _| candidate.parameters.set_expression(3, "3 cm"),
            |_, value, view| {
                assert!(source.is_current(view));
                Ok(value)
            },
        )
        .unwrap();
    assert_eq!(value, ());
    assert!(transition.matches(&source, owner.history()));
    assert_eq!(transition.target_generation(), owner.generation());
    let foreign = session(7);
    assert!(!transition.matches(&foreign.history().evaluation_scope(), owner.history()));
    assert!(!transition.matches(&source, foreign.history()));
    owner.undo().unwrap();
    assert!(!transition.matches(&source, owner.history()));
    owner.redo().unwrap();
    assert!(!transition.matches(&source, owner.history()));
}

#[test]
fn rejected_edit_returns_no_generation_transition() {
    let mut owner = session(7);
    let source = owner.history().evaluation_scope();
    let result = owner.run_projected_with_transition(
        "Rejected",
        |candidate, _| {
            candidate
                .parameters
                .set_expression(3, "3 cm")
                .map_err(|error| error.to_string())
        },
        |_, _, _| Err::<(), _>("rejected".to_owned()),
    );
    assert!(result.is_err());
    assert!(source.is_current(owner.history()));
}

#[test]
fn aborting_a_group_invalidates_its_accepted_edit_transition() {
    let mut owner = session(7);
    let source = owner.history().evaluation_scope();
    owner.begin("Group").unwrap();
    let (_, transition) = owner
        .run_projected_with_transition(
            "Depth",
            |candidate, _| candidate.parameters.set_expression(3, "3 cm"),
            |_, value, _| Ok(value),
        )
        .unwrap();
    assert!(transition.matches(&source, owner.history()));
    owner.abort().unwrap();
    assert!(!transition.matches(&source, owner.history()));
}

#[test]
fn candidate_result_matches_only_the_exact_accepted_edit() {
    let mut owner = session(7);
    owner
        .run("Hide source sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let (candidate, transition) = owner
        .run_projected_with_transition(
            "Second prism",
            |snapshot, ids| {
                snapshot
                    .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                    .map_err(|error| error.to_string())
            },
            |snapshot, feature, source| {
                prefix
                    .prepare_tail(
                        source,
                        snapshot,
                        feature,
                        &mut FakePrismEvaluator::default(),
                    )
                    .map(|result| result.unwrap())
            },
        )
        .unwrap();
    assert_eq!(candidate.body_count(), 2);
    assert!(candidate.matches_transition(&transition, owner.history()));
    owner.undo().unwrap();
    assert!(!candidate.matches_transition(&transition, owner.history()));
    owner.redo().unwrap();
    assert!(!candidate.matches_transition(&transition, owner.history()));
}

#[test]
fn candidate_result_rejects_a_different_edit_from_the_same_source() {
    let mut owner = session(7);
    owner
        .run("Hide source sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let candidate = owner
        .preview_projected(
            |snapshot, ids| {
                snapshot
                    .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                    .map_err(|error| error.to_string())
            },
            |snapshot, feature| {
                prefix
                    .prepare_tail(
                        owner.history(),
                        snapshot,
                        feature,
                        &mut FakePrismEvaluator::default(),
                    )
                    .map(|result| result.unwrap())
            },
        )
        .unwrap();
    let (_, transition) = owner
        .run_projected_with_transition(
            "Different edit",
            |snapshot, _| snapshot.parameters.set_expression(3, "3 cm"),
            |_, value, _| Ok(value),
        )
        .unwrap();
    assert!(!candidate.matches_transition(&transition, owner.history()));
}

#[path = "capture.rs"]
mod capture;
