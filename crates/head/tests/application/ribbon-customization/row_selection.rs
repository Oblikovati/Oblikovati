// SPDX-License-Identifier: GPL-2.0-only
//! Selection gestures for the ribbon customization lists.
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user presses Enter while the list has focus")]
fn press_enter_in_source_list(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
}

#[when("the user presses Tab and then Space")]
fn tab_to_transfer_and_press_space(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
    world.key(ViewportPhysicalKey::Space, false);
}

#[given("All Commands lists \"2D Sketches\" immediately before \"3D Annotations\"")]
fn available_rows_in_order(world: &mut ApplicationWorld) {
    let first = world.run_in_window("Customize", "2DSketches");
    let second = world.run_in_window("Customize", "3DAnnotations");
    assert!(first.center()[1] < second.center()[1]);
}
#[when("the user selects \"2D Sketches\" in the available commands list")]
#[when("the user selects \"2D Sketches\" in the available command list")]
#[when("the user selects \"2D Sketches\" in the custom panel list")]
#[when("the user selects \"2D Sketches\" in the target list")]
#[when("I click the 2D Sketches name in the staged target list")]
#[when("the user clicks the 2D Sketches name in the staged target list")]
fn select_sketches_row(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}

#[when("I press Ctrl+A while the target list has focus")]
#[when("the user presses Ctrl+A while the target list has focus")]
fn select_all_staged(world: &mut ApplicationWorld) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
}

#[when("I cancel ribbon customization")]
#[when("the user cancels ribbon customization")]
fn cancel_from_keyboard_scenario(world: &mut ApplicationWorld) {
    cancel_customization(world);
}
#[when("the user presses Shift+Down")]
pub(super) fn shift_down(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{
        ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey,
    };
    let key = ViewportPhysicalKey::DownArrow;
    let event = |pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![
        event(
            true,
            ViewportModifiers {
                shift: true,
                ..Default::default()
            },
        ),
        event(false, Default::default()),
    ]);
}
