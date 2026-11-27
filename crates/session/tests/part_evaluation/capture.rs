// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn committed_capture(
    owner: &mut PartSession,
    slot: &mut PartSlot,
) -> (
    oblikovati_session::part_evaluation::CapturedPartCandidate<f64, &'static str>,
    oblikovati_model::history::EvaluationTransition,
) {
    owner
        .run("Hide source sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    publish(owner, slot);
    let prefix = slot.candidate_prefix(owner).unwrap();
    owner
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
                    )?
                    .unwrap()
                    .capture(source, |_, body| Ok(*body))
            },
        )
        .unwrap()
}

#[test]
fn accepted_capture_can_be_completed_and_adopted() {
    let mut owner = session(7);
    let mut slot = PartSlot::default();
    let (candidate, transition) = committed_capture(&mut owner, &mut slot);
    let completed = candidate
        .try_complete(&transition, &owner, |_, body, _| Ok(body))
        .unwrap();
    owner.adopt_part_evaluation(completed, &mut slot).unwrap();
    assert_eq!(slot.current(&owner).unwrap().ordered_bodies().count(), 2);
    owner.undo().unwrap();
    assert!(slot.current(&owner).is_err());
    owner.redo().unwrap();
    assert!(slot.current(&owner).is_err());
}

#[test]
fn completion_rejects_foreign_or_stale_session_before_restoring() {
    for foreign in [true, false] {
        let mut owner = session(7);
        let mut slot = PartSlot::default();
        let (candidate, transition) = committed_capture(&mut owner, &mut slot);
        if !foreign {
            owner.undo().unwrap();
        }
        let other = session(8);
        let target = if foreign { &other } else { &owner };
        let mut called = false;
        assert!(
            candidate
                .try_complete(&transition, target, |_, body, _| {
                    called = true;
                    Ok(body)
                })
                .is_err()
        );
        assert!(!called);
        assert!(!slot.latest().unwrap().is_current);
    }
}

#[test]
fn failed_body_restore_does_not_replace_the_published_slot() {
    let mut owner = session(7);
    let mut slot = PartSlot::default();
    let (candidate, transition) = committed_capture(&mut owner, &mut slot);
    let mut count = 0;
    let result = candidate.try_complete(&transition, &owner, |_, body, _| {
        count += 1;
        if count == 2 {
            return Err("restore refused".into());
        }
        Ok(body)
    });
    assert!(result.is_err());
    assert_eq!(count, 2);
    assert!(!slot.latest().unwrap().is_current);
}

#[test]
fn captured_candidate_survives_prefix_retirement_and_checks_its_transition() {
    let mut owner = session(7);
    owner
        .run("Hide source sketch", |snapshot, _| {
            snapshot.sketches.set_visible(fixture::SKETCH, false)
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let (captured, transition) = {
        let prefix = slot.candidate_prefix(&owner).unwrap();
        owner
            .run_projected_with_transition(
                "Second prism",
                |snapshot, ids| {
                    snapshot
                        .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                        .map_err(|error| error.to_string())
                },
                |snapshot, feature, source| {
                    let candidate = prefix
                        .prepare_tail(
                            source,
                            snapshot,
                            feature,
                            &mut FakePrismEvaluator::default(),
                        )?
                        .unwrap();
                    candidate.capture(source, |_, body| Ok(*body))
                },
            )
            .unwrap()
    };
    slot.clear();
    assert_eq!(captured.body_count(), 2);
    assert!(captured.matches_transition(&transition, owner.history()));
    owner.undo().unwrap();
    assert!(!captured.matches_transition(&transition, owner.history()));
}

#[test]
fn candidate_capture_rejects_backend_failure_and_a_stale_owner() {
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
    let mut captures = 0;
    assert_eq!(
        candidate
            .copy_current(owner.history())
            .unwrap()
            .ordered_bodies()
            .count(),
        2
    );
    assert!(
        candidate
            .capture(owner.history(), |_, body| {
                captures += 1;
                if captures == 2 {
                    return Err("capture refused".into());
                }
                Ok(*body)
            })
            .is_err()
    );
    assert_eq!(captures, 2);
    assert!(slot.current(&owner).is_ok());
    edit_depth(&mut owner, "3 cm");
    assert!(candidate.copy_current(owner.history()).is_err());
    let mut called = false;
    assert!(
        candidate
            .capture(owner.history(), |_, body| {
                called = true;
                Ok(*body)
            })
            .is_err()
    );
    assert!(!called);
}

#[test]
fn model_candidate_guard_tracks_the_exact_accepted_history_and_inputs() {
    use oblikovati_model::history::PartCandidateGuard;
    let mut owner = session(7);
    let source = owner.history().evaluation_scope();
    let foreign = session(7);
    assert!(
        PartCandidateGuard::capture(&source, foreign.history().snapshot(), foreign.history())
            .is_err()
    );
    let ((guard, wrong_inputs), transition) = owner
        .run_projected_with_transition(
            "Second prism",
            |snapshot, ids| {
                snapshot
                    .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                    .map_err(|error| error.to_string())
            },
            |snapshot, _, history| {
                Ok((
                    PartCandidateGuard::capture(&source, snapshot, history)?,
                    PartCandidateGuard::capture(&source, history.snapshot(), history)?,
                ))
            },
        )
        .unwrap();
    assert!(guard.matches(&transition, owner.history()));
    assert!(!wrong_inputs.matches(&transition, owner.history()));
    assert!(!guard.matches(&transition, foreign.history()));
    assert!(
        PartCandidateGuard::capture(&source, owner.history().snapshot(), owner.history()).is_err()
    );
    owner.undo().unwrap();
    assert!(!guard.matches(&transition, owner.history()));
}

#[test]
fn model_transfer_rejects_wrong_recipe_and_stale_completion() {
    use oblikovati_model::feature::part::CapturedPartTransfer;
    let mut owner = session(7);
    let scope = owner.history().evaluation_scope();
    let old = owner
        .history()
        .snapshot()
        .evaluate_part_features(&mut FakePrismEvaluator::default())
        .unwrap();
    let old_checkpoints = owner
        .history()
        .snapshot()
        .evaluate_part_with_checkpoints(&mut FakePrismEvaluator::default())
        .unwrap();
    let (captured, transition) = owner
        .run_projected_with_transition(
            "Second prism",
            |snapshot, ids| {
                snapshot
                    .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                    .map_err(|error| error.to_string())
            },
            |snapshot, _, history| {
                assert!(
                    CapturedPartTransfer::capture_result(7, &scope, snapshot, old, history)
                        .is_err()
                );
                assert!(
                    CapturedPartTransfer::capture_checkpoints(
                        7,
                        &scope,
                        snapshot,
                        old_checkpoints,
                        history,
                    )
                    .is_err()
                );
                let evaluated = snapshot
                    .evaluate_part_features(&mut FakePrismEvaluator::default())
                    .map_err(|error| error.to_string())?;
                CapturedPartTransfer::capture_result(7, &scope, snapshot, evaluated, history)
            },
        )
        .unwrap();
    let completed = captured
        .complete(7, &transition, owner.history(), |payload, _| Ok(payload))
        .unwrap();
    owner.undo().unwrap();
    assert!(completed.into_current(7, owner.history()).is_err());
}

#[test]
fn unchanged_capture_accepts_view_metadata_and_rejects_geometry_edits() {
    let mut owner = session(7);
    owner
        .run("Initialize views", |snapshot, _| {
            snapshot.part_design_views = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    let mut slot = PartSlot::default();
    publish(&mut owner, &mut slot);
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let (capture, transition) = owner
        .run_projected_with_transition(
            "View options",
            |snapshot, _| {
                snapshot.part_design_views.as_mut().unwrap().set_options(
                    &oblikovati_api::wire::SetPartDesignViewOptionsArgs {
                        id: 1,
                        published: Some(Box::new(false)),
                        ..Default::default()
                    },
                )
            },
            |candidate, (), history| {
                prefix.capture_unchanged(history, candidate, |_, body| Ok(*body))
            },
        )
        .unwrap();
    let completed = capture
        .try_complete(&transition, &owner, |_, body, _| Ok(body))
        .unwrap();
    owner.adopt_part_evaluation(completed, &mut slot).unwrap();
    assert_eq!(slot.current(&owner).unwrap().ordered_bodies().count(), 1);
    let prefix = slot.candidate_prefix(&owner).unwrap();
    let result = owner.run_projected_with_transition(
        "Add geometry",
        |snapshot, ids| {
            snapshot
                .create_sketch_prism("Second", fixture::selected_prism_definition(), ids)
                .map_err(|cause| cause.to_string())
        },
        |candidate, _, history| prefix.capture_unchanged(history, candidate, |_, body| Ok(*body)),
    );
    assert!(result.is_err());
    assert_eq!(slot.current(&owner).unwrap().ordered_bodies().count(), 1);
}
