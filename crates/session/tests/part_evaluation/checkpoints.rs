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
