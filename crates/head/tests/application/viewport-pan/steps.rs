// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-pan`: Pan as a running command of the navigation bar.
use super::ApplicationWorld;
use super::diagnostics;
use super::viewport_orientation::{
    expect_same_part, open_block, open_saved_navigation_block, solid_extent,
};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportCamera;

#[cfg(target_os = "linux")]
#[path = "native_input.rs"]
mod native_input;

const NAVIGATION: &str = "##navigation-bar";
pub(super) const EMPTY_VIEW: [i32; 2] = [420, 300];
const ON_THE_STATUS: &str = "Escape cancels";
const PAN_FIXTURE: &str = "fixtures/cad/body-properties-start.opd";

#[derive(Clone, Copy)]
pub(in super::super) enum NavigationControl {
    Wheel,
    Pan,
    Zoom,
    Orbit,
}

fn bar_icon(control: NavigationControl) -> ([f32; 2], [f32; 2]) {
    let (min, size) =
        diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    let max = [min[0] + size[0], min[1] + size[1]];
    let mut icons = super::ribbon_appearance::icon_quads::icon_quads_inside((min, max));
    icons.sort_by(|left, right| left.0[1].total_cmp(&right.0[1]));
    assert_eq!(icons.len(), 4, "expected four navigation glyphs");
    icons[control as usize]
}

pub(in super::super) fn bar_icon_center(control: NavigationControl) -> [f32; 2] {
    let (min, max) = bar_icon(control);
    [(min[0] + max[0]) / 2., (min[1] + max[1]) / 2.]
}

pub(super) fn bar_dropdown_center(control: NavigationControl) -> [f32; 2] {
    let zoom = bar_icon_center(NavigationControl::Zoom);
    let orbit = bar_icon_center(NavigationControl::Orbit);
    let mut point = bar_icon_center(control);
    point[1] += (orbit[1] - zoom[1]) * 0.5;
    point
}

/// Press a control at the center of its actual drawn icon.
pub(super) fn press_bar(world: &mut ApplicationWorld, control: NavigationControl) {
    world.click(diagnostics::pointer_pixels(bar_icon_center(control)));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn camera(world: &mut ApplicationWorld) -> ViewportCamera {
    crate::HeadWindow::view_camera(&world.app().window).unwrap()
}
pub(super) fn expect_part_and_solid(world: &mut ApplicationWorld) {
    expect_same_part(world);
    solid_extent(world);
}
fn start_pan(world: &mut ApplicationWorld) {
    press_bar(world, NavigationControl::Pan);
}
#[given("a part containing a visible solid is active in the MCAD Software")]
fn visible_solid(world: &mut ApplicationWorld) {
    open_block(world);
    solid_extent(world);
}
#[given(
    "`fixtures/cad/body-properties-start.opd` is open with its solid visible in the MCAD Software"
)]
fn saved_pan_block_visible(world: &mut ApplicationWorld) {
    let path = super::scenario_root().join(PAN_FIXTURE);
    open_saved_navigation_block(world, &path);
}
#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with its solid visible in the MCAD Software"
)]
fn current_pan_block_visible(world: &mut ApplicationWorld) {
    let path = super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    open_saved_navigation_block(world, &path);
}
#[given("the viewport navigation bar is visible")]
fn bar_visible(_: &mut ApplicationWorld) {
    diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
}
#[when("I select Pan on the navigation bar")]
#[when("the user selects Pan on the navigation bar")]
fn select_pan(world: &mut ApplicationWorld) {
    start_pan(world);
}
#[given("Pan is active from the viewport navigation bar")]
fn pan_active_given(world: &mut ApplicationWorld) {
    start_pan(world);
    world.observed_view_camera = Some(camera(world));
}
#[then("Pan is active")]
#[then("the status area still indicates Pan is active")]
fn pan_active(world: &mut ApplicationWorld) {
    world.expect_status("Pan:");
}
#[then(
    "the status area prompts me to pan the current view and indicates cancellation is available"
)]
#[then(
    "the status area prompts the user to pan the current view and indicates cancellation is available"
)]
fn pan_prompt(world: &mut ApplicationWorld) {
    world.expect_status("pan the current view");
    world.expect_status(ON_THE_STATUS);
}
#[then("the active part and solid remain visible")]
#[then("the same part and solid remain visible")]
fn part_and_solid(world: &mut ApplicationWorld) {
    expect_part_and_solid(world);
}
#[then("the status area returns to Ready")]
#[then("the status area displays Ready")]
fn status_ready(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.expect_ready();
}
#[when("I click an empty area of the viewport without dragging")]
#[when("the user clicks an empty area of the viewport without dragging")]
fn click_empty(world: &mut ApplicationWorld) {
    world.click(EMPTY_VIEW);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("Pan ends and the status area displays Ready")]
fn pan_ends(world: &mut ApplicationWorld) {
    world.expect_ready();
}
#[then("the same part and solid remain visible in their previous view")]
fn previous_view(world: &mut ApplicationWorld) {
    expect_part_and_solid(world);
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_eq!(camera(world), earlier, "view");
}
#[when("I right-click an empty area of the viewport")]
#[when("the user right-clicks an empty area of the viewport")]
fn right_click_empty(world: &mut ApplicationWorld) {
    world.right_click(EMPTY_VIEW);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a contextual menu offers Done, OK, Home View, Orbit, Zoom and Previous View")]
fn pan_menu(world: &mut ApplicationWorld) {
    for row in ["Done", "OK", "Home View", "Orbit", "Zoom", "Previous View"] {
        world.expect_text(row);
    }
}
#[then("the contextual menu closes and the status area displays Ready")]
fn menu_closes_ready(world: &mut ApplicationWorld) {
    world.expect_no_text("Previous View");
    world.expect_ready();
}
#[when("I choose Zoom from the contextual menu")]
#[when("the user chooses Zoom from the contextual menu")]
fn zoom_from_menu(world: &mut ApplicationWorld) {
    world.click_text("Zoom");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the contextual menu closes")]
fn menu_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("Previous View");
}
#[then("the status area prompts for zooming and indicates cancellation is available")]
fn zoom_prompt(world: &mut ApplicationWorld) {
    world.expect_status("Zoom:");
    world.expect_status(ON_THE_STATUS);
}
/// A running command's button draws in the pressed colour; its neighbour stays plain.
pub(super) fn bar_button_lit(
    world: &mut ApplicationWorld,
    control: NavigationControl,
    other: NavigationControl,
) -> bool {
    let probe = |control| {
        let (min, max) = bar_icon(control);
        diagnostics::pointer_pixels([min[0] - 1., (min[1] + max[1]) / 2.])
            .map(|coordinate| coordinate as usize)
    };
    world.pixel_at(probe(control)) != world.pixel_at(probe(other))
}
#[then("the Pan control is highlighted")]
fn pan_lit(world: &mut ApplicationWorld) {
    assert!(
        bar_button_lit(world, NavigationControl::Pan, NavigationControl::Wheel),
        "expected the Pan button lit"
    );
}

#[test]
fn native_pan_drag_moves_the_view_and_keeps_pan_active() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    start_pan(&mut world);
    let before = camera(&mut world);
    world.drag(EMPTY_VIEW, [EMPTY_VIEW[0] + 80, EMPTY_VIEW[1] + 40]);
    let after = camera(&mut world);
    assert_pan_translation(before, after);
    world.expect_status("Pan:");
    world.escape();
    world.expect_ready();
    world.close();
}

fn assert_pan_translation(before: ViewportCamera, after: ViewportCamera) {
    assert_ne!(after.target, before.target, "pan must move the target");
    assert_ne!(after.eye, before.eye, "pan must move the camera");
    for axis in 0..3 {
        let eye = after.eye[axis] - before.eye[axis];
        let target = after.target[axis] - before.target[axis];
        assert!((eye - target).abs() < 1e-9);
    }
    assert_eq!(after.up, before.up, "pan must retain orientation");
    assert_eq!(after.scale, before.scale, "pan must retain scale");
}
