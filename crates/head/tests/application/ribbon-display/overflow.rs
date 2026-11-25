// SPDX-License-Identifier: GPL-2.0-only
//! Reach ribbon tabs that do not fit at a narrow window width.
use super::*;

#[given("a part is active in the MCAD Software with the full Manage ribbon visible")]
fn part_on_manage(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
    world.click_text("Manage");
    assert!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "Parameters")
            .count()
            > 1
    );
}

#[given("a part is active in the MCAD Software with the full ribbon displayed")]
fn part_with_full_ribbon(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
    world.expect_text("Start 2D Sketch");
}

#[given("View is active but outside the visible tab row because the window is narrow")]
fn active_view_in_overflow(world: &mut ApplicationWorld) {
    world.click_text("View");
    view_in_overflow(world);
    open_overflow(world);
    select_hidden_view(world);
    let command = sketch_visibility_label(world);
    world.expect_text(command);
}

#[given("the application window is narrow enough that View is outside the visible tab row")]
#[given("the narrow application window places View in the tab overflow menu")]
pub(super) fn view_in_overflow(world: &mut ApplicationWorld) {
    use crate::HeadWork;
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(1.25, 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    super::window_compaction::resize_window(world, 640);
    assert!(!view_in_tab_row(world));
    world.expect_text(">>");
}

#[given("a part is active in the MCAD Software with Tools active in tabs-only ribbon display")]
fn part_with_minimized_tools(world: &mut ApplicationWorld) {
    super::super::workspace_layout::part_open(world);
    full_tools(world);
    choose_tabs_mode(world);
}

fn view_in_tab_row(world: &ApplicationWorld) -> bool {
    let [_, y] = super::display_control_point(world);
    world
        .frame
        .runs
        .iter()
        .any(|run| run.text == "View" && (run.center()[1] - y as f32).abs() < 12.)
}

#[when("the user clicks the ribbon tab overflow control")]
#[when("the user opens the ribbon tab overflow menu")]
#[when("the user reopens the ribbon tab overflow menu")]
#[when("the user clicks the overflow control again")]
fn open_overflow(world: &mut ApplicationWorld) {
    let label = world.run(">>");
    let point =
        super::super::diagnostics::pointer_pixels([label.minimum[0] + 4., label.center()[1]]);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}

#[then("a menu lists the tabs that do not fit in the row, including View")]
fn hidden_tabs_listed(world: &mut ApplicationWorld) {
    world.expect_text("View");
    assert!(!view_in_tab_row(world));
}

#[when("the user selects View from the overflow menu")]
fn select_hidden_view(world: &mut ApplicationWorld) {
    let point = super::super::diagnostics::pointer_pixels(world.run("View").center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}

#[when("the user opens the tab overflow menu and selects View")]
#[when("the user selects View from the tab overflow menu")]
fn open_and_select_view(world: &mut ApplicationWorld) {
    open_overflow(world);
    select_hidden_view(world);
}

#[then("the overflow menu closes")]
fn overflow_menu_closed(_: &mut ApplicationWorld) {
    assert!(
        !overflow_is_open(),
        "expected the ribbon overflow popup to close"
    );
}

fn overflow_is_open() -> bool {
    super::super::diagnostics::window_popup_open(
        "##application-ribbon",
        "##ribbon-tab-overflow-menu",
    )
}

#[test]
fn overflow_popup_check_ignores_the_browser_view_row() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    view_in_overflow(&mut world);
    assert!(world.shows("View"), "expected the browser View row");
    assert!(!overflow_is_open(), "expected the popup closed initially");
    open_overflow(&mut world);
    assert!(overflow_is_open(), "expected the overflow popup open");
    world.escape();
    assert!(!overflow_is_open(), "expected Escape to close the popup");
    assert!(
        world.shows("View"),
        "expected the browser View row retained"
    );
    world.close();
}

#[then("the View command groups remain displayed")]
#[then("the overflow menu closes and the View command groups remain displayed")]
fn view_commands_remain(world: &mut ApplicationWorld) {
    let command = sketch_visibility_label(world);
    world.expect_text(command);
    overflow_menu_closed(world);
}

#[when("the user clicks the visible Tools tab")]
#[when("the user selects the visible Tools tab")]
fn click_visible_tools(world: &mut ApplicationWorld) {
    let point = super::super::diagnostics::pointer_pixels(world.run("Tools").center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}

#[then("Tools becomes active and its command groups replace the View groups")]
fn tools_replace_view(world: &mut ApplicationWorld) {
    let command = sketch_visibility_label(world);
    world.expect_no_text(command);
    world.hover([500, 600]);
    let inactive = world.pixel_left_of("Sketch");
    assert_ne!(world.pixel_left_of("Tools"), inactive);
}

#[then("no temporary ribbon command panel opens")]
fn no_temporary_ribbon(_world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}

#[then("the menu closes and the View command groups replace the Manage command groups")]
fn view_commands_replace_manage(world: &mut ApplicationWorld) {
    let command = sketch_visibility_label(world);
    world.expect_text(command);
    assert!(
        !world
            .window_text("##application-ribbon")
            .contains("Parameters")
    );
    overflow_menu_closed(world);
}

#[then("View remains outside the visible tab row at this window width")]
fn view_stays_hidden(world: &mut ApplicationWorld) {
    assert!(!view_in_tab_row(world));
}

#[then("View is visible and active in the tab row")]
#[then("View is visible as the active tab")]
fn view_visible_and_active(world: &mut ApplicationWorld) {
    assert!(view_in_tab_row(world));
    world.hover([1000, 600]);
    let inactive = world.pixel_left_of("Tools");
    assert_eq!(world.pixel_left_of("Manage"), inactive);
    assert_ne!(world.pixel_left_of("View"), inactive);
}

#[then("its command groups and the part remain visible")]
fn view_groups_and_part(world: &mut ApplicationWorld) {
    world.expect_text("User Interface");
    assert_eq!(world.active_id(), world.previous_document.unwrap());
}
