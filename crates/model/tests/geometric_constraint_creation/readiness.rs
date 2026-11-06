// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn parallel_seed_for_perpendicular_is_retained_as_a_named_refusal_corpus_case() {
    let mut history = owner(fixture());
    let before = history.snapshot().sketches.clone();
    let cause = caught_failure(
        &mut history,
        &direct(PlanarConstraint::Lines {
            lines: [entity(7), entity(8)],
            relation: LineRelation::Perpendicular,
        }),
    );
    assert!(cause.contains("expected Converged"), "{cause}");
    assert!(history.snapshot().sketches.same_evaluated_state(&before));
    assert_eq!(history.sketch_reservations().constraints[&1], 1);
}
#[test]
fn exhausted_iterations_return_a_typed_readiness_cause_before_adoption() {
    let mut history = owner(fixture());
    let token = history.begin("Iteration limit").unwrap();
    let before = history.snapshot().sketches.clone();
    let error = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_geometric_constraint(
                DocumentSketchId(1),
                &direct(PlanarConstraint::Coincident {
                    points: [entity(2), entity(4)],
                }),
                SolveOptions {
                    max_iterations: 0,
                    ..Default::default()
                },
                ids,
            )
        })
        .unwrap_err();
    assert!(error.to_string().contains("IterationLimit"), "{error}");
    assert!(history.snapshot().sketches.same_evaluated_state(&before));
    history.abort(&token).unwrap();
    assert_eq!(history.sketch_reservations().constraints[&1], 1);
}
