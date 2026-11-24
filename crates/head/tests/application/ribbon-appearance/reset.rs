// SPDX-License-Identifier: GPL-2.0-only
//! Reset Ribbon confirmation and the restored active ribbon.
use super::*;

#[given("a part is active in the MCAD Software with the full Tools ribbon selected")]
fn part_with_tools(world: &mut ApplicationWorld) {
    world.previous_document = Some(open_appearance_part(world));
    select_full_tools(world);
}

#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open and active with the full Tools ribbon visible in the MCAD Software"
)]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open and active with the full Tools ribbon selected in the MCAD Software"
)]
fn saved_part_with_tools(world: &mut ApplicationWorld) {
    super::super::workspace_layout::native_part::open_named_extrusion(world);
    select_full_tools(world);
}

fn select_full_tools(world: &mut ApplicationWorld) {
    select_tab(world, "Tools");
    world.expect_text("Application Options");
}
#[when("the user right-clicks the Options panel title")]
fn options_context(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(title);
    world.expect_text("Ribbon Appearance");
}
#[when("the user opens Ribbon Appearance")]
#[when("the user opens the Ribbon Appearance submenu")]
#[when("the user opens Ribbon Appearance again")]
fn open_appearance(world: &mut ApplicationWorld) {
    if !world.shows("Ribbon Appearance") {
        let panel = if world.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch {
            "Create"
        } else {
            "Options"
        };
        modes_native::open_appearance_from(world, panel);
        return;
    }
    world.click_text("Ribbon Appearance");
}
#[when("the user selects Reset Ribbon")]
fn open_reset(world: &mut ApplicationWorld) {
    world.click_text("Reset Ribbon");
}
#[then("a confirmation says ribbon customizations will be removed")]
fn reset_message(world: &mut ApplicationWorld) {
    let text = world.window_text("Reset Ribbon");
    assert!(
        text.contains("ribbonuserinterfacecustomizationswillberemoved"),
        "{text:?}: expected reset warning"
    );
}
#[when("the user selects No")]
fn cancel_reset(world: &mut ApplicationWorld) {
    world.click_text("No");
}
#[when("the user selects Yes")]
fn confirm_reset(world: &mut ApplicationWorld) {
    world.click_text("Yes");
}
#[then("the confirmation closes without resetting the ribbon")]
fn cancelled_reset(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Reset Ribbon").is_none());
    world.expect_text("Application Options");
}
#[then("the full Tools ribbon and active part remain visible")]
fn tools_and_part(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    assert_eq!(world.state().document, world.previous_document);
}
#[then("the 3D Model tab becomes active with its full command groups")]
fn model_after_reset(world: &mut ApplicationWorld) {
    for label in ["Start 2D Sketch", "Extrude", "Hole", "Work Features"] {
        world.expect_text(label);
    }
}
#[then(
    "the Tools ribbon displays its standard groups with Team Web visible and Tutorials and User Commands hidden"
)]
fn standard_tools_after_reset(world: &mut ApplicationWorld) {
    for label in [
        "Measure",
        "Material and Appearance",
        "Options",
        "Clipboard",
        "Find",
        "Team Web",
    ] {
        world.expect_text(label);
    }
    world.expect_no_text("Tutorials");
    world.expect_no_text("User Commands");
}
