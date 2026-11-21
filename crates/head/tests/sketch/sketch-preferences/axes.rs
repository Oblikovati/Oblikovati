// SPDX-License-Identifier: GPL-2.0-only
//! Native scenario steps for the staged sketch-axis preference.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given(expr = "Axes is checked {word} Sketch preferences")]
fn axes_checked(world: &mut ApplicationWorld, _preposition: String) {
    open_checked_axes(world);
}
#[given("Axes is checked on Sketch preferences page")]
fn axes_checked_on_page(world: &mut ApplicationWorld) {
    open_checked_axes(world);
}
fn open_checked_axes(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.open_application_options();
    assert!(world.checked("Axes"));
}
#[when("the user unchecks Axes")]
fn uncheck_axes(world: &mut ApplicationWorld) {
    world.click_text("Axes");
    assert!(!world.checked("Axes"));
}
#[then("OK and enabled Cancel and Apply controls appear")]
fn pending_controls(world: &mut ApplicationWorld) {
    for label in ["OK", "Cancel", "Apply"] {
        assert_eq!(
            world.run_in_window("Application Options", label).alpha,
            u8::MAX
        );
    }
}
#[when("the user presses Escape without applying and reopens the Sketch page")]
fn escape_and_reopen(world: &mut ApplicationWorld) {
    world.close_application_options();
    world.open_application_options();
}
#[then("Axes is checked again")]
fn axes_restored(world: &mut ApplicationWorld) {
    assert!(world.checked("Axes"));
}
#[when("the user unchecks Axes and clicks Apply")]
fn apply_axes_off(world: &mut ApplicationWorld) {
    uncheck_axes(world);
    world.click_text("Apply");
}
#[then("Apply becomes disabled and Close is displayed")]
fn clean_controls(world: &mut ApplicationWorld) {
    assert_ne!(
        world.run_in_window("Application Options", "Apply").alpha,
        u8::MAX
    );
    world.run_in_window("Application Options", "Close");
}
#[when("the user closes Application Options")]
fn close_options(world: &mut ApplicationWorld) {
    world.click_text("Close");
}
#[then("the long sketch-axis line is absent while the global view triad remains")]
fn axes_hidden(world: &mut ApplicationWorld) {
    assert_axis(world, false);
    assert!(super::super::diagnostics::window_geometry("##viewport-axis-triad").is_some());
}
#[when("the user reopens Application Options")]
fn reopen_options(world: &mut ApplicationWorld) {
    world.open_application_options();
}
#[then("Axes remains unchecked")]
fn axes_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Axes"));
}
#[when("the user checks Axes and applies and closes")]
fn apply_axes_on(world: &mut ApplicationWorld) {
    world.click_text("Axes");
    world.click_text("Apply");
    world.click_text("Close");
}
#[then("the long sketch-axis line returns")]
fn axes_visible(world: &mut ApplicationWorld) {
    assert_axis(world, true);
}
fn assert_axis(world: &mut ApplicationWorld, expected: bool) {
    let point = world.sketch_pixel_of([2., 0.]);
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let found = (-2..=2).any(|dy| {
        (-2..=2).any(|dx| {
            let x = usize::try_from(point[0] + dx).unwrap();
            let y = usize::try_from(point[1] + dy).unwrap();
            let at = (y * capture.extent[0] as usize + x) * 4;
            let rgba = &capture.rgba[at..at + 3];
            rgba[0] > 110 && rgba[0] > rgba[1].saturating_add(35)
        })
    });
    assert_eq!(found, expected, "sketch X axis at {point:?}");
}
