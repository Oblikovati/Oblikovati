// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-pan`: Pan as a running command of the navigation bar.
use super::ApplicationWorld;
use super::diagnostics;
use super::viewport_orientation::{expect_same_part, open_block, solid_extent};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportCamera;

const NAVIGATION: &str = "##navigation-bar";
pub(super) const EMPTY_VIEW: [i32; 2] = [420, 300];
const ON_THE_STATUS: &str = "Escape cancels";

/// Press one of the bar's buttons by the letter it carries.
pub(super) fn press_bar(world: &mut ApplicationWorld, letter: &str) {
    let at = world.run_in_window(NAVIGATION, letter).center();
    world.click(at.map(|value| value as i32));
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
    press_bar(world, "P");
}
#[given("a part containing a visible solid is active in the MCAD Software")]
fn visible_solid(world: &mut ApplicationWorld) {
    open_block(world);
    solid_extent(world);
}
#[given("the viewport navigation bar is visible")]
fn bar_visible(_: &mut ApplicationWorld) {
    diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
}
#[when("I select Pan on the navigation bar")]
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
pub(super) fn bar_button_lit(world: &mut ApplicationWorld, letter: &str, other: &str) -> bool {
    let beside = |world: &mut ApplicationWorld, letter: &str| {
        let run = world.run_in_window(NAVIGATION, letter).clone();
        let at = [
            run.minimum[0] as usize - BUTTON_PROBE,
            run.center()[1] as usize,
        ];
        world.pixel_at(at)
    };
    beside(world, letter) != beside(world, other)
}
const BUTTON_PROBE: usize = 3;
#[then("the Pan control is highlighted")]
fn pan_lit(world: &mut ApplicationWorld) {
    assert!(
        bar_button_lit(world, "P", "W"),
        "expected the Pan button lit"
    );
}
