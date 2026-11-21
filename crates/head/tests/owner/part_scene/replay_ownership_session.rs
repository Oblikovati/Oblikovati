// SPDX-License-Identifier: GPL-2.0-only
//! Document sessions own ordered replay and its policy-scoped results.
use super::*;

#[test]
fn appended_part_capability_rejects_foreign_and_stale_owners() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "append-capability-first.opd");
    let second = open_named_prism(&mut owner, "append-capability-second.opd");
    let policy = Some(crate::part_scene::resolution_policy::desktop_policy());
    owner
        .files
        .recompute_owned_part(first, policy, &mut 0)
        .unwrap();
    let workspace = owner.files.workspace();
    let replay = workspace.owned_part_replay(first, policy).unwrap();
    assert!(replay.owns_identity(workspace.identity(first).unwrap().internal_name));
    assert!(!replay.owns_identity(workspace.identity(second).unwrap().internal_name));
    assert!(workspace.owned_part_replay(second, policy).is_err());
    assert!(workspace.owned_part_replay(first, None).is_err());
    let foreign = workspace.session(second).unwrap();
    assert!(
        replay
            .validate(foreign.history(), foreign.snapshot(), FeatureId(1))
            .is_err()
    );
    owner
        .files
        .commands(first)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    let changed = owner.files.workspace().session(first).unwrap();
    assert!(
        replay
            .validate(changed.history(), changed.snapshot(), FeatureId(1))
            .is_err()
    );
}

#[test]
fn private_preview_uses_document_identity_without_publishing() {
    let (mut owner, _) = owner();
    let document = open_named_prism(&mut owner, "preview-owner.opd");
    let policy = Some(crate::part_scene::resolution_policy::desktop_policy());
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let generation = session.generation();
    let had_published_part = session.owned_part_slot(policy).is_some();
    let candidate = session.snapshot().clone();
    let preview = workspace
        .preview_owned_part(document, policy, &candidate)
        .unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    assert!(!preview.used_checkpoint);
    assert!(
        !preview
            .evaluation
            .body(FeatureId(1))
            .unwrap()
            .face_reference_keys(session.history(), guid)
            .unwrap()
            .is_empty()
    );
    assert_eq!(session.generation(), generation);
    assert_eq!(
        session.owned_part_slot(policy).is_some(),
        had_published_part
    );
    let error = workspace
        .preview_owned_part(u64::MAX, policy, &candidate)
        .err()
        .unwrap();
    assert!(error.to_string().contains(&u64::MAX.to_string()));
}

#[test]
fn session_recomputes_each_document_under_its_own_identity() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "session-replay-first.opd");
    let second = open_named_prism(&mut owner, "session-replay-second.opd");
    let desktop_policy = crate::part_scene::resolution_policy::desktop_policy();
    let policy = Some(desktop_policy);
    let mut attempts = 0;
    assert!(
        !owner
            .files
            .workspace()
            .session(first)
            .unwrap()
            .snapshot()
            .requires_named_part_replay()
    );
    owner
        .files
        .recompute_owned_part(first, policy, &mut attempts)
        .unwrap();
    owner
        .files
        .recompute_owned_part(second, policy, &mut attempts)
        .unwrap();
    assert_eq!(attempts, 2);
    let workspace = owner.files.workspace();
    let first_session = workspace.session(first).unwrap();
    let second_session = workspace.session(second).unwrap();
    let first_guid = workspace.identity(first).unwrap().internal_name;
    let second_guid = workspace.identity(second).unwrap().internal_name;
    assert_ne!(first_guid, second_guid);
    for (session, guid) in [(first_session, first_guid), (second_session, second_guid)] {
        let part = session
            .owned_part_slot(policy)
            .unwrap()
            .current(session)
            .unwrap();
        let body = part.body(FeatureId(1)).unwrap();
        assert!(
            !body
                .face_reference_keys(session.history(), guid)
                .unwrap()
                .is_empty()
        );
    }
    owner
        .files
        .commands(first)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    let workspace = owner.files.workspace();
    let first_session = workspace.session(first).unwrap();
    let second_session = workspace.session(second).unwrap();
    assert!(
        first_session
            .owned_part_slot(policy)
            .unwrap()
            .current(first_session)
            .is_err()
    );
    assert!(
        second_session
            .owned_part_slot(policy)
            .unwrap()
            .current(second_session)
            .is_ok()
    );
    owner
        .files
        .recompute_owned_part(first, policy, &mut attempts)
        .unwrap();
    assert_eq!(attempts, 3);
    assert!(
        owner
            .files
            .recompute_owned_part(u64::MAX, policy, &mut attempts)
            .is_err()
    );
    assert_eq!(attempts, 3);
    let oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
        reference,
        minimum_size,
        ..
    } = desktop_policy
    else {
        panic!("expected model-relative policy");
    };
    let invalid = oblikovati_martensite::PrismResolutionPolicy::ModelRelative {
        reference,
        reference_size: 0.,
        minimum_size,
    };
    assert!(
        owner
            .files
            .recompute_owned_part(first, Some(invalid), &mut attempts)
            .is_err()
    );
    assert_eq!(attempts, 3);
}

#[test]
fn session_builds_and_reuses_checkpoints_per_document() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "session-checkpoints-first.opd");
    let second = open_named_prism(&mut owner, "session-checkpoints-second.opd");
    let policy = Some(crate::part_scene::resolution_policy::desktop_policy());
    let mut attempts = 0;
    owner
        .files
        .ensure_owned_checkpoints(first, policy, &mut attempts)
        .unwrap();
    owner
        .files
        .ensure_owned_checkpoints(first, policy, &mut attempts)
        .unwrap();
    assert_eq!(attempts, 1);
    owner
        .files
        .ensure_owned_checkpoints(second, policy, &mut attempts)
        .unwrap();
    assert_eq!(attempts, 2);
    owner
        .files
        .commands(first)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    let second_session = owner.files.workspace().session(second).unwrap();
    assert!(
        second_session
            .owned_checkpoint_slot(policy)
            .unwrap()
            .candidate_prefix(second_session)
            .is_ok()
    );
    owner
        .files
        .ensure_owned_checkpoints(first, policy, &mut attempts)
        .unwrap();
    assert_eq!(attempts, 3);
    assert!(
        owner
            .files
            .ensure_owned_checkpoints(u64::MAX, policy, &mut attempts)
            .is_err()
    );
    assert!(
        owner
            .files
            .ensure_owned_checkpoints(first, None, &mut attempts)
            .is_err()
    );
    assert_eq!(attempts, 3);
}

#[test]
fn candidate_capability_rejects_foreign_policy_and_stale_prefix() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "candidate-owner-first.opd");
    let second = open_named_prism(&mut owner, "candidate-owner-second.opd");
    let policy = Some(crate::part_scene::resolution_policy::desktop_policy());
    let mut attempts = 0;
    owner
        .files
        .ensure_owned_checkpoints(first, policy, &mut attempts)
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(first).unwrap();
    let slot = session.owned_checkpoint_slot(policy).unwrap();
    let prefix = slot.candidate_prefix(session).unwrap();
    assert!(
        workspace
            .owned_checkpoint_replay(second, policy, prefix)
            .is_err()
    );
    let prefix = slot.candidate_prefix(session).unwrap();
    assert!(
        workspace
            .owned_checkpoint_replay(first, None, prefix)
            .is_err()
    );
    let prefix = slot.candidate_prefix(session).unwrap();
    let replay = workspace
        .owned_checkpoint_replay(first, policy, slot.candidate_prefix(session).unwrap())
        .unwrap();
    let foreign = workspace.session(second).unwrap();
    assert!(
        replay
            .prepare(foreign.snapshot(), foreign.history())
            .is_err()
    );
    owner
        .files
        .commands(first)
        .unwrap()
        .run("Change width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "5 cm")
        })
        .unwrap();
    assert!(
        owner
            .files
            .workspace()
            .owned_checkpoint_replay(first, policy, prefix)
            .is_err()
    );
    let changed = owner.files.workspace().session(first).unwrap();
    assert!(
        replay
            .prepare(changed.snapshot(), changed.history())
            .is_err()
    );
}
