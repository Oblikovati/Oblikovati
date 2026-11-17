// SPDX-License-Identifier: GPL-2.0-only
//! Native Chamfer viewport selection and creation (#4046).
use super::acceptance::feature_count;
use super::viewport_interaction::{click_text_native, committed_extrusion};
use super::*;

#[given("Chamfer is open in equal-distance mode with a valid distance")]
fn chamfer_is_open(world: &mut ApplicationWorld) {
    committed_extrusion(world);
    click_text_native(world, "Chamfer");
    world.expect_text("Select Edges");
    world.expect_text("1 mm");
    assert!(!world.state().extrude_preview_active);
}

#[when("the user clicks an eligible block edge")]
fn click_block_edge(world: &mut ApplicationWorld) {
    super::viewport_fillet::pick_fillet_edge(world);
}

#[then("the edge remains selected in the tool")]
fn chamfer_edge_is_selected(world: &mut ApplicationWorld) {
    world.expect_text("1 Edge");
    assert_eq!(world.state().selected_edge_count, 1);
    super::viewport_fillet::selected_edge_is_visible(world);
}

#[then("a bevel preview appears")]
fn bevel_preview_is_visible(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 1);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    change_chamfer_distance(world, "3");
    let updated = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        super::viewport_interaction::preview_changed_outside_windows(&capture, &updated, &windows)
    );
    assert!(world.state().extrude_preview_active);
    world.save_capture("chamfer-preview-4046");
}

fn change_chamfer_distance(world: &mut ApplicationWorld, distance: &str) {
    let label = world.run_in_window("Chamfer", "Distance").clone();
    let position = [label.minimum[0] as i32 - 30, label.center()[1] as i32];
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    super::super::x11_input::type_digits(distance);
    world.pump(Vec::new());
    world.expect_text(distance);
}

#[then("the model can be updated by accepting the preview")]
fn commit_chamfer_preview(world: &mut ApplicationWorld) {
    click_text_native(world, "OK");
    assert!(!world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 2);
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Chamfer1" && feature.kind == "Chamfer")
    );
    world.expect_text("Chamfer1");
    world.save_capture("chamfer-committed-4046");
}

#[when("the user cancels the chamfer preview")]
fn cancel_chamfer_preview(world: &mut ApplicationWorld) {
    click_text_native(world, "Cancel");
}

#[then("the chamfer preview is discarded without changing the solid")]
fn chamfer_cancel_restores_solid(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 1);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
    world.save_capture("chamfer-cancelled-4046");
}

#[given("a committed chamfer is open for editing")]
fn committed_chamfer_is_open(world: &mut ApplicationWorld) {
    chamfer_is_open(world);
    click_block_edge(world);
    bevel_preview_is_visible(world);
    commit_chamfer_preview(world);
    open_chamfer_editor(world);
}

fn open_chamfer_editor(world: &mut ApplicationWorld) {
    let position = world.run("Chamfer1").center().map(|value| value as i32);
    super::super::x11_input::right_click(position);
    world.pump(Vec::new());
    world.expect_text("Show Dimensions");
    let position = super::acceptance::edit_menu_position(world);
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    world.expect_text("Distance");
    world.expect_text("1 Edge");
}

#[when("the user changes the chamfer distance to 2 mm")]
fn change_chamfer_to_two(world: &mut ApplicationWorld) {
    change_chamfer_distance(world, "2");
}

#[then("the chamfer edit preview does not change the committed distance")]
fn chamfer_edit_preview_is_private(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert!((committed_distance(world) - 3.).abs() < 1e-9);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    world.save_capture("chamfer-edit-preview-4046");
}

fn committed_distance(world: &mut ApplicationWorld) -> f64 {
    use oblikovati_api::{client::Client, wire::FeatureRefArgs};
    Client::new(world.app().owner.document_host())
        .features()
        .get(&FeatureRefArgs { id: 2 })
        .unwrap()
        .feature
        .scalars
        .unwrap()[0]
        .value
}

#[then("the committed chamfer distance is still 3 mm")]
fn cancelled_chamfer_edit_is_unchanged(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert!((committed_distance(world) - 3.).abs() < 1e-9);
}

#[when("the user reopens the chamfer editor")]
fn reopen_chamfer_editor(world: &mut ApplicationWorld) {
    open_chamfer_editor(world);
}

#[when("the user accepts the chamfer edit")]
fn accept_chamfer_edit(world: &mut ApplicationWorld) {
    click_text_native(world, "OK");
}

#[then("the committed chamfer distance is 2 mm")]
fn committed_chamfer_is_updated(world: &mut ApplicationWorld) {
    assert!(!world.state().extrude_preview_active);
    assert!((committed_distance(world) - 2.).abs() < 1e-9);
    world.save_capture("chamfer-edit-committed-4046");
}
