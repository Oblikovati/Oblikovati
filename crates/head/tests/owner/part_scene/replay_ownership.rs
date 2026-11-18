// SPDX-License-Identifier: GPL-2.0-only
//! Part replay slots remain scoped to their own document generations.
use super::*;
use oblikovati_automation::{PartReplayService, PartUpdateAccess};

fn open_named_prism(owner: &mut TestOwner, name: &str) -> u64 {
    let path = std::env::temp_dir().join(name);
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), stored_bytes(false));
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

#[test]
fn cold_checkpoint_request_publishes_part_with_one_replay() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "cold-checkpoint-first.opd");
    let second = open_named_prism(&mut owner, "cold-checkpoint-second.opd");
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.ensure_checkpoints(&mut owner.files, first).unwrap();
    assert_eq!(replay.replay_count(), 1);
    assert!(replay.current_part(&owner.files, first).is_ok());
    assert!(
        replay
            .checkpoint_prefix(&owner.files, first)
            .unwrap()
            .is_some()
    );
    assert!(replay.slot(&owner.files, second).is_none());
    replay.ensure_checkpoints(&mut owner.files, first).unwrap();
    assert_eq!(replay.replay_count(), 1);
    replay.ensure_checkpoints(&mut owner.files, second).unwrap();
    assert_eq!(replay.replay_count(), 2);
    assert!(replay.current_part(&owner.files, second).is_ok());
}

#[test]
fn checkpoint_cache_is_lazy_and_scoped_to_each_document_generation() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "checkpoint-first.opd");
    let second = open_named_prism(&mut owner, "checkpoint-second.opd");
    let mut replay =
        PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
    replay.ensure_part(&mut owner.files, first).unwrap();
    replay.ensure_part(&mut owner.files, second).unwrap();
    assert!(
        replay
            .checkpoint_prefix(&owner.files, first)
            .unwrap()
            .is_none()
    );
    let before = replay.replay_count();
    replay.ensure_checkpoints(&mut owner.files, first).unwrap();
    assert_eq!(replay.replay_count(), before + 1);
    replay.ensure_checkpoints(&mut owner.files, first).unwrap();
    assert_eq!(replay.replay_count(), before + 1);
    replay.ensure_checkpoints(&mut owner.files, second).unwrap();
    assert_eq!(replay.replay_count(), before + 2);
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    set_parameter(&mut owner, "width", "6 cm");
    assert!(
        replay
            .checkpoint_prefix(&owner.files, first)
            .unwrap()
            .is_none()
    );
    assert!(
        replay
            .checkpoint_prefix(&owner.files, second)
            .unwrap()
            .is_some()
    );
    replay.ensure_checkpoints(&mut owner.files, first).unwrap();
    assert!(
        replay
            .checkpoint_prefix(&owner.files, first)
            .unwrap()
            .is_some()
    );
    let replay = {
        let _previous_service = replay;
        PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap()
    };
    assert!(
        replay
            .checkpoint_prefix(&owner.files, first)
            .unwrap()
            .is_some()
    );
    assert!(
        replay
            .checkpoint_prefix(&owner.files, second)
            .unwrap()
            .is_some()
    );
}

#[test]
fn closing_a_document_retires_its_part_and_checkpoints_together() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "checkpoint-close.opd");
    let mut replay =
        PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
    replay
        .ensure_checkpoints(&mut owner.files, document)
        .unwrap();
    assert!(replay.slot(&owner.files, document).is_some());
    assert!(
        replay
            .checkpoint_prefix(&owner.files, document)
            .unwrap()
            .is_some()
    );
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    assert!(replay.slot(&owner.files, document).is_none());
    assert!(replay.checkpoint_prefix(&owner.files, document).is_err());
    let reopened = open_named_prism(&mut owner, "checkpoint-close.opd");
    assert_ne!(reopened, document);
    assert!(replay.slot(&owner.files, reopened).is_none());
    assert!(
        replay
            .checkpoint_prefix(&owner.files, reopened)
            .unwrap()
            .is_none()
    );
}

#[test]
fn headless_replay_tracks_each_document_and_its_own_generation() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "replay-owner-first.opd");
    let second = open_named_prism(&mut owner, "replay-owner-second.opd");
    let mut replay = PartReplayService::default();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    assert!(
        replay
            .update_part(&mut owner.files, second)
            .unwrap()
            .is_empty()
    );
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    set_parameter(&mut owner, "width", "6 cm");
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(replay.current_part(&owner.files, first).is_err());
    assert!(replay.current_part(&owner.files, second).is_ok());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    owner.files.commands(first).unwrap().undo().unwrap();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    owner.files.commands(first).unwrap().redo().unwrap();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: second,
            force: false,
        })
        .unwrap();
    assert!(replay.slot(&owner.files, second).is_none());
    assert!(replay.slot(&owner.files, first).is_some());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    let reopened = open_named_prism(&mut owner, "replay-owner-second.opd");
    assert_ne!(reopened, second);
    assert!(replay.part_requires_update(&owner.files, reopened).unwrap());
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, reopened)
            .unwrap()
            .is_empty()
    );
}

#[test]
fn desktop_replay_keeps_current_results_for_both_documents() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "desktop-replay-first.opd");
    let second = open_named_prism(&mut owner, "desktop-replay-second.opd");
    assert!(
        owner
            .part_scenes
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    assert!(
        owner
            .part_scenes
            .update_part(&mut owner.files, second)
            .unwrap()
            .is_empty()
    );
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, first)
            .unwrap()
    );
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, second)
            .unwrap()
    );
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    owner.part_scenes.refresh(&mut owner.files).unwrap();
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: second })
        .unwrap();
    owner.part_scenes.refresh(&mut owner.files).unwrap();
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, first)
            .unwrap()
    );
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    set_parameter(&mut owner, "width", "6 cm");
    assert!(
        owner
            .part_scenes
            .part_requires_update(&owner.files, first)
            .unwrap()
    );
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, second)
            .unwrap()
    );
    owner
        .part_scenes
        .update_part(&mut owner.files, first)
        .unwrap();
    owner.files.commands(first).unwrap().undo().unwrap();
    assert!(
        owner
            .part_scenes
            .part_requires_update(&owner.files, first)
            .unwrap()
    );
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, second)
            .unwrap()
    );
    owner
        .part_scenes
        .update_part(&mut owner.files, first)
        .unwrap();
    owner.files.commands(first).unwrap().redo().unwrap();
    assert!(
        owner
            .part_scenes
            .part_requires_update(&owner.files, first)
            .unwrap()
    );
    assert!(
        !owner
            .part_scenes
            .part_requires_update(&owner.files, second)
            .unwrap()
    );
}

#[test]
fn document_replay_is_shared_by_policy_and_replaced_on_policy_change() {
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
    assert!(first.part_requires_update(&owner.files, document).unwrap());
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
            .is_err()
    );
}

#[test]
fn rejected_foreign_replay_does_not_replace_document_policy_or_result() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "owned-replay-first.opd");
    let second = open_named_prism(&mut owner, "owned-replay-second.opd");
    let mut replay = PartReplayService::default();
    replay.ensure_part(&mut owner.files, first).unwrap();
    let session = owner.files.workspace().session(second).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(second)
        .unwrap()
        .internal_name;
    let mut backend = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_optional_policy(
        session.history(),
        guid,
        None,
    );
    let foreign = session
        .prepare_part_evaluation()
        .evaluate(&mut backend)
        .unwrap();
    let alternate = Some(crate::part_scene::resolution_policy::desktop_policy());
    assert!(
        owner
            .files
            .commands(first)
            .unwrap()
            .adopt_owned_part(alternate, foreign)
            .is_err()
    );
    assert!(replay.current_part(&owner.files, first).is_ok());
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
}

#[test]
fn checkpoints_require_a_current_document_owned_result() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "checkpoint-without-result.opd");
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let session = owner.files.workspace().session(document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let mut backend = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let checkpoints = session
        .prepare_part_evaluation()
        .evaluate_checkpoints(&mut backend)
        .unwrap();
    assert!(
        owner
            .files
            .commands(document)
            .unwrap()
            .adopt_owned_checkpoints(Some(policy), checkpoints)
            .is_err()
    );
    assert!(
        PartReplayService::with_policy(policy)
            .unwrap()
            .slot(&owner.files, document)
            .is_none()
    );
}

#[path = "replay_ownership_parameter.rs"]
mod parameter;
