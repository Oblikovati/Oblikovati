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
