// SPDX-License-Identifier: GPL-2.0-only
use super::*;

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
