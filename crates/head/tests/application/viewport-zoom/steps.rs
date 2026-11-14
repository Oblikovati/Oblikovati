// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-zoom`: the wheel, and Zoom as a running command with its menu.
use super::ApplicationWorld;
use super::viewport_orientation::{
    expect_same_part, open_block, solid_bounds, solid_extent, whole_block_visible,
};
use super::viewport_pan::{camera, expect_part_and_solid, press_bar};
use cucumber::{given, then, when};
use oblikovati_viewport::{ViewportCamera, ViewportEvent, ViewportInput, ViewportNavigation};

const ZOOM_OUT_NOTCHES: i32 = 3;
const SMALL: f64 = 5.;
/// A face that fills most of the view reaches at least this far down the window.
const MOST_OF_THE_VIEW: usize = 400;

fn set_camera(world: &mut ApplicationWorld, camera: ViewportCamera) {
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
}
fn wheel(world: &mut ApplicationWorld, notches: i32) {
    let (low, high) = solid_bounds(world);
    let position = [(low[0] + high[0]) as i32 / 2, (low[1] + high[1]) as i32 / 2];
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches,
        position,
    })]);
    world.pump(Vec::new());
}
fn orientation_of(camera: ViewportCamera) -> [[f64; 3]; 2] {
    let delta = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    [delta, camera.up]
}
fn remember_view(world: &mut ApplicationWorld) {
    let extent = solid_extent(world);
    world.observed_view_camera = Some(camera(world));
    world.observed_counts = vec![extent[0], extent[1]];
}
fn fit_the_block(world: &mut ApplicationWorld) {
    crate::HeadWindow::navigate_view(
        &mut world.app().window,
        ViewportNavigation::Fit { margin: 0.1 },
    )
    .unwrap();
    world.pump(Vec::new());
}
fn shrink_the_block(world: &mut ApplicationWorld) {
    fit_the_block(world);
    let view = camera(world);
    set_camera(
        world,
        ViewportCamera {
            scale: view.scale * SMALL,
            ..view
        },
    );
    remember_view(world);
}
fn zoom_menu(world: &mut ApplicationWorld) {
    press_zoom_arrow(world);
}
/// The bar's first dropdown belongs to Zoom.
fn press_zoom_arrow(world: &mut ApplicationWorld) {
    let at = world.run_in_window("##navigation-bar", "v").center();
    world.click(at.map(|value| value as i32));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn choose_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn choose_from_zoom_menu(world: &mut ApplicationWorld, row: &str) {
    zoom_menu(world);
    choose_row(world, row);
}
fn face_middle(world: &mut ApplicationWorld) -> [i32; 2] {
    let (low, high) = solid_bounds(world);
    [(low[0] + high[0]) as i32 / 2, (low[1] + high[1]) as i32 / 2]
}
/// Ensure the click is on the rendered fixture, not merely its bounding rectangle.
fn visible_face_middle(world: &mut ApplicationWorld) -> [i32; 2] {
    let middle = face_middle(world);
    let pixel = world.pixel_at(middle.map(|coordinate| coordinate as usize));
    assert!(
        super::solid_visibility::neutral_fixture_hue(&pixel),
        "face center {middle:?} has pixel {pixel:?}: expected visible fixture fill"
    );
    middle
}
/// Part of the face, a quarter of its extent each way from its middle.
fn window_corner(world: &mut ApplicationWorld, side: i32) -> [i32; 2] {
    let (low, high) = solid_bounds(world);
    let middle = face_middle(world);
    [
        middle[0] + side * (high[0] - low[0]) as i32 / 8,
        middle[1] + side * (high[1] - low[1]) as i32 / 8,
    ]
}
fn click_pump(world: &mut ApplicationWorld, at: [i32; 2]) {
    world.click(at);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn expect_same_orientation(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    assert_eq!(orientation_of(now), orientation_of(earlier), "orientation");
}
// The mouse wheel
#[given("a rectangular block is visible in an active part in the MCAD Software")]
fn visible_block(world: &mut ApplicationWorld) {
    open_block(world);
    remember_view(world);
}
#[given("no navigation command is active")]
fn no_command(world: &mut ApplicationWorld) {
    world.expect_ready();
}
#[when("I scroll the mouse wheel over the block in the direction that zooms out")]
#[when("the user scrolls the mouse wheel over the block in the direction that zooms out")]
fn wheel_out(world: &mut ApplicationWorld) {
    wheel(world, -ZOOM_OUT_NOTCHES);
}
#[then("the block appears smaller in the viewport")]
fn appears_smaller(world: &mut ApplicationWorld) {
    let [wide, _] = solid_extent(world);
    assert!(wide < world.observed_counts[0], "width {wide}");
    world.observed_counts[0] = wide;
}
#[then("the same part and orientation remain active")]
fn part_and_orientation(world: &mut ApplicationWorld) {
    expect_same_part(world);
    expect_same_orientation(world);
}
#[when("I scroll the mouse wheel over the block in the opposite direction")]
#[when("the user scrolls the mouse wheel over the block in the opposite direction")]
fn wheel_in(world: &mut ApplicationWorld) {
    wheel(world, ZOOM_OUT_NOTCHES);
}
#[then("the block appears larger than after zooming out")]
fn appears_larger(world: &mut ApplicationWorld) {
    let [wide, _] = solid_extent(world);
    assert!(wide > world.observed_counts[0], "width {wide}");
}
// The command
#[when("I select Zoom on the navigation bar")]
#[when("the user selects Zoom on the navigation bar")]
fn select_zoom(world: &mut ApplicationWorld) {
    press_bar(world, "Z");
}
#[then("Zoom is active")]
fn zoom_active(world: &mut ApplicationWorld) {
    world.expect_status("Zoom:");
}
// Zoom All
#[given("the block appears small within the viewport")]
#[given("its visible planar face appears small within the viewport")]
fn appears_small(world: &mut ApplicationWorld) {
    shrink_the_block(world);
}
#[when("I open the Zoom menu on the navigation bar")]
#[when("the user opens the Zoom menu on the navigation bar")]
#[when("I reopen the Zoom menu")]
#[when("the user reopens the Zoom menu")]
fn open_zoom_menu(world: &mut ApplicationWorld) {
    zoom_menu(world);
}
#[then("the menu offers Zoom, Zoom All, Zoom Window and Zoom Selected")]
fn zoom_rows(world: &mut ApplicationWorld) {
    for row in ["Zoom", "Zoom All", "Zoom Window", "Zoom Selected"] {
        world.expect_text(row);
    }
}
#[when("I choose Zoom All")]
#[when("the user chooses Zoom All")]
fn choose_zoom_all(world: &mut ApplicationWorld) {
    choose_row(world, "Zoom All");
}
#[then("the block appears larger and its visible outline fits within the viewport")]
fn fitted(world: &mut ApplicationWorld) {
    let [wide, _] = solid_extent(world);
    assert!(wide > world.observed_counts[0], "width {wide}");
    assert!(
        whole_block_visible(world),
        "expected the whole block in view"
    );
}
#[then("Zoom All is marked as the selected action")]
fn all_marked(world: &mut ApplicationWorld) {
    assert!(world.checked("Zoom All"), "expected Zoom All marked");
}
#[when("I select Zoom and press Escape")]
#[when("the user selects Zoom and presses Escape")]
fn zoom_then_escape(world: &mut ApplicationWorld) {
    choose_row(world, "Zoom");
    world.escape();
    world.pump(Vec::new());
}
// Zoom Window
#[given("its visible face occupies most of the viewport with its outline visible")]
#[given("its visible planar face occupies most of the viewport")]
fn face_large(world: &mut ApplicationWorld) {
    fit_the_block(world);
    remember_view(world);
    assert!(
        whole_block_visible(world),
        "expected the whole outline in view"
    );
}
#[when("I choose Zoom Window from the navigation bar Zoom menu")]
#[when("the user chooses Zoom Window from the navigation bar Zoom menu")]
fn choose_window(world: &mut ApplicationWorld) {
    choose_from_zoom_menu(world, "Zoom Window");
}
#[then("the status area prompts for a zoom window")]
fn window_prompt(world: &mut ApplicationWorld) {
    world.expect_status("Zoom Window");
}
#[when("I click one corner within the visible face")]
#[when("the user clicks one corner within the visible face")]
#[when("I click a first corner within the face")]
#[when("the user clicks a first corner within the face")]
fn first_corner(world: &mut ApplicationWorld) {
    let at = window_corner(world, -1);
    click_pump(world, at);
}
#[when("I click an opposite corner within that face")]
#[when("the user clicks an opposite corner within that face")]
fn opposite_corner(world: &mut ApplicationWorld) {
    let at = window_corner(world, 1);
    click_pump(world, at);
}
#[then("the view zooms into the chosen area")]
fn zoomed_in(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    assert!(
        now.scale < earlier.scale,
        "scale {} from {}",
        now.scale,
        earlier.scale
    );
}
#[then("the face extends beyond the viewport")]
fn beyond(world: &mut ApplicationWorld) {
    assert!(
        !whole_block_visible(world),
        "expected the face past the view"
    );
}
#[then("Zoom Window is marked as the selected action")]
fn window_marked(world: &mut ApplicationWorld) {
    assert!(world.checked("Zoom Window"), "expected Zoom Window marked");
}
#[when("I choose Zoom All and then restore Zoom as the selected action")]
#[when("the user chooses Zoom All and then restores Zoom as the selected action")]
fn all_then_zoom(world: &mut ApplicationWorld) {
    choose_row(world, "Zoom All");
    choose_from_zoom_menu(world, "Zoom");
}
// Zoom Selected
#[given("the face is selected")]
fn select_face(world: &mut ApplicationWorld) {
    world.click_text("Select");
    world.click_text("Faces");
    assert_eq!(
        world.state().selection_mode,
        oblikovati_gui::GuiSelectionMode::Faces
    );
    let at = visible_face_middle(world);
    click_pump(world, at);
    assert!(world.state().model_face_chosen, "expected the face chosen");
}
#[when("I choose Zoom Selected from the navigation bar Zoom menu")]
#[when("the user chooses Zoom Selected from the navigation bar Zoom menu")]
fn choose_selected(world: &mut ApplicationWorld) {
    choose_from_zoom_menu(world, "Zoom Selected");
}
fn expect_fills_view(world: &mut ApplicationWorld) {
    let [wide, tall] = solid_extent(world);
    assert!(wide > world.observed_counts[0], "width {wide}");
    assert!(tall > MOST_OF_THE_VIEW, "height {tall}");
}
#[then("the selected face appears larger and fills most of the viewport")]
#[then("the face appears larger and fills most of the viewport")]
fn fills_view(world: &mut ApplicationWorld) {
    expect_fills_view(world);
}
#[then("the face remains selected")]
fn stays_selected(world: &mut ApplicationWorld) {
    assert!(world.state().model_face_chosen, "expected the face chosen");
}
#[then("Zoom Selected is marked as the selected action")]
fn selected_marked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Zoom Selected"),
        "expected Zoom Selected marked"
    );
}
#[given("no model entity is selected")]
fn none_selected(world: &mut ApplicationWorld) {
    assert!(!world.state().model_face_chosen, "expected nothing chosen");
}
#[then("the status area prompts me to select an entity to zoom")]
#[then("the status area prompts the user to select an entity to zoom")]
fn entity_prompt(world: &mut ApplicationWorld) {
    world.expect_status("select an entity to zoom");
}
#[when("I click the visible planar face")]
#[when("the user clicks the visible planar face")]
fn click_face(world: &mut ApplicationWorld) {
    let at = visible_face_middle(world);
    click_pump(world, at);
}
// A cancelled window
#[when("I press Escape before choosing the opposite corner")]
#[when("the user presses Escape before choosing the opposite corner")]
fn escape_window(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the face retains its previous apparent size and position")]
fn unchanged_face(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_eq!(camera(world), earlier, "view");
    expect_part_and_solid(world);
}
#[then("the Zoom control is highlighted")]
fn zoom_lit(world: &mut ApplicationWorld) {
    let lit = super::viewport_pan::bar_button_lit(world, "Z", "W");
    assert!(lit, "expected the Zoom button lit");
}
#[then("the face is no longer highlighted as selected")]
fn face_let_go(world: &mut ApplicationWorld) {
    assert!(!world.state().model_face_chosen, "expected no face chosen");
}

#[cfg(test)]
#[path = "face_selection_regression.rs"]
mod face_selection_regression;
