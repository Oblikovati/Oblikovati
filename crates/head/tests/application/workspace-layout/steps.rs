// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
const BROWSER: &str = "Model";
fn browser_geometry() -> ([f32; 2], [f32; 2]) {
    window_geometry(BROWSER).expect("model browser: expected a drawn window")
}
fn drag_header(world: &mut ApplicationWorld, to: [i32; 2]) {
    let from = world.run(BROWSER).center().map(|value| value as i32);
    world.drag(from, to);
}
fn expect_docked_left(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    assert_eq!(position[0], 8., "browser: expected the left dock");
    let height = world.extent()[1] as f32;
    assert!(
        size[1] > height / 2.,
        "browser {size:?}: expected full height"
    );
}
#[given("a part document is open in the MCAD Software")]
fn part_open(world: &mut ApplicationWorld) {
    world.app().owner.create_part_document().unwrap();
    world.previous_document = world.app().owner.gui_command_state().unwrap().document;
    world.pump(Vec::new());
}
#[given("the 3D Model ribbon tab is active")]
fn model_tab_active(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("Sketch XY");
}
#[when("I select the View ribbon tab")]
fn select_view(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("the ribbon displays view, appearance, window and navigation controls")]
fn view_groups(world: &mut ApplicationWorld) {
    for group in ["Visibility", "Appearance", "Windows", "Navigate"] {
        world.expect_text(group);
    }
}
#[then("the active document remains displayed")]
fn document_displayed(world: &mut ApplicationWorld) {
    let active = world.app().owner.gui_command_state().unwrap().document;
    assert!(active.is_some() && active == world.previous_document);
    world.expect_text("End of Part");
}
#[given("the model browser is visible at the left of the workspace")]
#[given("the model browser is docked to the left edge of the workspace")]
fn browser_left(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[when("I open the User Interface menu from the View ribbon tab")]
fn open_user_interface(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}
#[when("I clear the Model Browser option")]
#[when("I select the Model Browser option again")]
fn toggle_browser(world: &mut ApplicationWorld) {
    world.click_text("Model Browser");
}
#[then("the model browser is hidden")]
fn browser_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Origin");
}
#[then("the drawing area extends into the space it occupied")]
#[then("the viewport occupies the space previously used by the docked browser")]
fn left_space_free(world: &mut ApplicationWorld) {
    let height = world.extent()[1] as usize;
    world.expect_blank_region([20, 290], [height - 200, height - 20]);
}
#[then("the Model Browser option is unchecked")]
fn option_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Model Browser"), "expected a cleared option");
}
#[then("the Model Browser option is checked")]
fn option_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Model Browser"), "expected a checked option");
}
#[then("the model browser returns to the left of the workspace")]
#[then("the model browser docks at the left edge")]
fn browser_returned(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    expect_docked_left(world);
}
#[when("I drag the model browser by its header into the viewport")]
fn drag_into_viewport(world: &mut ApplicationWorld) {
    drag_header(world, [800, 300]);
}
#[then("the model browser becomes a floating panel over the viewport")]
fn browser_floating(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let extent = world.extent();
    assert!(
        position[0] > 400.,
        "browser {position:?}: expected to float over the view"
    );
    assert!(
        size[1] < extent[1] as f32 / 2.,
        "browser {size:?}: expected a compact panel"
    );
    world.expect_text("Origin");
}
#[when("I drag the floating browser by its header to the left edge of the workspace")]
#[when("I drag its header to the left edge of the workspace")]
fn drag_left(world: &mut ApplicationWorld) {
    drag_header(world, [10, 400]);
}
#[when("I drag its header to the right edge of the workspace")]
fn drag_right(world: &mut ApplicationWorld) {
    let width = world.extent()[0] as i32;
    drag_header(world, [width - 10, 400]);
}
#[then("the viewport occupies the remaining workspace")]
fn right_space_free(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.expect_blank_region(
        [330, extent[0] as usize - 20],
        [250, extent[1] as usize - 20],
    );
}
#[then("the model browser docks at the right edge")]
fn browser_right(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let width = world.extent()[0] as f32;
    assert_eq!(
        position[0] + size[0],
        width - 8.,
        "browser: expected the right dock"
    );
}
#[then("the viewport occupies the workspace to its left")]
fn view_left_of_browser(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.expect_blank_region(
        [20, extent[0] as usize - 330],
        [250, extent[1] as usize - 20],
    );
}
