// SPDX-License-Identifier: GPL-2.0-only
//! Navigation wheel opacity preference scenarios.
use super::*;

// Wheel opacity
fn set_opacity(world: &mut ApplicationWorld, percent: &str) {
    open_list(world, "Wheel Opacity");
    choose_row(world, percent);
}

#[test]
fn wheel_options_title_close_discards_the_pending_opacity() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    open_options(&mut world);
    set_opacity(&mut world, "90%");
    let (position, size) = diagnostics::window_geometry(OPTIONS_TITLE).unwrap();
    let close = diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    assert!(diagnostics::window_geometry(OPTIONS_TITLE).is_none());
    open_options(&mut world);
    expect_field_reads(&world, "Wheel Opacity", "50%");
    choose_row(&mut world, "Cancel");
    world.close();
}
#[given("the full navigation wheel is open in the MCAD Software")]
fn wheel_open_plain(world: &mut ApplicationWorld) {
    open_wheel_on_block(world);
}
#[given("its wheel opacity preference is 50%")]
fn opacity_half(world: &mut ApplicationWorld) {
    open_options(world);
    expect_field_reads(world, "Wheel Opacity", "50%");
    choose_row(world, "Cancel");
}
#[when("I open the wheel options and choose 90% wheel opacity")]
#[when("the user opens the wheel options and chooses 90% wheel opacity")]
fn choose_ninety(world: &mut ApplicationWorld) {
    open_options(world);
    set_opacity(world, "90%");
}
#[then("the options dialog closes and the wheel remains visible")]
fn closed_wheel_visible(world: &mut ApplicationWorld) {
    world.expect_no_text(OPTIONS_TITLE);
    wheel_geometry();
}
#[when("I reopen the wheel options")]
#[when("the user reopens the wheel options")]
fn reopen_wheel_options(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the wheel opacity field displays 90%")]
fn field_ninety(world: &mut ApplicationWorld) {
    expect_field_reads(world, "Wheel Opacity", "90%");
}
#[when("I choose 50% wheel opacity and click OK")]
#[when("the user chooses 50% wheel opacity and clicks OK")]
fn choose_half(world: &mut ApplicationWorld) {
    set_opacity(world, "50%");
    choose_row(world, "OK");
}
