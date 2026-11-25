// SPDX-License-Identifier: GPL-2.0-only
//! Category-list keyboard behavior in the customization dialog.
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[given("there are no pending customization changes")]
fn no_pending_changes(world: &mut ApplicationWorld) {
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}

#[when("the user opens the command category list")]
#[when("the user opens the command-source list")]
#[when("the user clicks the command-source selector")]
#[when("the user clicks the command-source selector again")]
fn open_category(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
}

#[given("the Ribbon customization dialog has no pending edits")]
fn open_clean_customization(world: &mut ApplicationWorld) {
    open_tools_customization(world);
    no_pending_changes(world);
    for control in ["Apply", "Cancel"] {
        assert_ne!(world.run_in_window("Customize", control).alpha, u8::MAX);
    }
}

#[then("the command-source list closes")]
#[then("the command-source list is not open")]
fn source_list_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Macros");
}

#[then("the customization dialog remains open with All Commands selected")]
fn all_commands_open(world: &mut ApplicationWorld) {
    customization_remains_open(world);
    transaction_steps::all_commands(world);
}

#[when("the user clicks Close again")]
fn close_again(world: &mut ApplicationWorld) {
    close_customization_button(world);
}

#[then("the command-source list opens with All Commands selected")]
fn source_list_selected(world: &mut ApplicationWorld) {
    categories_visible(world);
    transaction_steps::all_commands(world);
}

#[then("All Commands and other categories are visible in the list")]
fn categories_visible(world: &mut ApplicationWorld) {
    world.expect_text("Macros");
    world.expect_text("Sketch Tab");
}

#[when("the user presses Down")]
#[when("the user presses Down again")]
fn down(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::DownArrow, false);
}

#[when("the user opens the command category list and presses Down")]
fn open_and_down(world: &mut ApplicationWorld) {
    open_category(world);
    down(world);
}

#[then("Macros has the keyboard focus in the open list")]
#[then("Macros has keyboard focus in the open category list")]
fn macros_highlighted(world: &mut ApplicationWorld) {
    world.hover([1100, 800]);
    let focused = world.run("Macros").clone();
    let plain = world.run("Sketch Tab").clone();
    let x = focused.maximum[0].max(plain.maximum[0]) + 8.;
    let focused_pixel =
        diagnostics::pointer_pixels([x, focused.center()[1]]).map(|value| value as usize);
    let plain_pixel =
        diagnostics::pointer_pixels([x, plain.center()[1]]).map(|value| value as usize);
    assert_ne!(world.pixel_at(focused_pixel), world.pixel_at(plain_pixel));
}

#[then("the customization dialog and category list close")]
#[then("the category list and customization dialog close")]
fn category_and_dialog_closed(_world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Customize").is_none());
}

#[then("the Tools ribbon and part remain visible")]
fn tools_and_part_remain(world: &mut ApplicationWorld) {
    world.expect_text("Tools");
    assert!(world.state().document.is_some());
}

#[then("the category shows All Commands")]
fn all_commands_restored(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("AllCommands"));
}
