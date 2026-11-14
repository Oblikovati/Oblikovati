// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchReservationMarks;
use std::error::Error;
fn attempt(
    history: &mut DocumentEditHistory,
    request: &SketchLineChainCreation,
    sketch: DocumentSketchId,
    policy: SketchInferencePolicy,
) -> String {
    let token = history.begin("Catch chain failure").unwrap();
    let cause = history
        .edit(&token, |snapshot, ids| -> Result<String, String> {
            let before = snapshot.sketches.clone();
            let error = snapshot
                .create_inferred_sketch_line_chain(sketch, request, policy, Default::default(), ids)
                .unwrap_err();
            assert_eq!(snapshot.sketches, before);
            assert!(error.source().is_some());
            Ok(error.to_string())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    cause
}
fn policy(history: &DocumentEditHistory) -> SketchInferencePolicy {
    SketchInferencePolicy::for_units(history.snapshot().parameters.units())
}
#[test]
fn malformed_chain_vertices_are_refused_before_reserving_any_identity() {
    for points in [
        vec![],
        vec![[0.0, 0.0]],
        vec![[0.0, 0.0], [f64::NAN, 0.0]],
        vec![[0.0, 0.0], [0.0, f64::INFINITY]],
    ] {
        let mut history = owner(None);
        let before = history.sketch_reservations();
        let policy = policy(&history);
        let cause = attempt(
            &mut history,
            &input(&points, true),
            DocumentSketchId(1),
            policy,
        );
        assert!(cause.contains("finite"), "{cause}");
        assert_eq!(history.sketch_reservations(), before);
    }
}
#[test]
fn unknown_sketch_and_invalid_policy_do_not_reserve_identities() {
    let mut history = owner(None);
    let valid = policy(&history);
    let before = history.sketch_reservations();
    let request = input(&[[0.0, 0.0], [2.0, 1.0]], false);
    let invalid = SketchInferencePolicy {
        snap_distance_working: 0.0,
        ..valid
    };
    assert!(attempt(&mut history, &request, DocumentSketchId(99), valid).contains("99"));
    assert!(
        attempt(&mut history, &request, DocumentSketchId(1), invalid).contains("positive finite")
    );
    assert_eq!(history.sketch_reservations(), before);
}
#[test]
fn exhausted_second_segment_retires_reserved_ids_without_partial_chain_adoption() {
    let marks = SketchReservationMarks {
        sketch: 1,
        entity: u64::MAX - 4,
        ..Default::default()
    };
    let mut history = owner(None).with_sketch_reservations(marks).unwrap();
    let policy = policy(&history);
    let request = input(&[[10.0, 10.0], [13.0, 11.0], [15.0, 14.0]], false);
    assert!(attempt(&mut history, &request, DocumentSketchId(1), policy).contains("identity"));
    assert_eq!(history.sketch_reservations().entity, u64::MAX);
    assert_eq!(geometry(&history).points().len(), 3);
}
#[test]
fn exhausted_snap_constraint_retires_point_and_line_ids_without_adoption() {
    let marks = SketchReservationMarks {
        sketch: 1,
        entity: 4,
        constraints: [(1, u64::MAX)].into(),
        dimension_names: Default::default(),
    };
    let mut history = owner(None).with_sketch_reservations(marks).unwrap();
    let policy = policy(&history);
    let request = input(&[[0.0, 0.0], [3.0, 2.0]], false);
    assert!(attempt(&mut history, &request, DocumentSketchId(1), policy).contains("constraint"));
    assert_eq!(history.sketch_reservations().entity, 7);
    assert_eq!(geometry(&history).lines().len(), 1);
}
#[test]
fn undo_then_new_chain_never_reuses_retired_vertices_or_relations() {
    let mut history = owner(None);
    let request = input(&[[0.0, 0.0], [4.0, 0.0], [4.0, 3.0]], false);
    let first = create(&mut history, &request);
    let marks = history.sketch_reservations();
    history.undo().unwrap();
    let second = create(&mut history, &request);
    assert!(second.points.iter().all(|point| point.0 > marks.entity));
    assert!(
        second.lines[0].direction.unwrap().constraint
            > first.lines[1].direction.unwrap().constraint
    );
}
