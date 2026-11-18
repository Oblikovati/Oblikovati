// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn two_prism_session() -> PartSession {
    let mut owner = session(7);
    owner
        .run("Hide sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    owner
        .create_sketch_prism("Second", fixture::selected_prism_definition())
        .unwrap();
    owner
}

fn commit_second_suppression(
    owner: &mut PartSession,
    part: &mut PartSlot,
    checkpoints: &mut PartCheckpointSlot<f64, &'static str>,
    suppressed: bool,
) -> usize {
    let mut backend = FakePrismEvaluator::default();
    let (captured, transition) = {
        let prefix = checkpoints.candidate_prefix(owner).unwrap();
        owner
            .run_projected_with_transition(
                "Change second prism",
                |snapshot, _| {
                    snapshot
                        .work_geometry
                        .set_suppressed(FeatureId(2), suppressed)
                        .map(|_| ())
                        .map_err(|error| error.to_string())
                },
                |snapshot, _, history| {
                    prefix
                        .prepare_checkpoints(history, snapshot, &mut backend)?
                        .ok_or("expected changed checkpoints")?
                        .capture(history, |_, body| Ok(*body))
                },
            )
            .unwrap()
    };
    let completed = captured
        .try_complete(&transition, owner, |_, body, _| Ok(body))
        .unwrap();
    owner
        .adopt_replayed_checkpoints(completed, part, checkpoints)
        .unwrap();
    backend.calls
}

#[test]
fn accepted_checkpoint_sets_support_successive_session_edits() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let mut part = PartSlot::default();
    publish(&mut owner, &mut part);
    assert_eq!(
        commit_second_suppression(&mut owner, &mut part, &mut checkpoints, true),
        0
    );
    assert_eq!(part.current(&owner).unwrap().ordered_bodies().count(), 1);
    assert_eq!(
        commit_second_suppression(&mut owner, &mut part, &mut checkpoints, false),
        1
    );
    assert_eq!(part.current(&owner).unwrap().ordered_bodies().count(), 2);
    assert!(checkpoints.candidate_prefix(&owner).is_ok());
    owner.undo().unwrap();
    assert!(part.current(&owner).is_err());
    assert!(checkpoints.candidate_prefix(&owner).is_err());
}

#[test]
fn captured_checkpoint_completion_rejects_foreign_and_stale_owners() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let ((foreign, stale), transition) = {
        let prefix = checkpoints.candidate_prefix(&owner).unwrap();
        owner
            .run_projected_with_transition(
                "Suppress second",
                |snapshot, _| {
                    snapshot
                        .work_geometry
                        .set_suppressed(FeatureId(2), true)
                        .map(|_| ())
                        .map_err(|error| error.to_string())
                },
                |snapshot, _, history| {
                    let candidate = prefix
                        .prepare_checkpoints(history, snapshot, &mut FakePrismEvaluator::default())?
                        .ok_or("expected changed checkpoints")?;
                    Ok((
                        candidate.capture(history, |_, body| Ok(*body))?,
                        candidate.capture(history, |_, body| Ok(*body))?,
                    ))
                },
            )
            .unwrap()
    };
    let mut restored = false;
    assert!(
        foreign
            .try_complete(&transition, &session(8), |_, body, _| {
                restored = true;
                Ok(body)
            })
            .is_err()
    );
    assert!(!restored);
    owner.undo().unwrap();
    assert!(
        stale
            .try_complete(&transition, &owner, |_, body, _| {
                restored = true;
                Ok(body)
            })
            .is_err()
    );
    assert!(!restored);
}

#[test]
fn checkpoint_capture_failure_aborts_edit_and_keeps_published_results() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let mut part = PartSlot::default();
    publish(&mut owner, &mut part);
    let generation = owner.generation();
    let mut captures = 0;
    let result: Result<_, _> = {
        let prefix = checkpoints.candidate_prefix(&owner).unwrap();
        owner.run_projected_with_transition(
            "Suppress second",
            |snapshot, _| {
                snapshot
                    .work_geometry
                    .set_suppressed(FeatureId(2), true)
                    .map(|_| ())
                    .map_err(|error| error.to_string())
            },
            |snapshot, _, history| {
                let candidate = prefix
                    .prepare_checkpoints(history, snapshot, &mut FakePrismEvaluator::default())?
                    .ok_or("expected changed checkpoints")?;
                candidate.capture(history, |_, body| {
                    captures += 1;
                    if captures == 2 {
                        return Err("capture refused".into());
                    }
                    Ok(*body)
                })
            },
        )
    };
    assert!(result.is_err());
    assert_eq!(captures, 2);
    assert_eq!(owner.generation(), generation);
    assert!(part.current(&owner).is_ok());
    assert!(checkpoints.candidate_prefix(&owner).is_ok());
}

#[test]
fn checkpoint_completion_rejects_another_edit_from_the_same_source() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let prefix = checkpoints.candidate_prefix(&owner).unwrap();
    let candidate = owner
        .preview_projected(
            |snapshot, _| {
                snapshot
                    .work_geometry
                    .set_suppressed(FeatureId(2), true)
                    .map(|_| ())
                    .map_err(|error| error.to_string())
            },
            |snapshot, _| {
                prefix
                    .prepare_checkpoints(
                        owner.history(),
                        snapshot,
                        &mut FakePrismEvaluator::default(),
                    )?
                    .ok_or("expected changed checkpoints")?
                    .capture(owner.history(), |_, body| Ok(*body))
            },
        )
        .unwrap();
    let (_, transition) = owner
        .run_projected_with_transition(
            "Move cutoff",
            |snapshot, _| {
                snapshot
                    .work_geometry
                    .set_end_of_part(Some(1))
                    .map(|_| ())
                    .map_err(|error| error.to_string())
            },
            |_, _, _| Ok::<_, String>(()),
        )
        .unwrap();
    let mut called = false;
    assert!(
        candidate
            .try_complete(&transition, &owner, |_, body, _| {
                called = true;
                Ok(body)
            })
            .is_err()
    );
    assert!(!called);
}

#[test]
fn failed_checkpoint_restore_does_not_publish_partial_results() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let mut part = PartSlot::default();
    publish(&mut owner, &mut part);
    let source_generation = owner.generation();
    let (captured, transition) = {
        let prefix = checkpoints.candidate_prefix(&owner).unwrap();
        owner
            .run_projected_with_transition(
                "Suppress second",
                |snapshot, _| {
                    snapshot
                        .work_geometry
                        .set_suppressed(FeatureId(2), true)
                        .map(|_| ())
                        .map_err(|error| error.to_string())
                },
                |snapshot, _, history| {
                    prefix
                        .prepare_checkpoints(history, snapshot, &mut FakePrismEvaluator::default())?
                        .ok_or("expected changed checkpoints")?
                        .capture(history, |_, body| Ok(*body))
                },
            )
            .unwrap()
    };
    let mut calls = 0;
    let completed = captured.try_complete(&transition, &owner, |_, body, _| {
        calls += 1;
        if calls == 2 {
            return Err("restore refused".into());
        }
        Ok(body)
    });
    assert!(completed.is_err());
    assert_eq!(calls, 2);
    assert_eq!(part.latest().unwrap().generation, source_generation);
    assert!(!part.latest().unwrap().is_current);
    assert!(checkpoints.candidate_prefix(&owner).is_err());
}

#[test]
fn checkpoint_candidate_adopts_an_edited_tail_without_rebuilding_the_prefix() {
    let mut owner = two_prism_session();
    let mut checkpoints = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    checkpoints.adopt(&owner, completed).unwrap();
    let mut result = PartSlot::default();
    publish(&mut owner, &mut result);
    let mut backend = FakePrismEvaluator::default();
    let (captured, transition) = {
        let prefix = checkpoints.candidate_prefix(&owner).unwrap();
        owner
            .run_projected_with_transition(
                "Suppress second",
                |snapshot, _| {
                    snapshot
                        .work_geometry
                        .set_suppressed(FeatureId(2), true)
                        .map(|_| ())
                        .map_err(|error| error.to_string())
                },
                |snapshot, _, history| {
                    prefix
                        .prepare_candidate(history, snapshot, &mut backend)?
                        .ok_or("expected a changed tail")?
                        .capture(history, |_, body| Ok(*body))
                },
            )
            .unwrap()
    };
    assert_eq!(backend.calls, 0);
    let completed = captured
        .try_complete(&transition, &owner, |_, body, _| Ok(body))
        .unwrap();
    owner.adopt_part_evaluation(completed, &mut result).unwrap();
    assert_eq!(result.current(&owner).unwrap().ordered_bodies().count(), 1);
    assert!(checkpoints.candidate_prefix(&owner).is_err());
    owner.undo().unwrap();
    assert!(result.current(&owner).is_err());
    assert!(checkpoints.candidate_prefix(&owner).is_err());
}

#[test]
fn checkpoints_reject_foreign_or_superseded_generations_without_replacing_current() {
    let mut owner = session(7);
    let mut other = session(8);
    let mut slot = PartCheckpointSlot::default();
    let initial = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    slot.adopt(&owner, initial).unwrap();
    let foreign = other
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    assert!(matches!(
        slot.adopt(&owner, foreign),
        Err(PartEvaluationError::ForeignDocument { .. })
    ));
    assert!(slot.candidate_prefix(&owner).is_ok());
    let reopened = session(7);
    let foreign_history = reopened
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    assert!(matches!(
        slot.adopt(&owner, foreign_history),
        Err(PartEvaluationError::ForeignHistory { .. })
    ));
    assert!(matches!(
        slot.candidate_prefix(&reopened),
        Err(PartEvaluationError::ForeignHistory { .. })
    ));
    let stale = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    edit_depth(&mut owner, "3 cm");
    assert!(matches!(
        slot.adopt(&owner, stale),
        Err(PartEvaluationError::Superseded { .. })
    ));
    assert!(slot.candidate_prefix(&owner).is_err());
    edit_depth(&mut other, "3 cm");
    let fresh = other
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    assert!(matches!(
        slot.adopt(&other, fresh),
        Err(PartEvaluationError::ForeignDocument { .. })
    ));
}

#[test]
fn superseded_checkpoint_request_stops_before_kernel_evaluation() {
    let mut owner = session(7);
    let request = owner.prepare_part_evaluation();
    edit_depth(&mut owner, "3 cm");
    let mut backend = FakePrismEvaluator::default();
    assert!(matches!(
        request.evaluate_checkpoints(&mut backend),
        Err(PartEvaluationError::Replay(
            oblikovati_model::feature::part::PartFeatureProgramError::Cancelled { .. }
        ))
    ));
    assert_eq!(backend.calls, 0);
}

#[test]
fn checkpointed_sketch_replay_publishes_only_after_sketch_state_settles() {
    let mut document = fixture::selected_prism_document();
    let geometry = document
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry()
        .clone();
    document
        .sketches
        .replace_geometry(fixture::SKETCH, geometry)
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, document.parameters.ids()).unwrap();
    let mut owner = DocumentSession::new(
        7,
        DocumentEditHistory::new(document, ids).unwrap(),
        RecordingPartObserver::default(),
    );
    let mut slot = PartSlot::default();
    let mut checkpoints = PartCheckpointSlot::default();
    let first = owner
        .prepare_part_evaluation()
        .recompute_sketches_with_checkpoints(&mut FakePrismEvaluator::default(), Default::default())
        .unwrap();
    let first_published = owner
        .adopt_checkpointed_sketch_replay(first, &mut slot, &mut checkpoints)
        .unwrap();
    assert!(!first_published);
    assert!(slot.current(&owner).is_err());
    assert!(checkpoints.candidate_prefix(&owner).is_err());
    let settled = owner
        .prepare_part_evaluation()
        .recompute_sketches_with_checkpoints(&mut FakePrismEvaluator::default(), Default::default())
        .unwrap();
    assert!(
        owner
            .adopt_checkpointed_sketch_replay(settled, &mut slot, &mut checkpoints)
            .unwrap()
    );
    assert_eq!(current_depth(&slot, &owner), 2.);
    assert!(checkpoints.candidate_prefix(&owner).is_ok());
}

#[test]
fn stale_checkpointed_sketch_replay_does_not_replace_current_results() {
    let mut owner = session(7);
    let old = owner
        .prepare_part_evaluation()
        .recompute_sketches_with_checkpoints(&mut FakePrismEvaluator::default(), Default::default())
        .unwrap();
    edit_depth(&mut owner, "3 cm");
    let mut slot = PartSlot::default();
    let mut checkpoints = PartCheckpointSlot::default();
    assert!(matches!(
        owner.adopt_checkpointed_sketch_replay(old, &mut slot, &mut checkpoints),
        Err(PartSketchAdoptionError::Evaluation(
            PartEvaluationError::Superseded { .. }
        ))
    ));
    assert!(slot.current(&owner).is_err());
    assert!(checkpoints.candidate_prefix(&owner).is_err());
}
