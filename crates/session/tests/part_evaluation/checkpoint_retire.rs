// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn dropped_checkpoint_slot_retires_shared_prefix_and_pending_candidate() {
    let owner = two_prism_session();
    let mut slot = PartCheckpointSlot::default();
    let completed = owner
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    slot.adopt(&owner, completed).unwrap();
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let mut candidate = owner.snapshot().clone();
    candidate
        .work_geometry
        .set_suppressed(FeatureId(2), true)
        .unwrap();
    let pending = prefix
        .prepare_checkpoints(
            owner.history(),
            &candidate,
            &mut FakePrismEvaluator::default(),
        )
        .unwrap()
        .unwrap();
    drop(slot);
    assert!(
        pending
            .capture(owner.history(), |_, body| Ok(*body))
            .is_err()
    );
    assert!(
        prefix
            .prepare_checkpoints(
                owner.history(),
                &candidate,
                &mut FakePrismEvaluator::default()
            )
            .is_err()
    );
}
