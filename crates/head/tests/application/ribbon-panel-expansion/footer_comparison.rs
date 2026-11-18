// SPDX-License-Identifier: GPL-2.0-only
//! Compare expanded panel footers in full and temporary ribbons.
use super::{ApplicationWorld, WINDOW, diagnostics};
use cucumber::{then, when};

#[then("its lower title strip shows a pin control and the Modify title")]
fn full_footer(world: &mut ApplicationWorld) {
    let pin = world.run_in_window(WINDOW, "Pin");
    let title = world.run_in_window(WINDOW, "Modify");
    assert!(pin.center()[0] < title.center()[0]);
}
#[when("the user minimizes the ribbon to tabs")]
fn minimize_to_tabs(world: &mut ApplicationWorld) {
    super::super::ribbon_display::click_display_menu(world);
    world.click_text("Minimize to Tabs");
}
#[when("the user opens the temporary 3D Model ribbon")]
fn open_temporary_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
}
#[then("the same three additional commands are visible")]
fn same_commands(world: &mut ApplicationWorld) {
    super::extra_commands(world);
}
#[then("the lower title strip shows the Modify title without a visible pin control")]
fn temporary_footer(world: &mut ApplicationWorld) {
    world.run_in_window(WINDOW, "Modify");
    assert!(!world.window_text(WINDOW).contains("Pin"));
}
#[then("the additional area and the temporary ribbon close")]
fn temporary_areas_close(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}
