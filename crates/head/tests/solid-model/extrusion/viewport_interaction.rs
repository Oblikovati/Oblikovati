// SPDX-License-Identifier: GPL-2.0-only
//! Solid feature editing through the viewport scenario (#4038).
use super::acceptance::{edit_menu_position, feature_count, feature_distance, named_solid_edges};
use super::*;

#[given("the MCAD Software is displaying a part in the solid modeling workspace")]
fn open_part(world: &mut ApplicationWorld) {
    new_metric_part(world);
}

#[given("the part contains an editable solid feature")]
fn committed_extrusion(world: &mut ApplicationWorld) {
    rectangle_in_waiting_extrusion(world);
    finish_requested_profile(world);
    world.pump(Vec::new());
    waiting_extrusion_shows_profile(world);
    waiting_extrusion_shows_default_preview(world);
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 10.);
    world.expect_text("Extrusion1");
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 },
    )
    .unwrap();
    world.pump(Vec::new());
}

#[when("the user opens the feature for modification")]
fn open_editor(world: &mut ApplicationWorld) {
    let position = world.run("Extrusion1").center().map(|value| value as i32);
    super::super::x11_input::right_click(position);
    world.pump(Vec::new());
    world.expect_text("Show Dimensions");
    let position = edit_menu_position(world);
    super::super::x11_input::click(position);
    world.pump(Vec::new());
}

#[then("its current selection and parameters are displayed")]
fn restored_parameters(world: &mut ApplicationWorld) {
    world.expect_text("Distance A");
    world.expect_text("1 Profile");
    world.expect_text("1 cm");
    assert_eq!(feature_distance(world), 10.);
}

#[when("the user changes a parameter or requested viewport selection")]
fn change_distance(world: &mut ApplicationWorld) {
    replace_distance_native(world, "12");
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
    replace_distance_native(world, "20");
}

#[then("the preview reflects the pending modification")]
fn changed_preview(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_distance(world), 10.);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    assert!(preview_changed_outside_windows(
        world.last_capture.as_ref().expect("12 mm preview capture"),
        &capture,
        &windows
    ));
    if std::env::var_os("OBK_BDD_SCENARIO").is_some() {
        world.save_capture("solid-extrude-edit-preview-4038");
    }
}

#[when("the user confirms the modification")]
fn accept_edit(world: &mut ApplicationWorld) {
    click_text_native(world, "OK");
}

#[then("the existing feature is updated in the model browser")]
fn updated_feature(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 20.);
    world.expect_text("Extrusion1");
}

#[then("the updated result appears in the viewport")]
fn updated_result(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
    assert_extrusion_edges(world, 20.);
    world.save_capture("solid-extrude-edit-accepted-4038");
}

#[given("an existing solid feature is open for modification with a changed preview")]
fn open_changed_preview(world: &mut ApplicationWorld) {
    committed_extrusion(world);
    open_editor(world);
    restored_parameters(world);
    change_distance(world);
    changed_preview(world);
}

#[when("the user cancels the modification")]
fn cancel_edit(world: &mut ApplicationWorld) {
    click_text_native(world, "Cancel");
}

#[then("the edit panel closes")]
fn edit_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Distance A");
    assert!(!world.state().extrude_preview_active);
}

#[then("the previously committed model remains visible")]
fn original_result(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 10.);
    assert_extrusion_edges(world, 10.);
    updated_solid_is_visible(world);
    world.save_capture("solid-extrude-edit-cancelled-4038");
}

fn updated_solid_is_visible(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
}

fn click_text_native(world: &mut ApplicationWorld, text: &str) {
    let position = world.run(text).center().map(|value| value as i32);
    super::super::x11_input::click(position);
    world.pump(Vec::new());
}

fn replace_distance_native(world: &mut ApplicationWorld, distance: &str) {
    let label = world.run_in_window("Extrude", "DistanceA").clone();
    let position = [label.minimum[0] as i32 - 30, label.center()[1] as i32];
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    super::super::x11_input::type_digits(distance);
    world.pump(Vec::new());
    world.expect_text(distance);
}

// A small changed patch avoids counting isolated raster noise as a geometry update.
const MIN_CHANGED_PREVIEW_PIXELS: usize = 20;

fn preview_changed_outside_windows(
    before: &oblikovati_viewport::ViewportCapture,
    after: &oblikovati_viewport::ViewportCapture,
    windows: &[[f32; 4]],
) -> bool {
    if before.extent != after.extent || before.rgba.len() != after.rgba.len() {
        return false;
    }
    let width = after.extent[0] as usize;
    before
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .zip(after.rgba.as_chunks::<4>().0)
        .enumerate()
        .filter(|(index, (a, b))| changed_visible_preview(*index, width, a, b, windows))
        .take(MIN_CHANGED_PREVIEW_PIXELS)
        .count()
        == MIN_CHANGED_PREVIEW_PIXELS
}

fn changed_visible_preview(
    index: usize,
    width: usize,
    before: &[u8; 4],
    after: &[u8; 4],
    windows: &[[f32; 4]],
) -> bool {
    let point = [(index % width) as f32, (index / width) as f32];
    let covered = windows
        .iter()
        .any(|w| point[0] >= w[0] && point[0] < w[2] && point[1] >= w[1] && point[1] < w[3]);
    !covered && before != after && (preview_pixel(before) || preview_pixel(after))
}

fn preview_pixel(pixel: &[u8; 4]) -> bool {
    pixel[0] > pixel[1].saturating_add(20) && pixel[1] > pixel[2].saturating_add(20)
}

fn assert_extrusion_edges(world: &mut ApplicationWorld, expected: f64) {
    let edges = named_solid_edges(world);
    let matches = edges
        .iter()
        .filter(|key| edge_length_matches(world, key, expected))
        .count();
    assert!(
        matches >= 4,
        "extrusion edges of length {expected} mm: expected at least four, got {matches}"
    );
}

fn edge_length_matches(world: &mut ApplicationWorld, key: &str, expected: f64) -> bool {
    let client = oblikovati_api::client::Client::new(world.app().owner.document_host());
    let result = client
        .analysis()
        .measure(&oblikovati_api::wire::MeasureArgs {
            body_index: 0,
            type_: "length".into(),
            key_a: key.into(),
            ..Default::default()
        })
        .unwrap();
    result.unit == "mm" && (result.value - expected).abs() < 1e-6
}

#[test]
fn preview_difference_excludes_unchanged_and_covered_pixels() {
    let before = sample_capture([0, 0, 0, 255]);
    let after = sample_capture([200, 100, 10, 255]);
    assert!(!preview_changed_outside_windows(&before, &before, &[]));
    assert!(preview_changed_outside_windows(&before, &after, &[]));
    assert!(!preview_changed_outside_windows(
        &before,
        &after,
        &[[0., 0., 5., 4.]]
    ));
}

fn sample_capture(color: [u8; 4]) -> oblikovati_viewport::ViewportCapture {
    oblikovati_viewport::ViewportCapture {
        view_id: 1,
        scene_generation: 0,
        camera_generation: 0,
        surface_generation: 0,
        overlay_generation: 0,
        extent: [5, 4],
        rgba: color.repeat(20),
    }
}
