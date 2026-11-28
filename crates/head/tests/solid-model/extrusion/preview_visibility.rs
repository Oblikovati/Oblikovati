// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::diagnostics;
use super::super::viewport_interaction::click_at;
use super::super::{
    acceptance::{feature_count, feature_distance},
    direction::preview_depth,
    edit_properties::start_published_edit,
    replace_extrusion_distance,
};
use super::ApplicationWorld;
use cucumber::{given, then, when};

fn preview_eye_point(world: &ApplicationWorld) -> [i32; 2] {
    let ([left, _], [width, _]) = diagnostics::window_geometry("Extrude").unwrap();
    let header_y = world
        .run_in_window("Extrude", "Extrusion1›Profile")
        .center()[1];
    diagnostics::pointer_pixels([left + width - 10. - 24. - 4. - 12., header_y])
}

#[given("a valid feature is being edited and its Properties breadcrumb offers preview visibility")]
fn edit_with_preview(world: &mut ApplicationWorld) {
    start_published_edit(world);
    replace_extrusion_distance(world, "8");
    let eye = preview_eye_point(world);
    world.hover(eye);
    world.expect_text("Hide Preview");
    assert_visible_draft(world);
}

#[when("the user hides the preview using that control")]
fn hide_preview(world: &mut ApplicationWorld) {
    let eye = preview_eye_point(world);
    click_at(world, eye);
    world.pump(Vec::new());
}

#[then("the feature preview disappears while the input selections and values remain displayed")]
fn assert_hidden(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert_eq!(preview_depth(world), [0., 2.]);
    assert_draft_and_document(world);
    world.hover(preview_eye_point(world));
    world.expect_text("Show Preview");
    world.save_capture("extrude-preview-hidden");
}

#[when("the user shows the preview again")]
fn show_preview(world: &mut ApplicationWorld) {
    let eye = preview_eye_point(world);
    click_at(world, eye);
    world.pump(Vec::new());
}

#[then("it returns using the same definition")]
fn assert_preview(world: &mut ApplicationWorld) {
    assert_visible_draft(world);
    world.save_capture("extrude-preview-shown");
}

fn assert_visible_draft(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(preview_depth(world), [0., 0.8]);
    assert_draft_and_document(world);
}

fn assert_draft_and_document(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "1Profile");
    world.run_in_window("Extrude", "1SketchPlane");
    world.run_in_window("Extrude", "8");
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 20.);
}
