// SPDX-License-Identifier: GPL-2.0-only
//! Big wheel size preference scenarios.
use super::*;

fn set_big_size(world: &mut ApplicationWorld, size: &str) {
    open_list(world, "Big Wheel Size");
    choose_row(world, size);
}
#[given("the full navigation wheel is visible in the MCAD Software")]
fn wheel_visible_given(world: &mut ApplicationWorld) {
    open_wheel_on_block(world);
}
#[given("its big wheel size preference is Normal")]
pub(super) fn big_normal(world: &mut ApplicationWorld) {
    set_saved_size(world, "Big Wheel Size", "Normal");
    remember_side(world);
}
#[when("I open the wheel options and choose Large for big wheel size")]
#[when("the user opens the wheel options and chooses Large for big wheel size")]
fn choose_large(world: &mut ApplicationWorld) {
    open_options(world);
    set_big_size(world, "Large");
}
#[then("the options dialog closes")]
fn options_closed(world: &mut ApplicationWorld) {
    world.expect_no_text(OPTIONS_TITLE);
}
#[then("the full wheel appears larger than before")]
fn wheel_larger(world: &mut ApplicationWorld) {
    assert!(
        wheel_side() as usize > world.observed_counts[0],
        "side {}",
        wheel_side()
    );
}
#[when("I reopen wheel options")]
#[when("the user reopens wheel options")]
fn reopen_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the big wheel size field displays Large")]
fn field_large(world: &mut ApplicationWorld) {
    expect_field_reads(world, "Big Wheel Size", "Large");
}
#[when("I choose Normal for big wheel size and click OK")]
#[when("the user chooses Normal for big wheel size and clicks OK")]
fn choose_normal(world: &mut ApplicationWorld) {
    set_big_size(world, "Normal");
    choose_row(world, "OK");
}
#[then("the full wheel returns to its previous apparent size")]
fn wheel_restored(world: &mut ApplicationWorld) {
    assert_eq!(wheel_side() as usize, world.observed_counts[0], "side");
}
#[then("the wheel closes")]
fn wheel_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("Rewind");
}
