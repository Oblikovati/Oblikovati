// SPDX-License-Identifier: GPL-2.0-only
//! Session tab retention through the Tools customization entry point.
use super::*;

#[given("the customization dialog is open on the Marking Menu tab with no pending edits")]
fn marking_menu_open(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.record_initial_sketch();
    open_tools_customization(world);
    click_in_customize(world, "MarkingMenu");
    no_pending(world);
}
#[given("a part with an active sketch is open in The MCAD Software")]
fn sketch_is_active(world: &mut ApplicationWorld) {
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        world.start_empty_sketch();
        world.record_initial_sketch();
    }
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("this is the same application session in which Marking Menu was previously selected")]
fn same_session(world: &mut ApplicationWorld) {
    assert!(
        world
            .window_text("Customize")
            .contains("Environment:2DSketch")
    );
}
#[when("the user selects the \"Keyboard\" tab")]
fn select_keyboard(world: &mut ApplicationWorld) {
    click_in_customize(world, "Keyboard");
}
#[when("the user closes the customization dialog")]
fn close_customization(world: &mut ApplicationWorld) {
    click_in_customize(world, "Close");
    assert!(diagnostics::window_geometry("Customize").is_none());
}
#[when("the user opens Customize again from the Tools Options panel")]
fn reopen_customization(world: &mut ApplicationWorld) {
    world.click_text("Customize");
    world.pump(Vec::new());
}
#[then("the customization dialog opens on the \"Keyboard\" tab")]
fn keyboard_visible(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("Searchcommands"));
}
#[then("the customization dialog opens on the Marking Menu tab")]
fn marking_visible(world: &mut ApplicationWorld) {
    assert!(
        world
            .window_text("Customize")
            .contains("Environment:2DSketch")
    );
}
#[then("Environment remains \"2D Sketch\"")]
fn environment_unchanged(world: &mut ApplicationWorld) {
    marking_visible(world);
}
#[then("Sub-Environment remains \"None Selected\"")]
fn sub_environment_unchanged(world: &mut ApplicationWorld) {
    assert!(
        world
            .window_text("Customize")
            .contains("Sub-Environment:NoneSelected")
    );
}
#[then("no keyboard or ribbon changes are pending")]
#[then("no customization change is pending")]
fn no_pending(world: &mut ApplicationWorld) {
    for label in ["Cancel", "Apply"] {
        assert!(world.run_in_window("Customize", label).alpha < u8::MAX);
    }
}
#[then("the active sketch remains unchanged")]
fn sketch_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    let current = world.edited_geometry();
    let before = &world
        .observed_sketch
        .as_ref()
        .expect("expected the initial sketch")
        .geometry;
    assert!(current.same_recipe(before));
}
