// SPDX-License-Identifier: GPL-2.0-only
//! The full-layout fixture works when the preceding layout differs.
use super::*;

#[test]
fn full_layout_fixture_replaces_a_mini_layout_and_keeps_wheel_closed() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    open_wheel(&mut world);
    right_click_wheel(&mut world);
    choose_row(&mut world, "Mini View Object Wheel");
    right_click_wheel(&mut world);
    expect_marked(&world, "Mini View Object Wheel");
    choose_row(&mut world, "Close Wheel");
    full_selected(&mut world);
    assert!(diagnostics::window_geometry(WHEEL).is_none());
    open_wheel(&mut world);
    right_click_wheel(&mut world);
    expect_marked(&world, "Full Navigation Wheel");
    world.close();
}
