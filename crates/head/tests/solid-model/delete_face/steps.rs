// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Delete Face scenarios (delete-face-01..05, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use setup::{
    DELETE_FACE_WINDOW, FAILURE_MODAL_WINDOW, click_curved_side_face_of_cylinder,
    click_delete_face_ribbon, click_exposed_lower_circular_face, click_modal_control,
    click_panel_control, click_top_face_of_single_solid, has_highlighted_face_triangles,
    selected_face_count, set_heal_remaining_faces,
};

#[given("Delete Face is open with no face selected")]
#[given("Delete Face is open on the cylindrical solid")]
fn delete_face_open_empty(world: &mut ApplicationWorld) {
    click_delete_face_ribbon(world);
    assert_eq!(selected_face_count(world), 0);
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK disabled initially"
    );
}

#[when("the user clicks a face on the part")]
fn user_clicks_face_on_part(world: &mut ApplicationWorld) {
    click_top_face_of_single_solid(world);
}

#[then("the face is highlighted, the selector shows 1 Face, and OK becomes enabled.")]
fn face_highlighted_one_face_ok_enabled(world: &mut ApplicationWorld) {
    assert_eq!(selected_face_count(world), 1);
    assert!(
        has_highlighted_face_triangles(world),
        "expected clicked face to be highlighted in viewport scene"
    );
    let text = super::super::world::squeezed(&world.window_text(DELETE_FACE_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected selector to display 1 Face, saw {text}"
    );
    assert!(
        world.run_is_enabled("OK"),
        "expected OK to become enabled after selecting a face"
    );
}

#[given("a separate reconstructed cylindrical surface is centered at X = -10 mm")]
fn verify_reconstructed_cylindrical_surface(world: &mut ApplicationWorld) {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let has_surface = eval.ordered_bodies().filter(|b| b.is_surface()).any(|b| {
        b.body_reference_key(session.history(), guid)
            .ok()
            .and_then(|key| b.body_bounds_center(session.history(), guid, key).ok())
            .is_some_and(|center| (center[0] + 1.0).abs() < 0.05)
    });
    assert!(
        has_surface,
        "expected reconstructed cylindrical surface centered at X = -10 mm (-1.0 cm)"
    );
}

#[given("the part has three solid bodies and one surface body")]
fn verify_three_solids_and_one_surface(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        3,
        "expected 3 solid bodies, saw {:?}",
        browser.bodies
    );
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected 1 surface body, saw {:?}",
        browser.surface_bodies
    );
}

#[given("its cylindrical body has an exposed lower circular end face centered at (10, 15, -10) mm")]
fn verify_exposed_lower_circular_end_face(world: &mut ApplicationWorld) {
    let doc = world.active_id();
    let app = world.app();
    let ws = app.owner.documents().workspace();
    let session = ws.session(doc).unwrap();
    let guid = ws.identity(doc).unwrap().internal_name;
    let eval = app.owner.part_scenes_current(doc).unwrap();
    let found = eval.ordered_bodies().any(|body| {
        body.face_reference_keys(session.history(), guid)
            .unwrap_or_default()
            .iter()
            .any(|face| {
                body.face_bounds_center(session.history(), guid, face)
                    .ok()
                    .is_some_and(|c| {
                        (c[0] - 1.0).abs() < 0.05
                            && (c[1] - 1.5).abs() < 0.05
                            && (c[2] + 1.0).abs() < 0.05
                    })
            })
    });
    assert!(
        found,
        "expected exposed lower circular end face centered at (10, 15, -10) mm"
    );
}

#[given(
    "Delete Face is open with the lower circular face selected and Heal Remaining Faces enabled"
)]
fn open_delete_face_with_lower_face_and_heal(world: &mut ApplicationWorld) {
    click_delete_face_ribbon(world);
    click_exposed_lower_circular_face(world);
    set_heal_remaining_faces(world, true);
}

#[then("the MCAD Software reports “Create DelFace feature failed” and offers Edit and Cancel.")]
fn reports_create_delface_feature_failed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(FAILURE_MODAL_WINDOW).is_some(),
        "expected Create DelFace feature failed modal, saw {}",
        world.text()
    );
    let text = super::super::world::squeezed(&world.window_text(FAILURE_MODAL_WINDOW));
    assert!(
        text.contains("CreateDelFacefeaturefailed")
            && text.contains("Edit")
            && text.contains("Cancel"),
        "expected failure message with Edit and Cancel, saw {text}"
    );
}

#[when("the user clicks Edit")]
fn user_clicks_edit(world: &mut ApplicationWorld) {
    click_modal_control(world, "Edit");
}

#[then("the properties return with the face selected and healing still enabled.")]
fn properties_return_with_face_and_healing(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(FAILURE_MODAL_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(DELETE_FACE_WINDOW).is_some());
    let preview = world
        .state()
        .delete_face_preview
        .expect("expected delete face preview");
    assert_eq!(preview.request.faces.len(), 1);
    assert!(preview.request.heal_remaining_faces);
    let text = super::super::world::squeezed(&world.window_text(DELETE_FACE_WINDOW));
    assert!(
        text.contains("1Face"),
        "expected 1 Face retained after Edit, saw {text}"
    );
}

#[given("the user attempts the deletion and chooses Edit after the failure")]
fn attempt_deletion_and_choose_edit(world: &mut ApplicationWorld) {
    click_panel_control(world, "OK");
    reports_create_delface_feature_failed(world);
    click_modal_control(world, "Edit");
}

#[given("Delete Face has returned with that face selected and healing enabled")]
fn verify_delete_face_returned_with_face_and_heal(world: &mut ApplicationWorld) {
    properties_return_with_face_and_healing(world);
}

#[when("the user clears Heal Remaining Faces and clicks OK")]
fn clear_heal_and_click_ok(world: &mut ApplicationWorld) {
    set_heal_remaining_faces(world, false);
    click_panel_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the properties close and the selected face is removed from the displayed model.")]
fn properties_close_and_face_removed(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(DELETE_FACE_WINDOW).is_none(),
        "expected Delete Face panel to close after OK"
    );
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Delete Face1" && feature.kind == "Delete Face"),
        "expected Delete Face1 in browser features: {:?}",
        browser.features
    );
    assert!(
        browser.surface_bodies.iter().any(|body| body.faces == 2),
        "expected converted cylindrical surface body with 2 remaining faces, saw {:?}",
        browser.surface_bodies
    );
}

#[then("the browser shows two solid bodies and two surface bodies in this part.")]
fn browser_shows_two_solids_and_two_surfaces(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        2,
        "expected 2 solid bodies after unhealed Delete Face, saw {:?}",
        browser.bodies
    );
    assert_eq!(
        browser.surface_bodies.len(),
        2,
        "expected 2 surface bodies after unhealed Delete Face, saw {:?}",
        browser.surface_bodies
    );
}

#[given("Delete Face contains one selected face and OK is enabled")]
fn delete_face_contains_one_face_ok_enabled(world: &mut ApplicationWorld) {
    click_delete_face_ribbon(world);
    click_top_face_of_single_solid(world);
    assert_eq!(selected_face_count(world), 1);
    assert!(world.run_is_enabled("OK"));
}

#[when("I click the clear control in the Faces selector")]
fn click_clear_in_faces_selector(world: &mut ApplicationWorld) {
    click_panel_control(world, "x");
}

#[then("the selector returns to Select Faces and OK is disabled")]
fn selector_returns_to_select_faces_ok_disabled(world: &mut ApplicationWorld) {
    assert_eq!(selected_face_count(world), 0);
    let text = super::super::world::squeezed(&world.window_text(DELETE_FACE_WINDOW));
    assert!(
        text.contains("SelectFaces"),
        "expected Select Faces after clear, saw {text}"
    );
    assert!(
        !world.run_is_enabled("OK"),
        "expected OK disabled after clearing face selection"
    );
}

#[when("I select its curved side face and then its top face")]
fn select_curved_side_then_top_face(world: &mut ApplicationWorld) {
    click_curved_side_face_of_cylinder(world);
    assert_eq!(selected_face_count(world), 1);
    click_top_face_of_single_solid(world);
    assert_eq!(selected_face_count(world), 2);
}

#[then("the Faces selector displays 2 Faces and OK is enabled")]
fn faces_selector_displays_two_faces_ok_enabled(world: &mut ApplicationWorld) {
    assert_eq!(selected_face_count(world), 2);
    let text = super::super::world::squeezed(&world.window_text(DELETE_FACE_WINDOW));
    assert!(
        text.contains("2Faces"),
        "expected 2 Faces in Faces selector, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("I enable Heal Remaining Faces and clear the face selector")]
fn enable_heal_and_clear_face_selector(world: &mut ApplicationWorld) {
    set_heal_remaining_faces(world, true);
    click_panel_control(world, "x");
}

#[then("Heal Remaining Faces remains checked")]
fn heal_remaining_faces_remains_checked(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .delete_face_preview
        .expect("expected delete face preview state");
    assert!(
        preview.request.heal_remaining_faces,
        "expected Heal Remaining Faces to remain checked after clearing faces"
    );
}
