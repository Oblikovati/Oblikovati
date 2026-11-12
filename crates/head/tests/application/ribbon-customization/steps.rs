// SPDX-License-Identifier: GPL-2.0-only
//! Native tests for staged user commands and their applied ribbon actions.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

#[given("a part document is active in The MCAD Software")]
fn active_part(world: &mut ApplicationWorld) {
    world.add_part();
}
#[when("the user right-clicks a ribbon panel title")]
fn open_panel_menu(world: &mut ApplicationWorld) {
    let center = world.run("Create").center();
    world.right_click(diagnostics::pointer_pixels(center));
}
#[when("the user chooses \"Customize User Commands\"")]
fn choose_customize(world: &mut ApplicationWorld) {
    world.click_text("Customize User Commands");
    world.pump(Vec::new());
}
#[then("the customization dialog opens on the \"Ribbon\" tab")]
fn ribbon_tab_open(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(
        world
            .window_text("Customize")
            .contains("Customizeusercommands")
    );
}
#[then("the available commands and target custom panel lists are displayed")]
fn command_lists_visible(world: &mut ApplicationWorld) {
    let content = world.window_text("Customize");
    assert!(content.contains("Availablecommands"));
    assert!(content.contains("Custompanel"));
    assert!(content.contains("AllCommands"));
}
#[then("the target tab is \"Part | 3D Model\"")]
fn model_target(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|3DModel"));
}

#[given(
    "the ribbon customization dialog is open with an empty custom panel for \"Part | 3D Model\""
)]
fn empty_model_customization(world: &mut ApplicationWorld) {
    world.add_part();
    open_panel_menu(world);
    choose_customize(world);
    model_target(world);
}
#[when("the user selects \"2D Sketches\" in the available commands")]
fn select_available_sketches(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}
#[when("the user clicks the right transfer button")]
fn transfer_to_target(world: &mut ApplicationWorld) {
    click_in_customize(world, ">");
}
#[then("\"2D Sketches\" appears in the custom panel list")]
fn sketches_in_target(world: &mut ApplicationWorld) {
    let command = world.run_in_window("Customize", "2DSketches");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(command.center()[0] >= target.minimum[0]);
}
#[then("the target tab selector indicates a pending change")]
fn target_has_pending_change(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|3DModel*"));
}
#[then("\"OK\", \"Cancel\", and \"Apply\" are available")]
fn editing_actions_are_available(world: &mut ApplicationWorld) {
    for action in ["OK", "Cancel", "Apply"] {
        assert_eq!(world.run_in_window("Customize", action).alpha, u8::MAX);
    }
}
#[given("a part is active in the MCAD Software with the Tools ribbon visible")]
fn part_with_tools(world: &mut ApplicationWorld) {
    world.add_part();
    world.click_text("Tools");
}
#[given("the Ribbon customization dialog has an initially empty Part Tools custom panel")]
fn empty_tools_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[when("the user stages \"2D Sketches\" in the custom panel")]
fn stage_sketch_command(world: &mut ApplicationWorld) {
    stage_sketch_visibility(world);
}
#[then("the target indicates a pending change")]
fn tools_pending(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}
#[then("the customization dialog closes")]
fn customization_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
}
#[then("no User Commands panel is displayed on the Tools ribbon")]
fn no_user_commands(world: &mut ApplicationWorld) {
    world.expect_no_text("User Commands");
}
#[when("the user reopens ribbon customization from Tools")]
fn reopen_tools_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
}
#[then("the Part Tools custom panel list is empty")]
fn empty_tools_target(world: &mut ApplicationWorld) {
    let command = world.run_in_window("Customize", "2DSketches");
    let target = world.run_in_window("Customize", "Custompanel");
    assert!(command.center()[0] < target.minimum[0]);
}
#[then("\"2D Sketches\" is available in the All Commands list")]
fn sketches_in_available_commands(world: &mut ApplicationWorld) {
    empty_tools_target(world);
    assert!(world.window_text("Customize").contains("AllCommands"));
}
#[then("the target no longer indicates a pending change")]
fn tools_not_pending(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools"));
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}

fn click_in_customize(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window("Customize", label).center();
    world.click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
fn open_tools_customization(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.click_text("Customize");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools"));
}
fn stage_sketch_visibility(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
    click_in_customize(world, ">");
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}
#[test]
fn customization_apply_adds_a_working_command_and_removal_hides_it() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Apply");
    world.expect_text("User Commands");
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    assert!(world.state().browser.unwrap().sketches[0].visible);
    world.click_text("2D Sketches");
    assert!(!world.state().browser.unwrap().sketches[0].visible);
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "2DSketches");
    click_in_customize(&mut world, "<");
    click_in_customize(&mut world, "Apply");
    world.expect_no_text("User Commands");
    click_in_customize(&mut world, "Close");
    world.close();
}
#[test]
fn customization_cancel_discards_a_staged_command() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Cancel");
    world.expect_no_text("User Commands");
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn escape_discards_customization_without_cancelling_the_active_sketch() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let geometry = world.edited_geometry();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    assert!(world.edited_geometry().same_recipe(&geometry));
    world.expect_no_text("User Commands");
    open_tools_customization(&mut world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
    click_in_customize(&mut world, "Close");
    world.close();
}

#[test]
fn escape_closes_the_target_list_before_the_customization_dialog() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Part|Tools*");
    world.expect_text("Part | Annotate");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    assert!(diagnostics::window_geometry("Customize").is_none());
    world.close();
}
