// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-orientation`: the cube in the view's corner, and its options.
use super::workspace_layout::browser_feature::open_extruded_part;
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_gui::GuiOrientationLayout;
use oblikovati_viewport::ViewportPhysicalKey;

#[path = "navigation.rs"]
mod navigation;
use navigation::{front_displayed, label_turned, taller, top_face_on};

#[path = "appearance.rs"]
mod appearance;
#[path = "behaviour.rs"]
mod behaviour;
#[path = "compass_observation.rs"]
mod compass_observation;
pub(super) use behaviour::whole_block_visible;
#[path = "native_input.rs"]
mod native_input;
#[path = "option_rows.rs"]
mod option_rows;
#[path = "selection_pivot_regression.rs"]
mod selection_pivot_regression;
#[path = "wireframe.rs"]
mod wireframe;

const CONTROL: &str = "##orientation-control";
const NAVIGATION: &str = "##navigation-bar";
const OPTIONS_TITLE: &str = "Orientation Options";
/// Letters used by the compass around the control.
const COMPASS_LETTERS: [&str; 4] = ["N", "E", "S", "W"];
const PIXEL_ROUNDING: usize = 2;
const AWAY: [i32; 2] = [420, 300];
/// A press on a face's middle, away from where its rim picks an edge or a corner.
const IN_THE_CORNER: f32 = 0.85;

fn control_geometry() -> ([f32; 2], [f32; 2]) {
    diagnostics::window_geometry(CONTROL).expect("expected the orientation control")
}
fn layout() -> ([f32; 2], GuiOrientationLayout) {
    let (position, size) = control_geometry();
    (position, GuiOrientationLayout::for_side(size[0]))
}
fn press_at(world: &mut ApplicationWorld, position: [f32; 2], local: [f32; 2]) {
    world.click([
        (position[0] + local[0]) as i32,
        (position[1] + local[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
pub(super) fn press_part(
    world: &mut ApplicationWorld,
    part: fn(&GuiOrientationLayout) -> [f32; 2],
) {
    let (position, parts) = layout();
    press_at(world, position, part(&parts));
}
/// Letters of a label drawn on the control, whichever way it is turned.
fn control_runs(world: &ApplicationWorld) -> Vec<diagnostics::RenderedRun> {
    let (position, layout) = layout();
    let (_, size) = control_geometry();
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            let center = run.center();
            let in_control = (0..2).all(|axis| {
                center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
            });
            let in_cube = (0..2).all(|axis| {
                (center[axis] - position[axis] - layout.center[axis]).abs() <= layout.half
            });
            in_control && (in_cube || !COMPASS_LETTERS.contains(&run.text.as_str()))
        })
        .cloned()
        .collect()
}
pub(super) fn control_letters(world: &ApplicationWorld) -> String {
    let mut letters: Vec<char> = control_runs(world)
        .iter()
        .flat_map(|run| run.text.chars().collect::<Vec<_>>())
        .collect();
    letters.sort_unstable();
    letters.into_iter().collect()
}
fn sorted(text: &str) -> String {
    let mut letters: Vec<char> = text.to_ascii_uppercase().chars().collect();
    letters.sort_unstable();
    letters.into_iter().collect()
}
pub(super) fn expect_control_shows(world: &ApplicationWorld, label: &str) {
    assert_eq!(control_letters(world), sorted(label), "control's label");
}
/// A label reads across or down: its letters spread further one way than the other.
fn label_is_upright(world: &ApplicationWorld) -> bool {
    let runs = control_runs(world);
    let minimum = |axis: usize| {
        runs.iter()
            .map(|run| run.minimum[axis])
            .fold(f32::MAX, f32::min)
    };
    let maximum = |axis: usize| {
        runs.iter()
            .map(|run| run.maximum[axis])
            .fold(f32::MIN, f32::max)
    };
    maximum(0) - minimum(0) > maximum(1) - minimum(1)
}
/// The solid's extent on screen, read from its colour.
/// Zero while nothing shaded is in the view.
pub(super) fn solid_extent_or_zero(world: &mut ApplicationWorld) -> [usize; 2] {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    if super::solid_visibility::visible_fixture_bounds(&capture, &windows).is_none() {
        return [0, 0];
    }
    solid_extent(world)
}
pub(super) fn solid_extent(world: &mut ApplicationWorld) -> [usize; 2] {
    let (low, high) = solid_bounds(world);
    [high[0] - low[0] + 1, high[1] - low[1] + 1]
}
pub(super) fn solid_bounds(world: &mut ApplicationWorld) -> ([usize; 2], [usize; 2]) {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    super::solid_visibility::visible_fixture_bounds(&capture, &windows)
        .expect("expected the solid in the viewport outside GUI windows")
}
/// Sides in this proportion, give or take the pixels an edge rounds to.
fn expect_proportion(wide: usize, tall: usize, across: usize, down: usize) {
    let difference = (wide * down).abs_diff(tall * across);
    assert!(
        difference <= PIXEL_ROUNDING * (across + down),
        "extent {wide} by {tall}: expected {across} to {down}"
    );
}
pub(super) fn expect_same_part(world: &mut ApplicationWorld) {
    let active = world.active_id();
    let remembered = *world.parts.last().expect("expected the fixture part");
    assert_eq!(active, remembered, "active part");
    world.expect_text("Origin");
}
pub(super) fn open_block(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    let id = world.active_id();
    world.parts.push(id);
}
fn press_top_in_iso(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(
        world,
        position,
        [
            parts.center[0],
            parts.center[1] - parts.half * IN_THE_CORNER,
        ],
    );
}
/// The middle of the cube picks Front while the view is on Front, as a part opens.
fn press_at_front(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(world, position, parts.center);
}
fn open_options(world: &mut ApplicationWorld) {
    world.observed_view_camera = Some(super::viewport_pan::camera(world));
    world.observed_rows = Some(control_letters(world));
    let (position, parts) = layout();
    world.right_click([
        (position[0] + parts.center[0]) as i32,
        (position[1] + parts.center[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.click_text("Options");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text(OPTIONS_TITLE);
}
fn choose_list_row(world: &mut ApplicationWorld, label: &str, row: &str) {
    let field = world.run(label).clone();
    world.click([field.minimum[0] as i32 - 40, field.center()[1] as i32]);
    world.pump(Vec::new());
    world.pump(Vec::new());
    if !row.is_empty() {
        world.click_text(row);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}
fn click_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn confirm(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
// Options
#[given("its compass preference is off")]
fn compass_off(world: &mut ApplicationWorld) {
    appearance::expect_no_compass(world);
}
#[when("I open the orientation control context menu and choose Options")]
#[when("the user opens the orientation control context menu and chooses Options")]
#[when("I open the orientation control options")]
#[when("the user opens the orientation control options")]
fn context_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the options dialog displays controls for position, size and inactive opacity")]
fn placement_controls(world: &mut ApplicationWorld) {
    for row in ["On-screen Position", "Size", "Inactive Opacity"] {
        world.expect_text(row);
    }
}
#[then("it displays view transition and orientation settings")]
fn behavior_controls(world: &mut ApplicationWorld) {
    for row in ["Use animated transitions", "Keep model upright"] {
        world.expect_text(row);
    }
}
#[then("the compass option is unchecked and the north angle field is unavailable")]
#[then("the compass option is unchecked")]
fn compass_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Show the compass"), "expected it unchecked");
    expect_north_angle(world, false);
}
fn expect_north_angle(world: &ApplicationWorld, available: bool) {
    assert_eq!(
        world.run_is_enabled("Angle of North"),
        available,
        "Angle of North"
    );
}
#[then("the north angle field is unavailable")]
fn north_off(world: &mut ApplicationWorld) {
    expect_north_angle(world, false);
}
#[when("I enable the compass option")]
#[when("the user enables the compass option")]
fn enable_compass(world: &mut ApplicationWorld) {
    world.click_text("Show the compass");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the north angle field becomes available")]
fn north_on(world: &mut ApplicationWorld) {
    expect_north_angle(world, true);
}
#[when("I cancel the options dialog and reopen it")]
#[when("the user cancels the options dialog and reopens it")]
fn cancel_reopen(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.pump(Vec::new());
    world.pump(Vec::new());
    open_options(world);
}
// Position
#[given("the orientation control and navigation bar are at the upper right of the viewport")]
fn at_upper_right(world: &mut ApplicationWorld) {
    expect_on_right(world);
}
fn expect_on_right(_: &mut ApplicationWorld) {
    let (control, _) = control_geometry();
    let (bar, _) = diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    assert!(control[0] > 700. && bar[0] > 700., "{control:?} {bar:?}");
}
#[when("I open the on-screen position list")]
#[when("the user opens the on-screen position list")]
fn open_position_list(world: &mut ApplicationWorld) {
    choose_list_row(world, "On-screen Position", "");
}
#[then("the list offers Top Right, Bottom Right, Top Left and Bottom Left")]
fn position_rows(world: &mut ApplicationWorld) {
    for row in ["Top Right", "Bottom Right", "Top Left", "Bottom Left"] {
        world.expect_text(row);
    }
}
#[when("I choose Top Left and confirm the options")]
#[when("the user chooses Top Left and confirms the options")]
fn choose_top_left(world: &mut ApplicationWorld) {
    world.click_text("Top Left");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[then("the orientation control appears at the upper left of the viewport")]
fn at_upper_left(_: &mut ApplicationWorld) {
    let (control, _) = control_geometry();
    assert!(control[0] < 700. && control[1] < 400., "{control:?}");
}
#[then("the navigation bar appears on the left side of the viewport")]
fn bar_left(_: &mut ApplicationWorld) {
    let (bar, _) = diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    assert!(bar[0] < 700., "{bar:?}");
}
#[then("the part remains in its existing view")]
fn existing_view(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_rows
        .clone()
        .expect("expected the earlier view");
    assert_eq!(control_letters(world), earlier, "control's label");
    assert_eq!(
        super::viewport_pan::camera(world),
        world.observed_view_camera.unwrap(),
        "view camera"
    );
    world.expect_blank_region([400, 600], [250, 300]);
}
#[then("the orientation control and navigation bar return to the right side")]
fn returned_right(world: &mut ApplicationWorld) {
    expect_on_right(world);
}
// Size
#[given("the orientation control is visible with its size set to Automatic")]
fn automatic_size(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    world.observed_counts = vec![size[0] as usize];
}
#[when("I open the orientation control options and expand the size list")]
#[when("the user opens the orientation control options and expands the size list")]
fn expand_size(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Size", "");
}
#[then("the list offers Automatic, Tiny, Small, Normal and Large")]
fn size_rows(world: &mut ApplicationWorld) {
    for row in ["Automatic", "Tiny", "Small", "Normal", "Large"] {
        world.expect_text(row);
    }
}
#[when("I choose Large and confirm the options")]
#[when("the user chooses Large and confirms the options")]
fn choose_large(world: &mut ApplicationWorld) {
    world.click_text("Large");
    world.pump(Vec::new());
    world.pump(Vec::new());
    confirm(world);
}
#[then("the orientation control is visibly larger in the viewport")]
fn is_larger(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    assert!(
        size[0] as usize > world.observed_counts[0],
        "side {}",
        size[0]
    );
}
#[then("the orientation control returns to its earlier size in this viewport")]
fn earlier_size(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    assert_eq!(size[0] as usize, world.observed_counts[0], "side");
}
#[when("I reopen the options")]
#[when("the user reopens the options")]
fn reopen_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the on-screen position displays Top Left")]
fn position_top_left(world: &mut ApplicationWorld) {
    world.expect_text("Top Left");
}
#[when("I choose Top Right and confirm")]
#[when("the user chooses Top Right and confirms")]
fn choose_top_right(world: &mut ApplicationWorld) {
    choose_list_row(world, "On-screen Position", "Top Right");
    confirm(world);
}
#[then("the size displays Large")]
fn size_large(world: &mut ApplicationWorld) {
    world.expect_text("Large");
}
#[when("I choose Automatic and confirm")]
#[when("the user chooses Automatic and confirms")]
fn choose_automatic(world: &mut ApplicationWorld) {
    choose_list_row(world, "Size", "Automatic");
    confirm(world);
}
