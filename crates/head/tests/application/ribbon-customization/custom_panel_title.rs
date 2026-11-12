// SPDX-License-Identifier: GPL-2.0-only
//! User Commands is a normal ribbon panel, including its title menu.
use super::*;

#[given("the Part Tools custom panel target is initially empty")]
fn empty_tools_panel(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
#[when("the user opens Customize from Tools")]
fn open_customize_from_tools(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[when("the user adds 2D Sketches to the Part Tools target")]
fn add_sketch_visibility(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
}
#[when("the user confirms the customization")]
fn confirm_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "OK");
}
#[then("a User Commands panel is visible on Tools")]
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
fn choose_customize_from_title(world: &mut ApplicationWorld) {
    choose_customize(world);
}
#[then("the Ribbon customization dialog opens with Part Tools selected")]
fn tools_target_selected(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools"));
}
#[then("the target list contains 2D Sketches without a pending marker")]
fn applied_sketch_command(world: &mut ApplicationWorld) {
    sketches_in_target(world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}
#[when("the user removes 2D Sketches from the target list")]
fn remove_sketch_command(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    click_in_customize(world, "<");
}
#[then("the User Commands panel is removed")]
fn user_panel_removed(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}

fn open_selector(world: &mut ApplicationWorld) {
    let (_, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    world.click([(size[0] - 22.) as i32, 45]);
    world.pump(Vec::new());
}
#[test]
fn user_panel_can_be_hidden_and_restored_from_both_visibility_menus() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    confirm_customization(&mut world);
    world.expect_text("User Commands");
    open_selector(&mut world);
    assert!(world.checked("User Commands"));
    world.click_text("User Commands");
    world.escape();
    world.expect_no_text("User Commands");
    open_selector(&mut world);
    assert!(!world.checked("User Commands"));
    world.click_text("User Commands");
    world.escape();
    world.expect_text("User Commands");
    open_user_panel_menu(&mut world);
    world.click_text("Show Panels");
    assert!(world.checked("User Commands"));
    world.click_text("User Commands");
    world.pump(Vec::new());
    world.expect_no_text("User Commands");
    open_selector(&mut world);
    assert!(!world.checked("User Commands"));
    world.click_text("User Commands");
    world.escape();
    world.expect_text("User Commands");
    world.close();
}
