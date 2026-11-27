// SPDX-License-Identifier: GPL-2.0-only
//! Solid feature editing through the viewport scenario (#4038).
use super::acceptance::{
    feature_count, feature_distance, named_solid_edges, open_extrusion_editor,
};
use super::*;

#[given("the MCAD Software is displaying a part in the solid modeling workspace")]
fn open_part(world: &mut ApplicationWorld) {
    new_metric_part(world);
}

#[given("the part contains an editable solid feature")]
pub(super) fn committed_extrusion(world: &mut ApplicationWorld) {
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

#[given("the fixture contains its accepted 8 mm extrusion")]
fn published_edit_fixture(world: &mut ApplicationWorld) {
    super::taper::open_published_part(world);
    world.extrusion_edit_distances = Some([8., 6.]);
}

#[when("the user opens the feature for modification")]
#[when("the user opens that extrusion for modification")]
fn open_editor(world: &mut ApplicationWorld) {
    world.extrusion_edit_distances.get_or_insert([10., 20.]);
    open_extrusion_editor(world);
}

#[given("the user opens the fixture extrusion for modification")]
fn open_published_edit(world: &mut ApplicationWorld) {
    published_edit_fixture(world);
    open_editor(world);
}

#[then("its current selection and parameters are displayed")]
fn restored_parameters(world: &mut ApplicationWorld) {
    world.expect_text("Distance A");
    world.expect_text("1 Profile");
    let browser = world.state().browser.unwrap();
    let seed = browser
        .extrusion_edits
        .values()
        .next()
        .expect("expected the saved Extrusion1 distance");
    world.run_in_window("Extrude", &super::super::world::squeezed(&seed.distance));
    assert_eq!(feature_distance(world), edit_distances(world)[0]);
}

#[when("the user changes a parameter or requested viewport selection")]
fn change_distance(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "12");
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
    replace_extrusion_distance(world, "20");
}

#[when("the user changes its extrusion distance from 8 mm to 6 mm")]
fn change_published_distance(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
    replace_extrusion_distance(world, "6");
}

#[given("the user changes its distance from 8 mm to 6 mm and the changed preview is visible")]
fn change_and_check_published_distance(world: &mut ApplicationWorld) {
    change_published_distance(world);
    changed_preview(world);
}

#[then("the preview reflects the pending modification")]
fn changed_preview(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_distance(world), edit_distances(world)[0]);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    if std::env::var_os("OBK_BDD_SCENARIO").is_some() {
        world.save_capture("solid-extrude-edit-preview-4038");
    }
    assert!(super::super::solid_visibility::visible_replacement_preview_fill(&capture, &windows));
    assert!(preview_changed_outside_windows(
        world.last_capture.as_ref().expect("12 mm preview capture"),
        &capture,
        &windows
    ));
}

#[when("the user confirms the modification")]
fn accept_edit(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
}

#[then("the existing feature is updated in the model browser")]
#[then("the existing extrusion remains in the model browser")]
fn updated_feature(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    let expected = edit_distances(world)[1];
    let actual = feature_distance(world);
    assert!(
        (actual - expected).abs() < 1e-8,
        "depth {actual} versus {expected} mm"
    );
    world.expect_text("Extrusion1");
}

#[then("the updated result appears in the viewport")]
fn updated_result(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
    assert_extrusion_edges(world, edit_distances(world)[1]);
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
    world.click_text("Cancel");
    world.pump(Vec::new());
}

#[then("the edit panel closes")]
fn edit_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Distance A");
    assert!(!world.state().extrude_preview_active);
    assert!(super::super::diagnostics::window_geometry("Revolution").is_none());
    assert!(world.state().revolve_preview.is_none());
}

#[then("the previously committed model remains visible")]
fn original_result(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 1);
    let original = edit_distances(world)[0];
    assert_eq!(feature_distance(world), original);
    assert_extrusion_edges(world, original);
    updated_solid_is_visible(world);
    world.save_capture("solid-extrude-edit-cancelled-4038");
}

fn edit_distances(world: &ApplicationWorld) -> [f64; 2] {
    world
        .extrusion_edit_distances
        .expect("expected an Extrude edit fixture")
}

fn updated_solid_is_visible(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
}

pub(super) fn click_text_native(world: &mut ApplicationWorld, text: &str) {
    let position = super::super::diagnostics::pointer_pixels(world.run(text).center());
    click_at(world, position);
}

pub(super) fn click_at(world: &mut ApplicationWorld, position: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        world.click(position);
    } else {
        super::super::x11_input::click(position);
        world.pump(Vec::new());
    }
}

pub(super) fn replace_distance_native(world: &mut ApplicationWorld, distance: &str) {
    super::super::x11_input::click(distance_a_input(world));
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    super::super::x11_input::type_digits(distance);
    world.pump(Vec::new());
    world.expect_text(distance);
}

// A small changed patch avoids counting isolated raster noise as a geometry update.
const MIN_CHANGED_PREVIEW_PIXELS: usize = 20;

pub(super) fn preview_changed_outside_windows(
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
    use super::super::solid_visibility::{preview_hue, replacement_preview_hue};
    let preview = |pixel: &[u8]| preview_hue(pixel) || replacement_preview_hue(pixel);
    !covered && before != after && (preview(before) || preview(after))
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
    let after = sample_capture([10, 115, 100, 255]);
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
