// SPDX-License-Identifier: GPL-2.0-only
//! Docking from the lower title of the Modify additional-command area.
use super::*;
use cucumber::{given, then, when};

const MODIFY_AREA: &str = "##ribbon-additional-PartModify";

#[given("a part is active in the MCAD Software with the full 3D Model ribbon docked at the top")]
#[given("a part is active in the MCAD Software with the full 3D Model ribbon at the top")]
#[given("a part is active with the full 3D Model ribbon docked at the top")]
#[given("a part is active with the 3D Model ribbon docked at the top")]
#[given("a part is active and the 3D Model ribbon is docked at the top")]
fn part_with_top_ribbon(world: &mut ApplicationWorld) {
    part_active(world);
    expect_top_dock(world);
}
#[when("the user right-clicks the lower Modify title")]
fn right_click_lower_title(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry(MODIFY_AREA).expect("expected Modify additional area");
    let title = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let center = run.center();
            run.text.contains("Modif")
                && center[0] >= position[0]
                && center[0] <= position[0] + size[0]
                && center[1] >= position[1]
                && center[1] <= position[1] + size[1]
        })
        .expect("expected lower Modify title");
    world.right_click(super::super::diagnostics::pointer_pixels(title.center()));
    world.pump(Vec::new());
}
#[when("the user right-clicks the lower Modify title and opens \"Docking Positions\"")]
#[when("the user opens Docking Positions from the expanded title")]
fn lower_title_docking_menu(world: &mut ApplicationWorld) {
    right_click_lower_title(world);
    open_docking_positions(world);
}
#[when("the user chooses Left from Docking Positions on its title")]
fn choose_left_from_expanded_modify(world: &mut ApplicationWorld) {
    lower_title_docking_menu(world);
    world.click_text("Left");
    world.pump(Vec::new());
}

#[when("the user chooses the checked Top position from Docking Positions on its title")]
fn reselect_top_from_expanded_modify(world: &mut ApplicationWorld) {
    lower_title_docking_menu(world);
    select_current_top(world);
}

#[when("the user chooses Undock Ribbon from its title context menu")]
fn undock_from_expanded_modify(world: &mut ApplicationWorld) {
    right_click_lower_title(world);
    choose_undock(world);
}

#[when("the user opens Modify and clicks empty viewport space")]
fn show_modify_then_click_outside(world: &mut ApplicationWorld) {
    super::super::ribbon_panel_expansion::opened_unpinned(world);
    world.click([1000, 700]);
    world.pump(Vec::new());
}

#[given("a part is active with the 3D Model ribbon docked on the left")]
fn part_with_left_ribbon(world: &mut ApplicationWorld) {
    part_active(world);
    docked_left_on_model(world);
}

#[given("the Modify command area is open")]
fn modify_command_area_open(world: &mut ApplicationWorld) {
    super::super::ribbon_panel_expansion::opened_unpinned(world);
}

#[when("the user selects Left")]
fn select_current_left(world: &mut ApplicationWorld) {
    assert!(world.checked("Left"));
    world.click_text("Left");
    world.pump(Vec::new());
}
#[then("\"Top\" is checked and the Modify additional commands remain visible")]
fn top_checked_with_modify(world: &mut ApplicationWorld) {
    assert!(world.checked("Top"));
    assert!(window_geometry(MODIFY_AREA).is_some());
    world.expect_text("Move Bodies");
}
#[when("the user selects the checked \"Top\" position")]
fn select_current_top(world: &mut ApplicationWorld) {
    assert!(world.checked("Top"));
    world.click_text("Top");
    world.pump(Vec::new());
}
#[then("the menus and Modify additional command area close")]
#[then("the menus and additional command area close")]
#[then("the menus and Modify command area close")]
fn menus_and_modify_close(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
    assert!(window_geometry(MODIFY_AREA).is_none());
}
#[then("the full 3D Model ribbon remains docked at the top")]
#[then("the ribbon remains docked at the top")]
fn full_ribbon_at_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
}

#[then("the 3D Model ribbon is docked vertically on the left")]
#[then("the ribbon tabs and command groups are arranged vertically on the left")]
fn model_on_left(world: &mut ApplicationWorld) {
    tabs_on_left(world);
    groups_beside_tabs(world);
}
#[then("Modify is represented by a compact panel button")]
fn compact_modify(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_no_text("Hole");
}
#[when("the user opens \"Docking Positions\" from the left-docked ribbon context menu")]
#[when("the user opens Docking Positions from the left-docked ribbon")]
#[when("the user opens Docking Positions from the active ribbon tab")]
fn left_context_docking(world: &mut ApplicationWorld) {
    let center = world.run("3D Model").center();
    world.right_click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    open_docking_positions(world);
}
#[when("the user chooses Top from the docked ribbon tab menu")]
fn restore_from_docked_tab(world: &mut ApplicationWorld) {
    left_context_docking(world);
    choose_open_top(world);
}
#[when("the user chooses Top")]
fn choose_open_top(world: &mut ApplicationWorld) {
    world.click_text("Top");
    world.pump(Vec::new());
}
#[then("the full 3D Model ribbon returns to the top")]
#[then("the full ribbon returns to the top")]
fn model_returns_to_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
}
#[then("the ribbon returns to the top with the additional command area closed")]
fn top_without_modify(world: &mut ApplicationWorld) {
    expect_top_dock(world);
    assert!(window_geometry(MODIFY_AREA).is_none());
}

#[test]
fn clicking_outside_the_open_docking_menu_closes_an_unpinned_expansion() {
    let mut world = ApplicationWorld::new();
    part_with_top_ribbon(&mut world);
    super::super::ribbon_panel_expansion::opened_unpinned(&mut world);
    lower_title_docking_menu(&mut world);
    top_checked_with_modify(&mut world);
    world.click([1000, 700]);
    world.pump(Vec::new());
    menus_and_modify_close(&mut world);
    world.close();
}

#[test]
fn pinned_expansion_survives_outside_click_but_not_docking() {
    let mut world = ApplicationWorld::new();
    part_with_top_ribbon(&mut world);
    super::super::ribbon_panel_expansion::opened_unpinned(&mut world);
    super::super::ribbon_panel_expansion::toggle_pin(&mut world);
    lower_title_docking_menu(&mut world);
    top_checked_with_modify(&mut world);
    world.click([1000, 700]);
    world.pump(Vec::new());
    assert!(window_geometry(MODIFY_AREA).is_some());
    lower_title_docking_menu(&mut world);
    world.click_text("Left");
    world.pump(Vec::new());
    assert!(window_geometry(MODIFY_AREA).is_none());
    assert!(window_geometry(COLUMN).is_some());
    world.close();
}
