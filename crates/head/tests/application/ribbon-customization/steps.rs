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
