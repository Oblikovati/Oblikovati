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
