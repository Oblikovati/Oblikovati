// SPDX-License-Identifier: GPL-2.0-only
//! Exercise a named face clear through the native frame loop.
use super::*;
use crate::HeadWindow;
use oblikovati_api::{client::Client, wire::AssignAppearanceArgs};
use oblikovati_model::assets::DocumentAssets;

fn face_overrides(world: &mut ApplicationWorld) -> DocumentAssets {
    let document = world.active_id();
    world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .assets
        .clone()
}
fn assign_faces(world: &mut ApplicationWorld) {
    let body = Client::new(world.app().owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let faces = body.bodies.unwrap().remove(0).faces.unwrap();
    assert_eq!(faces.len(), 6);
    for face in faces {
        Client::new(world.app().owner.document_host())
            .appearances()
            .assign(&AssignAppearanceArgs {
                scope: "face".into(),
                key: face.key,
                appearance_id: "abs-black".into(),
            })
            .unwrap();
    }
    world.pump(Vec::new());
}
fn prepare_face_tool(world: &mut ApplicationWorld) {
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(world);
    assign_faces(world);
    world
        .app()
        .window
        .navigate_view(oblikovati_viewport::ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    world.pump(Vec::new());
    world.click_text("Select");
    world.click_text("Faces");
    clear_appearance::enable_clear(world);
    world.click(row_center(world, "Clear"));
}
#[test]
fn clear_selected_named_face_from_viewport_and_undo() {
    let mut world = ApplicationWorld::new();
    prepare_face_tool(&mut world);
    verify_face_clear_input(&mut world);
    world.close();
}

fn verify_face_clear_input(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"));
    let before = face_overrides(world);
    let point = world.app().window.model_pixel([2., 1.5, 2.]).unwrap();
    world.click(point);
    let (key, has_override) = world.state().selected_face_appearances.pop().unwrap();
    assert!(has_override);
    assert!(world.run_is_enabled("OK"));
    world.click_text("Cancel");
    assert_eq!(face_overrides(world), before);
    apply_face_clear(world, &key);
    world.click(row_center(world, "Undo"));
    assert_eq!(face_overrides(world), before);
}

fn apply_face_clear(world: &mut ApplicationWorld, key: &str) {
    world.click(row_center(world, "Clear"));
    world.click_text("OK");
    let cleared = face_overrides(world);
    let keys: Vec<_> = cleared
        .assignments
        .face_appearances()
        .map(|(reference, _)| reference.reference_key())
        .collect();
    assert_eq!(keys.len(), 5);
    assert!(!keys.contains(&key));
}

#[test]
fn clear_two_named_faces_through_viewport_and_undo() {
    let mut world = ApplicationWorld::new();
    prepare_face_tool(&mut world);
    let before = face_overrides(&mut world);
    let first = world.app().window.model_pixel([2., 1.5, 2.]).unwrap();
    world.click(first);
    world
        .app()
        .window
        .navigate_view(oblikovati_viewport::ViewportNavigation::Orbit {
            yaw: std::f64::consts::PI,
            pitch: 0.,
        })
        .unwrap();
    world.pump(Vec::new());
    let second = world.app().window.model_pixel([2., 1.5, 0.]).unwrap();
    clear_appearance::modified_viewport_click(&mut world, second, true, false);
    let selected = world.state().selected_face_appearances;
    assert_eq!(
        selected.len(),
        2,
        "expected two selected faces: {selected:?}"
    );
    assert!(selected.iter().all(|(_, assigned)| *assigned));
    assert!(world.run_is_enabled("OK"));
    world.click_text("Cancel");
    assert_eq!(face_overrides(&mut world), before);
    world.click(row_center(&world, "Clear"));
    world.click_text("OK");
    assert_eq!(
        face_overrides(&mut world)
            .assignments
            .face_appearances()
            .count(),
        4
    );
    world.click(row_center(&world, "Undo"));
    assert_eq!(face_overrides(&mut world), before);
    world.close();
}

#[test]
fn clear_mixed_body_and_face_through_native_window() {
    let mut world = ApplicationWorld::new();
    prepare_face_tool(&mut world);
    let body = world.state().browser.unwrap().bodies[0].key.clone();
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "body".into(),
            key: body.clone(),
            appearance_id: "abs-black".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    let before = face_overrides(&mut world);
    let point = world.app().window.model_pixel([2., 1.5, 2.]).unwrap();
    world.click(point);
    let face = world.state().selected_face_appearances[0].0.clone();
    super::super::workspace_layout::browser_feature::click_branch_arrow(&mut world, "Solid Bodies");
    clear_appearance::ctrl_click(&mut world, "Solid1");
    let state = world.state();
    assert_eq!(state.selected_body_keys, [body.clone()]);
    assert_eq!(state.selected_face_appearances[0].0, face);
    assert!(world.run_is_enabled("OK"));
    world.click_text("Cancel");
    assert_eq!(face_overrides(&mut world), before);
    world.click(row_center(&world, "Clear"));
    world.click_text("OK");
    let cleared = face_overrides(&mut world);
    let mut expected = before.clone();
    let body_ref = oblikovati_model::assets::AssignmentReference::new(&body, vec![]).unwrap();
    let face_ref = oblikovati_model::assets::AssignmentReference::new(&face, vec![]).unwrap();
    expected
        .assignments
        .set_body_appearance(&body_ref, "")
        .unwrap();
    expected
        .assignments
        .set_face_appearance(&face_ref, "")
        .unwrap();
    assert_eq!(cleared, expected);
    world.click(row_center(&world, "Undo"));
    assert_eq!(face_overrides(&mut world), before);
    world.close();
}
