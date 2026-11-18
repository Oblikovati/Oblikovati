// SPDX-License-Identifier: GPL-2.0-only
//! Document sessions own ordered replay and its policy-scoped results.
use super::*;

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
