// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
#[cfg(target_os = "linux")]
#[path = "block_close_native.rs"]
mod block_close_native;
#[cfg(target_os = "linux")]
#[path = "browser_bottom_native.rs"]
mod browser_bottom_native;
#[path = "browser_branch_scenario.rs"]
mod browser_branch_scenario;
#[path = "browser_feature.rs"]
pub(super) mod browser_feature;
#[path = "browser_origin.rs"]
pub(super) mod browser_origin;
#[path = "browser_search_axes.rs"]
mod browser_search_axes;
#[cfg(target_os = "linux")]
#[path = "browser_sketch_native.rs"]
mod browser_sketch_native;
#[path = "browser_tree.rs"]
pub(super) mod browser_tree;
#[path = "dock_splitter.rs"]
mod dock_splitter;
#[cfg(target_os = "linux")]
#[path = "dockspace_native.rs"]
pub(super) mod dockspace_native;
#[cfg(target_os = "linux")]
#[path = "dockspace_restart.rs"]
mod dockspace_restart;
#[path = "drawing_browser.rs"]
mod drawing_browser;
#[path = "drawing_browser_steps.rs"]
mod drawing_browser_steps;
#[cfg(target_os = "linux")]
#[path = "external_close_native.rs"]
mod external_close_native;
#[cfg(target_os = "linux")]
#[path = "representation_preference_native.rs"]
mod representation_preference_native;
use drawing_browser_steps::{drag_over_drawing, drawing_tree, expect_sheet_at, floats_over_sheet};
#[cfg(target_os = "linux")]
#[path = "link_close_native.rs"]
mod link_close_native;
#[cfg(target_os = "linux")]
#[path = "model_dock_native.rs"]
mod model_dock_native;
#[path = "orientation_close_native.rs"]
mod orientation_close_native;
#[cfg(target_os = "linux")]
#[path = "origin_focus_native.rs"]
mod origin_focus_native;
#[cfg(target_os = "linux")]
#[path = "precise_close_native.rs"]
mod precise_close_native;
#[cfg(target_os = "linux")]
#[path = "scale_native.rs"]
mod scale_native;
#[cfg(target_os = "linux")]
#[path = "wheel_close_native.rs"]
mod wheel_close_native;
#[path = "wireframe_tabs.rs"]
mod wireframe_tabs;
const BROWSER: &str = "Model";
fn browser_geometry() -> ([f32; 2], [f32; 2]) {
    window_geometry(BROWSER).expect("model browser: expected a drawn window")
}
fn drag_header(world: &mut ApplicationWorld, to: [i32; 2]) {
    // The title bar, not the Model tab with the same words below it.
    let (position, _) = browser_geometry();
    let from = [position[0] as i32 + 40, position[1] as i32 + 9];
    #[cfg(target_os = "linux")]
    dockspace_native::native_drag(world, from, to);
    #[cfg(not(target_os = "linux"))]
    world.drag(from, to);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn expect_docked_left(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    assert_eq!(position[0], 0., "browser: expected the left dock");
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
#[given("the 3D Model ribbon is active")]
#[given("the 3D Model ribbon tab is active")]
fn model_tab_active(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("Start 2D Sketch");
}
#[when("I select the View ribbon tab")]
fn select_view(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("the ribbon displays view, appearance, window and navigation controls")]
#[then("the ribbon displays Visibility, Appearance and Windows panels")]
fn view_groups(world: &mut ApplicationWorld) {
    for group in ["Visibility", "Appearance", "Windows", "Navigate"] {
        world.expect_text(group);
    }
}
#[then("the viewport navigation controls remain at the right edge")]
fn navigation_controls_on_right(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    for name in ["##orientation-control", "##navigation-bar"] {
        assert_navigation_on_right(name, area[0] as f32, area[2] as f32);
    }
}
fn assert_navigation_on_right(name: &str, left: f32, right: f32) {
    let (position, size) = window_geometry(name).expect("expected viewport navigation control");
    let control_right = position[0] + size[0];
    assert!(
        control_right <= right && control_right >= left + (right - left) * 0.9,
        "navigation control {name} at {control_right}: expected viewport right edge {right}"
    );
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
#[when("the user opens the User Interface menu from the View ribbon tab")]
fn open_user_interface(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}
/// The option leaves its menu open, so reopening only opens what has since closed.
#[when("I reopen the User Interface menu")]
#[when("the user reopens the User Interface menu")]
fn reopen_user_interface(world: &mut ApplicationWorld) {
    if !world.shows("Model Browser") {
        open_user_interface(world);
    }
}
#[when("I clear the Model Browser option")]
#[when("the user clears the Model Browser option")]
#[when("I select the Model Browser option again")]
#[when("the user selects the Model Browser option again")]
fn toggle_browser(world: &mut ApplicationWorld) {
    world.click_text("Model Browser");
}
#[then("the model browser is hidden")]
fn browser_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Origin");
}
#[then("the viewport extends into the space it occupied")]
#[then("the drawing area extends into the space it occupied")]
#[then("the sketch viewport extends into the freed space")]
#[then("the viewport occupies the space previously used by the docked browser")]
pub(super) fn left_space_free(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert!(
        area[0] < 20,
        "viewport must extend into the former browser space"
    );
    world.expect_blank_region([40, 290], [area[1] as usize + 40, area[1] as usize + 160]);
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
#[when("the user drags the model browser by its header into the viewport")]
#[when("I drag the model browser by its header into the viewport")]
fn drag_into_viewport(world: &mut ApplicationWorld) {
    drag_header(world, [800, 300]);
}
#[then("the model browser becomes a floating panel over the viewport")]
fn browser_floating(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert_eq!(super::diagnostics::window_dock_id("Model"), Some(0));
    assert!(position[0] < area[2] as f32 && position[0] + size[0] > area[0] as f32);
    assert!(position[1] < area[3] as f32 && position[1] + size[1] > area[1] as f32);
    world.expect_text("Origin");
}

#[when("the user drags the floating browser by its header to the left edge of the workspace")]
#[when("the user drags its header to the left edge of the workspace")]
#[when("I drag the floating browser by its header to the left edge of the workspace")]
#[when("I drag its header to the left edge of the workspace")]
#[when("I drag the floating browser header to the left edge of the workspace")]
#[when("the user drags the floating browser header to the left edge of the workspace")]
fn drag_left(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    drag_header(world, [15, ((area[1] + area[3]) / 2) as i32]);
}
#[when("the user drags its header to the right edge of the workspace")]
#[when("I drag its header to the right edge of the workspace")]
fn drag_right(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    drag_header(
        world,
        [extent[0] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
}
#[then("the viewport occupies the remaining workspace")]
fn right_space_free(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert!(area[0] > 0, "browser must reserve the left edge");
    world.expect_blank_region(
        [area[0] as usize + 40, area[0] as usize + 240],
        [area[1] as usize + 40, area[1] as usize + 160],
    );
}

#[then("the model browser docks at the right edge")]
fn browser_right(world: &mut ApplicationWorld) {
    let (position, size) = browser_geometry();
    let width = world.extent()[0] as f32;
    assert_eq!(
        position[0] + size[0],
        width,
        "browser: expected the right dock"
    );
}
#[then("the viewport occupies the workspace to its left")]
fn view_left_of_browser(world: &mut ApplicationWorld) {
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert!(area[2] < extent[0], "browser must reserve the right edge");
    world.expect_blank_region(
        [area[0] as usize + 40, area[0] as usize + 240],
        [area[1] as usize + 40, area[1] as usize + 160],
    );
}

fn viewport_left(world: &mut ApplicationWorld) -> f32 {
    let extent = world.extent();
    world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap()[0] as f32
}
#[given("the model browser is docked to the left of the viewport")]
fn docked_left_of_viewport(world: &mut ApplicationWorld) {
    expect_docked_left(world);
    world.browser_width = Some(browser_geometry().1[0]);
    world.strip_left = Some(viewport_left(world));
}
/// Press the boundary the Model window's right edge draws, and drag it sideways.
fn drag_boundary(world: &mut ApplicationWorld, delta: i32) {
    let (position, size) = browser_geometry();
    let from = [
        (position[0] + size[0]) as i32,
        (position[1] + size[1] / 2.) as i32,
    ];
    world.drag(from, [from[0] + delta, from[1]]);
    world.pump(Vec::new());
}
#[when("I drag the boundary between the model browser and viewport to the right")]
#[when("the user drags the boundary between the model browser and viewport to the right")]
fn drag_right_boundary(world: &mut ApplicationWorld) {
    drag_boundary(world, 90);
}
#[then("the model browser becomes wider")]
fn browser_wider(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    assert!(
        now > before + 40.,
        "browser width {now}: expected more than {before}"
    );
}
#[then("the viewport becomes narrower")]
fn viewport_narrower(world: &mut ApplicationWorld) {
    let before = world.strip_left.expect("expected a starting strip");
    let now = viewport_left(world);
    assert!(
        now > before + 40.,
        "document area starts at {now}: expected past {before}"
    );
}
#[then("the document and model tree remain displayed")]
fn tree_remains(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[when("I drag the boundary back to its original position")]
#[when("the user drags the boundary back to its original position")]
fn drag_back(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    drag_boundary(world, (before - now) as i32);
}
#[then("the model browser and viewport return to their previous widths")]
fn widths_back(world: &mut ApplicationWorld) {
    let before = world.browser_width.expect("expected a starting width");
    let now = browser_geometry().1[0];
    assert!(
        (now - before).abs() <= 1.,
        "browser width {now}: expected {before}"
    );
    let strip = world.strip_left.expect("expected a starting strip");
    let left = viewport_left(world);
    assert!(
        (left - strip).abs() <= 1.,
        "document area {left}: expected {strip}"
    );
}
