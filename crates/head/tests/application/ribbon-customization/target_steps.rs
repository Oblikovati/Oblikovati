// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard switching between customized ribbon targets.
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[path = "entry_steps.rs"]
mod entry_steps;
#[path = "target_inventory.rs"]
mod target_inventory;

#[given("the Part Manage target is empty")]
fn manage_is_empty(world: &mut ApplicationWorld) {
    click_in_customize(world, "Part|Tools");
    world.click_text("Part | Manage");
    manage_target_is_empty(world);
    click_in_customize(world, "Part|Manage");
    world.click_text("Part | Tools");
}

#[when("the user opens the custom panel target selector")]
#[when("the user opens the target selector")]
#[when("the user opens the target tab selector")]
#[when("the user clicks the target selector")]
#[when("the user clicks the target selector again")]
fn open_target_selector(world: &mut ApplicationWorld) {
    let target = transaction_steps::target_label(world).to_owned();
    click_in_customize(world, &target);
}

#[then("the target list is not open")]
#[then("the target list closes")]
fn target_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Part | 3D Model");
}

#[then("the target list opens with Part Tools selected")]
fn menu_shows_tools(world: &mut ApplicationWorld) {
    world.expect_text("Part | 3D Model");
    world.run_in_window("Customize", "Part|Tools");
}

#[then("the ribbon customization target is \"Part | Tools\"")]
fn target_is_tools(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "Part|Tools");
}

#[when("the user presses Down and Enter to select Part Manage")]
fn select_manage_by_keyboard(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
}

#[when("the user presses Down to highlight Part Manage")]
fn highlight_manage(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::DownArrow, false);
    world.hover([1100, 800]);
    let selected = world.run("Part | Manage").clone();
    let other = world.run("Part | View").clone();
    let x = selected.maximum[0].max(other.maximum[0]) + 8.;
    let selected_at = diagnostics::pointer_pixels([x, selected.center()[1]]).map(|v| v as usize);
    let other_at = diagnostics::pointer_pixels([x, other.center()[1]]).map(|v| v as usize);
    assert_ne!(world.pixel_at(selected_at), world.pixel_at(other_at));
}

#[when("the user presses Up and Enter to select Part Tools")]
fn select_tools_by_keyboard(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::UpArrow, false);
    world.key(ViewportPhysicalKey::Enter, false);
}

#[then("the Part Manage custom panel list is empty")]
fn manage_target_is_empty(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Manage"));
    let source = world.run_in_window("Customize", "2DSketches").center()[0];
    assert!(source < world.run_in_window("Customize", "Custompanel").minimum[0]);
}

#[then("Apply and Cancel are enabled")]
fn pending_actions_enabled(world: &mut ApplicationWorld) {
    for label in ["Apply", "Cancel"] {
        assert_eq!(world.run_in_window("Customize", label).alpha, u8::MAX);
    }
}

#[then("the Part Tools target has a pending-change marker")]
fn tools_pending(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Part|Tools*"));
}
