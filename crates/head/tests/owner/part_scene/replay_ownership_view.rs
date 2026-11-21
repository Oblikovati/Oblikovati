// SPDX-License-Identifier: GPL-2.0-only
//! Part-view edits preserve only validated named results.
use super::*;

#[test]
#[ignore = "manual latency and peak-memory baseline for view-only edits"]
fn measure_solid_view_metadata_replay() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_named_prism(&mut owner, "view-replay-baseline.opd");
    let mut replay =
        PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    let before = replay.replay_count();
    let started = std::time::Instant::now();
    for index in 0..20 {
        Client::new(owner.document_host())
            .part_design_views()
            .set_options(&SetPartDesignViewOptionsArgs {
                id: 1,
                published: Some(Box::new(index % 2 == 0)),
                ..Default::default()
            })
            .unwrap();
        replay.ensure_part(&mut owner.files, document).unwrap();
    }
    eprintln!(
        "view_metadata iterations=20 elapsed_us={} replays={}",
        started.elapsed().as_micros(),
        replay.replay_count() - before
    );
    assert!(
        replay
            .current_part(&owner.files, document)
            .unwrap()
            .body(FeatureId(1))
            .is_some()
    );
}

#[test]
fn view_option_reuses_named_solid_but_geometry_edit_invalidates_it() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_named_prism(&mut owner, "view-replay-ownership.opd");
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let session = owner.files.workspace().session(document).unwrap();
    let old_keys = replay
        .current_part(&owner.files, document)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .face_reference_keys(session.history(), guid)
        .unwrap();
    let old_prefix = owner
        .files
        .workspace()
        .owned_part_replay(document, Some(policy))
        .unwrap();
    let before = replay.replay_count();
    let start = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    Client::new(owner.document_host())
        .part_design_views()
        .copy(&CopyPartDesignViewArgs {
            source_id: 1,
            name: "Copy".into(),
        })
        .unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before);
    Client::new(owner.document_host())
        .part_design_views()
        .set_options(&SetPartDesignViewOptionsArgs {
            id: 1,
            published: Some(Box::new(false)),
            ..Default::default()
        })
        .unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before);
    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    client.transactions().redo().unwrap();
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: start as i64,
        })
        .unwrap();
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: start as i64 + 2,
        })
        .unwrap();
    drop(client);
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before);
    let session = owner.files.workspace().session(document).unwrap();
    let new_keys = replay
        .current_part(&owner.files, document)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .face_reference_keys(session.history(), guid)
        .unwrap();
    assert_eq!(new_keys, old_keys);
    assert!(
        old_prefix
            .validate(session.history(), session.snapshot(), FeatureId(1))
            .is_err()
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Edit width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "6 cm")
        })
        .unwrap();
    assert!(replay.current_part(&owner.files, document).is_err());
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before + 1);
}

#[test]
fn grouped_view_and_geometry_history_requires_fresh_solid() {
    let (owner, _) = owner();
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_named_prism(&mut owner, "view-replay-group.opd");
    let mut replay = PartReplayService::with_policy(policy).unwrap();
    replay.ensure_part(&mut owner.files, document).unwrap();
    let client = Client::new(owner.document_host());
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Grouped edits".into(),
        })
        .unwrap();
    client
        .part_design_views()
        .set_options(&SetPartDesignViewOptionsArgs {
            id: 1,
            published: Some(Box::new(false)),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
    client.transactions().end().unwrap();
    drop(client);
    replay.ensure_part(&mut owner.files, document).unwrap();
    let before = replay.replay_count();
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(replay.current_part(&owner.files, document).is_err());
    replay.ensure_part(&mut owner.files, document).unwrap();
    assert_eq!(replay.replay_count(), before + 1);
}
