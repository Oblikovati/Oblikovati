// SPDX-License-Identifier: GPL-2.0-only
//! Selection gestures for the ribbon customization lists.
use super::*;

#[given("All Commands lists \"2D Sketches\" immediately before \"3D Annotations\"")]
fn available_rows_in_order(world: &mut ApplicationWorld) {
    let first = world.run_in_window("Customize", "2DSketches");
    let second = world.run_in_window("Customize", "3DAnnotations");
    assert!(first.center()[1] < second.center()[1]);
}
#[when("the user selects \"2D Sketches\" in the available commands list")]
#[when("the user selects \"2D Sketches\" in the custom panel list")]
fn select_sketches_row(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
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
