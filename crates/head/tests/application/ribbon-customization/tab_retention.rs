// SPDX-License-Identifier: GPL-2.0-only
//! Session tab retention through the Tools customization entry point.
use super::*;
#[path = "marking_environment.rs"]
mod marking_environment;
#[path = "marking_part.rs"]
mod marking_part;
#[path = "marking_presentation.rs"]
mod marking_presentation;
pub(in super::super) use marking_environment::choose_sketch_environment;

#[given("the customization dialog is open on Ribbon for an empty Part Tools custom panel")]
fn empty_ribbon(world: &mut ApplicationWorld) {
    empty_tools_customization(world);
    ribbon_tab_open(world);
}

#[when("the user selects the Marking Menu customization tab")]
#[when("the user selects the \"Marking Menu\" tab")]
fn select_marking_menu(world: &mut ApplicationWorld) {
    click_in_customize(world, "MarkingMenu");
}

#[then("the marking menu customization controls are displayed")]
fn marking_controls(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "RestoreDefault");
}

#[then("Apply and cancels remain available")]
fn pending_edit_controls(world: &mut ApplicationWorld) {
    customization_edits_remain_available(world);
}

#[when("the user returns to the Ribbon customization tab")]
fn select_ribbon(world: &mut ApplicationWorld) {
    return_to_ribbon_customization(world);
}

#[when("the user cancels the customization dialog")]
fn cancel_staged_dialog(world: &mut ApplicationWorld) {
    cancel_customization(world);
}

#[given("the customization dialog is open on the Marking Menu tab with no pending edits")]
fn marking_menu_open(world: &mut ApplicationWorld) {
    sketch_is_active(world);
    world.record_initial_sketch();
    open_tools_customization(world);
    click_in_customize(world, "MarkingMenu");
    no_pending(world);
}

#[given("the customization dialog is open on the Marking Menu tab in the MCAD Software")]
#[when("the user reopens customization on the Marking Menu tab")]
fn open_marking_menu_tab(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.click_text("Customize");
    world.pump(Vec::new());
    click_in_customize(world, "MarkingMenu");
    world.run_in_window("Customize", "RestoreDefault");
}

#[given("\"Use Classic Context Menu\" is unchecked")]
#[then("\"Use Classic Context Menu\" is unchecked")]
fn classic_unchecked(world: &mut ApplicationWorld) {
    let checkbox = world.run_in_window("Customize", "UseClassicContextMenu");
    assert!(!world.checked_run(checkbox));
    assert!(!world.state().sketch_classic_menu);
}

#[when("the user checks \"Use Classic Context Menu\"")]
fn stage_classic_menu(world: &mut ApplicationWorld) {
    click_in_customize(world, "UseClassicContextMenu");
}

#[then("the option is checked")]
fn staged_classic_checked(world: &mut ApplicationWorld) {
    let checkbox = world.run_in_window("Customize", "UseClassicContextMenu");
    assert!(world.checked_run(checkbox));
    assert!(!world.state().sketch_classic_menu);
}

#[then("\"OK\", \"Cancel\" and \"Apply\" are available")]
fn marking_actions_enabled(world: &mut ApplicationWorld) {
    for label in ["OK", "Cancel", "Apply"] {
        assert_eq!(world.run_in_window("Customize", label).alpha, u8::MAX);
    }
}

#[test]
fn applying_classic_menu_persists_through_native_restart() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_marking_menu_tab(&mut world);
    classic_unchecked(&mut world);
    stage_classic_menu(&mut world);
    staged_classic_checked(&mut world);
    click_in_customize(&mut world, "Apply");
    assert!(world.state().sketch_classic_menu);
    world.reopen_application();
    assert!(world.state().sketch_classic_menu);
    world.add_part();
    assert!(world.state().sketch_classic_menu);
    world.close();
}
#[given("a part with an active sketch is open in The MCAD Software")]
#[given("a part with an active sketch is open in the MCAD Software")]
#[given("a part with an active sketch is displayed in the MCAD Software")]
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
    marking_visible(world);
}
#[when("the user selects the \"Keyboard\" tab")]
fn select_keyboard(world: &mut ApplicationWorld) {
    click_in_customize(world, "Keyboard");
}
#[when("the user closes the customization dialog without making changes")]
#[when("the user closes customization without changes")]
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
    world.run_in_window("Customize", "2DSketch");
}
#[then("Environment remains \"2D Sketch\"")]
fn environment_unchanged(world: &mut ApplicationWorld) {
    marking_visible(world);
}
#[then("Sub-Environment remains \"None Selected\"")]
#[then("Sub-Environment is \"None Selected\"")]
#[given("Sub-Environment is \"None Selected\"")]
fn sub_environment_unchanged(world: &mut ApplicationWorld) {
    assert!(world.window_text("Customize").contains("NoneSelected"));
}
#[then("no keyboard or ribbon changes are pending")]
#[then("no customization change is pending")]
#[given("no customization change is pending")]
fn no_pending(world: &mut ApplicationWorld) {
    for label in ["Cancel", "Apply"] {
        assert!(world.run_in_window("Customize", label).alpha < u8::MAX);
    }
}
#[then("the active sketch remains unchanged")]
#[then("the editable sketch remains active and unchanged")]
#[then("the part and editable sketch remain active and unchanged")]
#[then("sketch editing remains active and unchanged")]
fn sketch_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert!(world.state().document.is_some());
    let current = world.edited_geometry();
    let before = &world
        .observed_sketch
        .as_ref()
        .expect("expected the initial sketch")
        .geometry;
    assert!(current.same_recipe(before));
}
