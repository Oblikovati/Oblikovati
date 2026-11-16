// SPDX-License-Identifier: GPL-2.0-only
//! Exercise Precise Input pin controls through the native window.
use super::{ApplicationWorld, diagnostics, move_precise};
use cucumber::{given, then, when};

const PIN: &str = "Pin Mini-Toolbar Position";
const BAR: &str = "Precise Input";

#[given("the precise-input mini-toolbar is visible at Move Base Point stage")]
fn precise_move(world: &mut ApplicationWorld) {
    move_precise::select_ellipse(world);
    assert!(diagnostics::window_geometry(BAR).is_some());
}

#[given("Pin Mini-Toolbar Position is unchecked")]
fn initially_unpinned(world: &mut ApplicationWorld) {
    open_settings(world);
    assert!(!world.checked(PIN));
}

#[when("Pin Mini-Toolbar Position is clicked and Settings is reopened")]
#[when("Pin is later clicked again and Settings is reopened")]
fn toggle_pin(world: &mut ApplicationWorld) {
    world.click_text(PIN);
    open_settings(world);
}

#[then("Pin Mini-Toolbar Position is checked")]
fn pin_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(PIN));
}

#[then("Pin Mini-Toolbar Position is unchecked")]
fn pin_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked(PIN));
}

fn open_settings(world: &mut ApplicationWorld) {
    let button = world.run_in_window(BAR, "...").center();
    world.click(diagnostics::pointer_pixels(button));
}

fn bar_position() -> [f32; 2] {
    diagnostics::window_geometry(BAR)
        .expect("expected precise input")
        .0
}

fn drag_bar(world: &mut ApplicationWorld) {
    let position = bar_position();
    let from = diagnostics::pointer_pixels([position[0] + 25., position[1] + 9.]);
    world.drag(from, [from[0] - 80, from[1] + 70]);
}

#[test]
fn position_pin_prevents_drag_until_it_is_cleared() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    move_precise::select_ellipse(&mut world);
    open_settings(&mut world);
    world.click_text(PIN);
    let pinned = bar_position();
    drag_bar(&mut world);
    assert_eq!(bar_position(), pinned, "pinned toolbar moved");
    open_settings(&mut world);
    world.click_text(PIN);
    drag_bar(&mut world);
    assert_ne!(bar_position(), pinned, "unpinned toolbar did not move");
    world.close();
}
