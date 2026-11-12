// SPDX-License-Identifier: GPL-2.0-only
//! Show and hide the Sketch Layout panel through the native title menu.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

#[given("the Sketch ribbon's Layout panel is hidden")]
#[then("the Layout panel is hidden from the Sketch ribbon")]
fn layout_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Layout");
}
#[when("the user opens Show Panels from the Create panel title context menu")]
fn open_create_show_panels(world: &mut ApplicationWorld) {
    let title = world.run("Create").center();
    world.right_click(diagnostics::pointer_pixels(title));
    world.click_text("Show Panels");
    world.pump(Vec::new());
}
#[then("Layout is listed as unchecked in the panel visibility submenu")]
fn layout_unchecked(world: &mut ApplicationWorld) {
    world.expect_text("Layout");
    assert!(!world.checked("Layout"));
}
#[when("the user selects Layout")]
#[when("the user selects Layout again")]
fn select_layout(world: &mut ApplicationWorld) {
    world.click_text("Layout");
    world.pump(Vec::new());
}
#[then("the Layout panel appears on the Sketch ribbon")]
fn layout_visible(world: &mut ApplicationWorld) {
    world.expect_text("Layout");
}
#[then("Make Part and Make Components are unavailable")]
fn layout_unavailable_commands(world: &mut ApplicationWorld) {
    for label in ["Make Part", "Make Components"] {
        world.expect_text(label);
        assert!(!world.run_is_enabled(label));
    }
}
#[then("Create Block is available")]
fn layout_create_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Create Block"));
}
#[test]
fn layout_panel_visibility_and_create_block_use_the_native_ribbon() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let geometry = world.edited_geometry();
    let before = world.state().undo_labels;
    layout_hidden(&mut world);
    open_create_show_panels(&mut world);
    layout_unchecked(&mut world);
    select_layout(&mut world);
    layout_visible(&mut world);
    layout_unavailable_commands(&mut world);
    layout_create_available(&mut world);
    world.click_text("Create Block");
    assert!(diagnostics::window_geometry("Create Block").is_some());
    let cancel = world.run_in_window("Create Block", "Cancel").center();
    world.click(diagnostics::pointer_pixels(cancel));
    open_create_show_panels(&mut world);
    select_layout(&mut world);
    layout_hidden(&mut world);
    assert!(world.edited_geometry().same_recipe(&geometry));
    assert_eq!(world.state().undo_labels, before);
    world.close();
}
