// SPDX-License-Identifier: GPL-2.0-only
//! Policy-specific replay retention and eviction.
use super::*;

#[test]
fn document_replay_retains_independent_policy_results_until_edit() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "replay-policy-owner.opd");
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let alternate = match policy {
        oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size,
            minimum_size,
        } => oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size: reference_size * 2.,
            minimum_size,
        },
        _ => panic!("expected a model-relative desktop policy"),
    };
    let mut first = PartReplayService::with_policy(policy).unwrap();
    first
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let prefix = first
        .checkpoint_prefix(&owner.files, document)
        .unwrap()
        .unwrap();
    let mut same = PartReplayService::with_policy(policy).unwrap();
    same.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(same.replay_count(), 0);
    let mut other = PartReplayService::with_policy(alternate).unwrap();
    assert!(other.part_requires_update(&owner.files, document).unwrap());
    other.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(other.replay_count(), 1);
    assert!(!first.part_requires_update(&owner.files, document).unwrap());
    first.ensure_part(&mut owner.files, document).unwrap();
    other.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(first.replay_count(), 1);
    assert_eq!(other.replay_count(), 1);
    let session = owner.files.workspace().session(document).unwrap();
    assert!(
        prefix
            .prepare_checkpoints(
                session.history(),
                session.snapshot(),
                &mut oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
                    session.history(),
                    owner
                        .files
                        .workspace()
                        .identity(document)
                        .unwrap()
                        .internal_name,
                    policy
                )
            )
            .is_ok()
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    assert!(first.part_requires_update(&owner.files, document).unwrap());
    assert!(other.part_requires_update(&owner.files, document).unwrap());
    first.ensure_part(&mut owner.files, document).unwrap();
    assert!(!first.part_requires_update(&owner.files, document).unwrap());
    assert!(other.part_requires_update(&owner.files, document).unwrap());
}

#[test]
fn policy_cache_evicts_oldest_results_and_retires_their_prefixes() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "replay-policy-bound.opd");
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut first = PartReplayService::with_policy(policy).unwrap();
    first
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    let prefix = first
        .checkpoint_prefix(&owner.files, document)
        .unwrap()
        .unwrap();
    let oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size,
        minimum_size,
    } = policy
    else {
        panic!("expected a model-relative desktop policy");
    };
    for factor in 2..=5 {
        let alternate = oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
            reference,
            reference_size: reference_size * f64::from(factor),
            minimum_size,
        };
        PartReplayService::with_policy(alternate)
            .unwrap()
            .ensure_part(&mut owner.files, document)
            .unwrap();
    }
    assert!(first.slot(&owner.files, document).is_none());
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    assert!(
        prefix
            .prepare_checkpoints(session.history(), session.snapshot(), &mut evaluator)
            .is_err()
    );
}
