// SPDX-License-Identifier: GPL-2.0-only
//! Moving the floating Customize window without replacing the active Part.
use super::*;

#[given("ribbon customization is open without pending edits")]
fn open_clean_dialog(world: &mut ApplicationWorld) {
    world.previous_document = world.state().document;
    empty_tools_customization(world);
    for label in ["Cancel", "Apply"] {
        assert!(world.run_in_window("Customize", label).alpha < u8::MAX);
    }
}

#[when("the user drags the dialog title bar to another visible position")]
fn move_dialog(world: &mut ApplicationWorld) {
    let (original, _) = diagnostics::window_geometry("Customize").unwrap();
    let from = diagnostics::pointer_pixels([original[0] + 95., original[1] + 10.]);
    let to = [from[0] - 125, from[1] + 72];
    world.drag(from, to);
    let moved = diagnostics::window_geometry("Customize").unwrap().0;
    assert!(
        moved[0] < original[0] - 80. && moved[1] > original[1] + 40.,
        "Customize stayed near its original position {original:?}: {moved:?}"
    );
    world.moved_customize_origin = Some(moved);
}

#[then("the dialog moves to the new position")]
#[then("the dialog reopens at the moved position")]
fn moved_position(world: &mut ApplicationWorld) {
    assert_eq!(
        diagnostics::window_geometry("Customize").unwrap().0,
        world.moved_customize_origin.unwrap()
    );
}

#[then("its Ribbon tab, command lists and target selector remain visible")]
#[then("its Ribbon tab and Part Tools target are displayed")]
fn ribbon_lists_remain(world: &mut ApplicationWorld) {
    for label in ["Ribbon", "Availablecommands", "Custompanel", "Part|Tools"] {
        world.run_in_window("Customize", label);
    }
    assert!(world.previous_document.is_some());
    assert_eq!(world.state().document, world.previous_document);
}

#[when("the user closes customization and reopens it from Tools in the same session")]
fn reopen_dialog(world: &mut ApplicationWorld) {
    close_customization_button(world);
    assert!(diagnostics::window_geometry("Customize").is_none());
    open_tools_customization(world);
}
