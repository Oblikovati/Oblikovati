// SPDX-License-Identifier: GPL-2.0-only
//! Change ribbon display while a panel context menu is open.
use super::*;

#[given(
    "the display control is configured to toggle between the full ribbon and compact group buttons"
)]
fn compact_policy_with_full_ribbon(world: &mut ApplicationWorld) {
    click_display_menu(world);
    world.click_text("Minimize to Panel Buttons");
    activate_display_control(world);
    world.expect_text("Start 2D Sketch");
}
#[then("the panel context menu opens")]
fn context_menu_visible(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
    world.expect_text("Ribbon Appearance");
}
#[when("the user clicks the ribbon display control to restore the full ribbon")]
#[when("the user clicks the ribbon display control")]
#[when("the user clicks the ribbon display control again")]
fn native_display_click(world: &mut ApplicationWorld) {
    let at = super::display_control_point(world);
    super::super::x11_input::click(at);
    world.pump(Vec::new());
}
#[then("the context menu remains closed")]
fn context_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panels");
}
#[then("the 3D Model ribbon shows compact group buttons")]
fn model_compact(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_no_text("Start 2D Sketch");
}
#[then("the full 3D Model ribbon returns")]
fn model_full(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Extrude");
}

#[test]
fn dragging_off_display_control_after_menu_dismissal_cancels_toggle() {
    let mut world = ApplicationWorld::new();
    world.previous_document = Some(world.add_part());
    compact_policy_with_full_ribbon(&mut world);
    let title = super::super::diagnostics::pointer_pixels(world.run("Modify").center());
    super::super::x11_input::right_click(title);
    world.pump(Vec::new());
    context_menu_visible(&mut world);
    let control = super::display_control_point(&world);
    super::super::x11_input::left_button(control, true);
    world.pump(Vec::new());
    super::super::x11_input::move_pointer([1000, 600]);
    world.pump(Vec::new());
    super::super::x11_input::left_button([1000, 600], false);
    world.pump(Vec::new());
    model_full(&mut world);
    context_menu_closed(&mut world);
    world.close();
}
