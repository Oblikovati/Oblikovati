// SPDX-License-Identifier: GPL-2.0-only
//! User Commands is a normal ribbon panel, including its title menu.
use super::*;
#[path = "shell_customization.rs"]
mod shell_customization;

#[given("the Part Tools custom panel target is initially empty")]
fn empty_tools_panel(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
#[when("the user opens Customize from Tools")]
#[when("the user opens Customize from Tools again")]
#[when("the user reopens Customize from Tools")]
fn open_customize_from_tools(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[when("the user adds 2D Sketches to the Part Tools target")]
#[when("the user adds 2D Sketches to the target list")]
fn add_sketch_visibility(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
}
#[when("the user confirms the customization")]
fn confirm_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "OK");
}
#[then("a User Commands panel is visible on Tools")]
#[then("the User Commands panel is visible behind the dialog")]
#[then("the User Commands panel remains visible behind the dialog")]
fn user_panel_visible(world: &mut ApplicationWorld) {
    world.expect_text("User Commands");
}
#[when("the user right-clicks the User Commands panel title")]
fn open_user_panel_menu(world: &mut ApplicationWorld) {
    let title = world.run("User Commands").center();
    world.right_click(diagnostics::pointer_pixels(title));
}
#[then(
    "the context menu offers Ribbon Appearance, Show Panels, Customize User Commands, Undock Ribbon, and Docking Positions"
)]
fn user_panel_menu_entries(world: &mut ApplicationWorld) {
    for label in [
        "Ribbon Appearance",
        "Show Panels",
        "Customize User Commands",
        "Undock Ribbon",
        "Docking Positions",
    ] {
        world.expect_text(label);
    }
}
#[when("the user selects Customize User Commands")]
#[when("the user chooses Customize User Commands")]
fn choose_customize_from_title(world: &mut ApplicationWorld) {
    choose_customize(world);
}
#[then("the Ribbon customization dialog opens with Part Tools selected")]
#[then("the Ribbon customization dialog opens with Part Tools as the target")]
fn tools_target_selected(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools"));
}
#[then("the target list contains 2D Sketches without a pending marker")]
#[then("the target list retains 2D Sketches without a pending marker")]
fn applied_sketch_command(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}
#[when("the user removes 2D Sketches from the target list")]
#[when("the user removes 2D Sketches from the Part Tools target")]
fn remove_sketch_command(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    click_in_customize(world, "<");
}
#[then("the User Commands panel is removed")]
fn user_panel_removed(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}

#[test]
fn part_tools_user_panel_visibility_uses_the_context_menu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    confirm_customization(&mut world);
    toggle_user_panel_from_context(&mut world, true);
    world.expect_no_text("User Commands");
    toggle_user_panel_from_context(&mut world, false);
    world.expect_text("User Commands");
    world.close();
}

fn toggle_user_panel_from_context(world: &mut ApplicationWorld, shown: bool) {
    let options = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(options);
    world.click_text("Show Panels");
    assert_eq!(world.checked("User Commands"), shown);
    world.click_text("User Commands");
    world.pump(Vec::new());
}

#[test]
fn user_command_moves_to_expansion_and_back_across_restart() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    confirm_customization(&mut world);
    world.right_click(diagnostics::pointer_pixels(
        world.run("2D Sketches").center(),
    ));
    world.expect_text("Move to Expanded Panel");
    world.click_text("Move to Expanded Panel");
    world.pump(Vec::new());
    world.expect_no_text("2D Sketches");
    world.reopen_application();
    world.add_part();
    world.add_finished_sketch();
    world.click_text("Tools");
    world.expect_no_text("2D Sketches");
    world.click_text("User Commands");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-UserCommands").is_some());
    world.expect_text("2D Sketches");
    world.click_text("3D Model");
    assert!(diagnostics::window_geometry("##ribbon-additional-UserCommands").is_none());
    world.click_text("Tools");
    world.expect_no_text("2D Sketches");
    world.click_text("User Commands");
    world.expect_text("2D Sketches");
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    world.right_click(diagnostics::pointer_pixels(
        world.run("2D Sketches").center(),
    ));
    world.expect_text("Move to Main Panel");
    world.click_text("Move to Main Panel");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-UserCommands").is_none());
    world.click_text("2D Sketches");
    assert!(world.state().browser.unwrap().sketches[0].visible);
    world.close();
}

#[test]
fn disabled_icon_without_text_has_a_move_context_menu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    confirm_customization(&mut world);
    let title = world.run("User Commands").clone();
    let point = [title.minimum[0] + 10., title.minimum[1] - 56.];
    world.right_click(diagnostics::pointer_pixels(point));
    world.expect_text("Move to Expanded Panel");
    world.click_text("Move to Expanded Panel");
    world.click_text("User Commands");
    let (position, _) = diagnostics::window_geometry("##ribbon-additional-UserCommands").unwrap();
    world.right_click(diagnostics::pointer_pixels([
        position[0] + 14.,
        position[1] + 18.,
    ]));
    world.expect_text("Move to Main Panel");
    world.click_text("Move to Main Panel");
    assert!(diagnostics::window_geometry("##ribbon-additional-UserCommands").is_none());
    world.close();
}

#[test]
fn failed_command_move_reports_error_and_keeps_the_main_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    confirm_customization(&mut world);
    let path = world
        .preferences_directory()
        .join("ribbon-customization.xml");
    std::fs::remove_file(&path).unwrap();
    std::fs::create_dir(&path).unwrap();
    world.right_click(diagnostics::pointer_pixels(
        world.run("2D Sketches").center(),
    ));
    world.click_text("Move to Expanded Panel");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Ribbon Settings Error").is_some());
    let message: String = world
        .window_text("Ribbon Settings Error")
        .chars()
        .filter(|ch| !ch.is_whitespace())
        .collect();
    assert!(message.contains("expectedacompleteatomicfilereplacement"));
    let close = world
        .run_in_window("Ribbon Settings Error", "Close")
        .center();
    world.click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    assert!(path.is_dir());
    world.close();
}

#[test]
fn expanded_user_command_remains_accessible_when_ribbon_is_docked_left() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    toggle_row_option(&mut world, "2DSketches", "Text");
    confirm_customization(&mut world);
    world.right_click(diagnostics::pointer_pixels(
        world.run("2D Sketches").center(),
    ));
    world.click_text("Move to Expanded Panel");
    open_user_panel_menu(&mut world);
    world.click_text("Docking Positions");
    world.click_text("Left");
    world.pump(Vec::new());
    world.click_text("User Commands");
    world.expect_text("2D Sketches");
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    world.close();
}
