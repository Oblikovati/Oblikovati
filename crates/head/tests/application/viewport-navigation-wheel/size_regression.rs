// SPDX-License-Identifier: GPL-2.0-only
//! The size fixtures must establish their preconditions from non-default preferences.
use super::*;

#[test]
fn mini_cancel_fixture_replaces_a_saved_large_preference() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    set_saved_size(&mut world, "Mini Wheel Size", "Large");
    mini_normal(&mut world);
    open_options(&mut world);
    expect_field_reads(&world, "Mini Wheel Size", "Normal");
    choose_row(&mut world, "Cancel");
    world.close();
}

#[test]
fn big_size_fixture_records_the_normal_wheel_after_large() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    set_saved_size(&mut world, "Big Wheel Size", "Large");
    let large = wheel_side();
    big_size::big_normal(&mut world);
    assert!(wheel_side() < large, "normal wheel must be smaller");
    assert_eq!(world.observed_counts[0], wheel_side() as usize);
    world.close();
}

#[test]
fn mini_size_fixture_records_the_normal_wheel_after_large() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    right_click_wheel(&mut world);
    choose_row(&mut world, "Mini View Object Wheel");
    set_saved_size(&mut world, "Mini Wheel Size", "Large");
    let large = wheel_side();
    preferences::mini_normal(&mut world);
    assert!(wheel_side() < large, "normal wheel must be smaller");
    assert_eq!(world.observed_counts[0], wheel_side() as usize);
    world.close();
}
