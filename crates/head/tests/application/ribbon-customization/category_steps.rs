// SPDX-License-Identifier: GPL-2.0-only
//! Category-list keyboard behavior in the customization dialog.
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[given("there are no pending customization changes")]
fn no_pending_changes(world: &mut ApplicationWorld) {
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}

#[when("the user opens the command category list")]
fn open_category(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
}

#[then("All Commands and other categories are visible in the list")]
fn categories_visible(world: &mut ApplicationWorld) {
    world.expect_text("Macros");
    world.expect_text("Sketch Tab");
}

#[when("the user presses Down")]
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
