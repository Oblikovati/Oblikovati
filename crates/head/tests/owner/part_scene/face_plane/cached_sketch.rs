// SPDX-License-Identifier: GPL-2.0-only
//! Inspect face sketches against current, policy-matched part results.
use super::*;
use oblikovati_automation::PartSketchContinuity;
use oblikovati_model::identity::DocumentGuid;

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
