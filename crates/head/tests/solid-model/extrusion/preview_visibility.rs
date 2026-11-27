// SPDX-License-Identifier: GPL-2.0-only
use super::super::{
    acceptance::{feature_count, feature_distance},
    direction::preview_depth,
    edit_properties::start_published_edit,
    replace_extrusion_distance,
};
use super::{ApplicationWorld, click_in_extrude};
use cucumber::{given, then, when};

#[given("a valid feature is being edited and its Properties breadcrumb offers preview visibility")]
fn edit_with_preview(world: &mut ApplicationWorld) {
    start_published_edit(world);
    replace_extrusion_distance(world, "8");
    let breadcrumb = world.run_in_window("Extrude", "Extrusion1›Profile");
    let toggle = world.run_in_window("Extrude", "HidePreview");
    assert!((breadcrumb.center()[1] - toggle.center()[1]).abs() < 3.);
    assert_visible_draft(world);
}

#[when("the user hides the preview using that control")]
fn hide_preview(world: &mut ApplicationWorld) {
    click_in_extrude(world, "HidePreview");
    world.pump(Vec::new());
}

#[then("the feature preview disappears while the input selections and values remain displayed")]
fn assert_hidden(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert_eq!(preview_depth(world), [0., 2.]);
    assert_draft_and_document(world);
    world.run_in_window("Extrude", "ShowPreview");
    world.save_capture("extrude-preview-hidden");
}

#[when("the user shows the preview again")]
fn show_preview(world: &mut ApplicationWorld) {
    click_in_extrude(world, "ShowPreview");
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
