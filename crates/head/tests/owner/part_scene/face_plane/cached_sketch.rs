// SPDX-License-Identifier: GPL-2.0-only
//! Inspect face sketches against current, policy-matched part results.
use super::*;
use oblikovati_api::wire::SketchArgs;
use oblikovati_automation::{FileHost, PartReplayService, PartSketchContinuity};
use oblikovati_model::identity::DocumentGuid;

#[test]
fn inspecting_one_independent_sketch_does_not_replay_an_unrelated_face_sketch() {
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    assert!(
        owner
            .start_sketch_on_face_line(document, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
    );
    owner.finish_sketch_edit().unwrap();
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Change depth", |snapshot, _| {
            let id = snapshot.parameters.find("depth").unwrap();
            snapshot.parameters.set_expression(id, "3 cm")
        })
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let foreign = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let foreign_resolver =
        PartSketchContinuity::new(foreign, Some(policy), session.sketch_solve_options());
    let sketch = SketchArgs { sketch_index: 0 };
    let readonly = Client::new(
        oblikovati_automation::FileQueryHost::new(&owner.files)
            .with_part_sketch_continuity(foreign_resolver),
    );
    assert!(readonly.sketch().get(&sketch).unwrap().healthy);
    assert!(
        readonly
            .sketch()
            .list()
            .unwrap_err()
            .to_string()
            .contains("expected sketch continuity owner")
    );

    let mut replay = PartReplayService::with_policy(policy).unwrap();
    let directed = SketchArgs { sketch_index: 1 };
    assert!(inspect_with_replay(&mut owner, &mut replay, &sketch).healthy);
    assert_eq!(replay.replay_count(), 0);
    assert!(inspect_with_replay(&mut owner, &mut replay, &directed).healthy);
    // Moving the face updates the sketch solve before publishing the new generation.
    assert_eq!(replay.replay_count(), 2);
    assert!(inspect_with_replay(&mut owner, &mut replay, &directed).healthy);
    assert_eq!(replay.replay_count(), 2);
}

fn inspect_with_replay(
    owner: &mut TestOwner,
    replay: &mut PartReplayService,
    sketch: &SketchArgs,
) -> oblikovati_api::wire::SketchInfo {
    Client::new(
        FileHost::new(&mut owner.files, &mut owner.guids, &owner.version).with_part_update(replay),
    )
    .sketch()
    .get(sketch)
    .unwrap()
}

#[test]
fn sketch_inspection_reuses_only_current_matching_datums() {
    let policy = crate::part_scene::resolution_policy::desktop_policy();
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_policy(policy).unwrap();
    let document = open_prism(&mut owner, false);
    assert!(
        owner
            .start_sketch_on_face_line(document, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
    );
    let sketch = owner.active_sketch_edit().unwrap().sketch;
    owner.finish_sketch_edit().unwrap();
    owner.take_scene().unwrap().unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let resolver = PartSketchContinuity::new(guid, Some(policy), session.sketch_solve_options());
    let datums = resolver
        .cached_datums(workspace, document)
        .unwrap()
        .expect("current part datums");
    let plane = session
        .snapshot()
        .resolve_sketch_plane_with_datums(sketch, &datums)
        .unwrap();
    assert!((plane.recipe().origin[2] - 2.).abs() < 1e-9);
    let different: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let foreign =
        PartSketchContinuity::new(different, Some(policy), session.sketch_solve_options());
    assert!(foreign.cached_datums(workspace, document).is_err());
    let other_policy = PartSketchContinuity::new(guid, None, session.sketch_solve_options());
    assert!(
        other_policy
            .cached_datums(workspace, document)
            .unwrap()
            .is_none()
    );
    let mut other_options = session.sketch_solve_options();
    other_options.max_iterations += 1;
    let other_solver = PartSketchContinuity::new(guid, Some(policy), other_options);
    assert!(
        other_solver
            .cached_datums(workspace, document)
            .unwrap()
            .is_none()
    );
    let solved = other_solver
        .datums_for_document(workspace, document)
        .unwrap();
    assert!(
        (session
            .snapshot()
            .resolve_sketch_plane_with_datums(sketch, &solved)
            .unwrap()
            .recipe()
            .origin[2]
            - 2.)
            .abs()
            < 1e-9
    );
    let sketch_index = session
        .snapshot()
        .sketches
        .sketches()
        .iter()
        .position(|item| item.id() == sketch)
        .unwrap() as i64;
    let inspected = Client::new(
        oblikovati_automation::FileQueryHost::new(&owner.files)
            .with_part_sketch_continuity(resolver),
    )
    .sketch()
    .list()
    .unwrap();
    assert!(
        inspected
            .sketches
            .unwrap()
            .iter()
            .any(|row| row.index == sketch_index && row.healthy)
    );
    let original_generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Draft depth")
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let resolver = PartSketchContinuity::new(guid, Some(policy), session.sketch_solve_options());
    assert!(
        resolver
            .cached_datums(workspace, document)
            .unwrap()
            .is_some()
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Draft depth edit", |snapshot, _| {
            let id = snapshot.parameters.find("depth").unwrap();
            snapshot.parameters.set_expression(id, "4 cm")
        })
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let draft_generation = session.generation();
    assert!(draft_generation > original_generation);
    assert!(
        resolver
            .cached_datums(workspace, document)
            .unwrap()
            .is_none()
    );
    let draft = resolver.datums_for_document(workspace, document).unwrap();
    let plane = session
        .snapshot()
        .resolve_sketch_plane_with_datums(sketch, &draft)
        .unwrap();
    assert!((plane.recipe().origin[2] - 4.).abs() < 1e-9);
    assert_eq!(session.generation(), draft_generation);
    assert!(session.in_transaction());
    assert!(resolver.cached_datums(workspace, document).unwrap().is_none());
    owner.files.commands(document).unwrap().abort().unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let datums = resolver.datums_for_document(workspace, document).unwrap();
    let restored = session
        .snapshot()
        .resolve_sketch_plane_with_datums(sketch, &datums)
        .unwrap();
    assert!((restored.recipe().origin[2] - 2.).abs() < 1e-9);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Change depth", |snapshot, _| {
            let id = snapshot.parameters.find("depth").unwrap();
            snapshot.parameters.set_expression(id, "3 cm")
        })
        .unwrap();
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let resolver = PartSketchContinuity::new(guid, Some(policy), session.sketch_solve_options());
    assert!(
        resolver
            .cached_datums(workspace, document)
            .unwrap()
            .is_none()
    );
    let updated = resolver.datums_for_document(workspace, document).unwrap();
    let plane = session
        .snapshot()
        .resolve_sketch_plane_with_datums(sketch, &updated)
        .unwrap();
    assert!((plane.recipe().origin[2] - 3.).abs() < 1e-9);
}
