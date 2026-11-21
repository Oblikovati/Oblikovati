// SPDX-License-Identifier: GPL-2.0-only
//! Wheel preferences that change how it shows, and bringing it back from the view's menu.
use super::*;

const SHOW_TOOL_TIPS: &str = "Show Tool Tips";
const REPEAT: &str = "Repeat Navigation Wheel";

// The mini wheel's size
#[given("the Mini View Object Wheel is open in the MCAD Software")]
fn mini_open(world: &mut ApplicationWorld) {
    open_wheel_on_block(world);
    right_click_wheel(world);
    choose_row(world, "Mini View Object Wheel");
}
#[given("the mini wheel size preference is Normal")]
pub(super) fn mini_normal(world: &mut ApplicationWorld) {
    set_saved_size(world, "Mini Wheel Size", "Normal");
    remember_side(world);
}
fn set_mini_size(world: &mut ApplicationWorld, size: &str) {
    open_list(world, "Mini Wheel Size");
    world.click_text(size);
    settle(world);
}
#[when("I open wheel options and choose Extra Large for mini wheel size")]
#[when("the user opens wheel options and chooses Extra Large for mini wheel size")]
fn choose_extra_large(world: &mut ApplicationWorld) {
    open_options(world);
    set_mini_size(world, "Extra Large");
}
#[then("the mini wheel appears larger than before")]
fn mini_larger(world: &mut ApplicationWorld) {
    let side = wheel_side();
    assert!(side as usize > world.observed_counts[0], "side {side}");
}
#[when("I choose Normal for mini wheel size and click OK")]
#[when("the user chooses Normal for mini wheel size and clicks OK")]
fn mini_back_to_normal(world: &mut ApplicationWorld) {
    set_mini_size(world, "Normal");
    world.click_text("OK");
    settle(world);
}
#[then("the mini wheel returns to its earlier apparent size")]
fn mini_earlier(world: &mut ApplicationWorld) {
    assert_eq!(wheel_side() as usize, world.observed_counts[0], "side");
}
#[when("I choose Full Navigation Wheel from the wheel menu")]
#[when("the user chooses Full Navigation Wheel from the wheel menu")]
fn menu_full(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Full Navigation Wheel");
}
// Tool tips
#[given("Show Tool Tips is enabled in wheel options")]
fn tips_enabled(world: &mut ApplicationWorld) {
    open_options(world);
    assert!(
        world.checked(SHOW_TOOL_TIPS),
        "expected {SHOW_TOOL_TIPS} checked"
    );
    world.click_text("Cancel");
    settle(world);
}
#[when("I open wheel options and clear Show Tool Tips")]
#[when("the user opens wheel options and clear Show Tool Tips")]
fn clear_tips(world: &mut ApplicationWorld) {
    open_options(world);
    world.click_text(SHOW_TOOL_TIPS);
    settle(world);
}
#[then("Show Tool Tips is unchecked")]
fn tips_unchecked(world: &mut ApplicationWorld) {
    assert!(
        !world.checked(SHOW_TOOL_TIPS),
        "expected {SHOW_TOOL_TIPS} clear"
    );
}
#[when("I select Show Tool Tips and click OK")]
#[when("the user selects Show Tool Tips and clicks OK")]
fn select_tips(world: &mut ApplicationWorld) {
    world.click_text(SHOW_TOOL_TIPS);
    settle(world);
    world.click_text("OK");
    settle(world);
    open_options(world);
    assert!(world.checked(SHOW_TOOL_TIPS), "expected tool tips restored");
    choose_row(world, "Cancel");
}
// Repeating the command
#[given("the full navigation wheel was the last command used")]
fn wheel_used(world: &mut ApplicationWorld) {
    open_full_wheel(world);
    wheel_geometry();
}
#[given("the wheel is closed")]
fn wheel_closed(world: &mut ApplicationWorld) {
    world.escape();
    settle(world);
    assert!(
        diagnostics::window_geometry(WHEEL).is_none(),
        "expected no wheel"
    );
}
#[when("I right-click the model viewport")]
#[when("the user right-clicks the model viewport")]
fn right_click_view(world: &mut ApplicationWorld) {
    world.right_click(PLACED_AT);
    settle(world);
}
#[then("the context menu offers to repeat the navigation wheel command")]
fn offers_repeat(world: &mut ApplicationWorld) {
    world.expect_text(REPEAT);
}
#[when("I choose the repeat navigation wheel command")]
#[when("the user chooses the repeat navigation wheel command")]
fn choose_repeat(world: &mut ApplicationWorld) {
    world.click_text(REPEAT);
    settle(world);
}
#[then("the full navigation wheel appears over the model")]
fn full_again(_: &mut ApplicationWorld) {
    assert!(wheel_side() >= FULL_WHEEL_SIDE, "side {}", wheel_side());
}

#[cfg(target_os = "linux")]
#[test]
fn native_viewport_menu_reopens_the_previous_navigation_wheel() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    let before = camera(&mut world);
    let document = world.state().document;
    wheel_closed(&mut world);
    super::super::x11_input::right_click(PLACED_AT);
    world.pump(Vec::new());
    world.expect_text(REPEAT);
    let position = super::super::rendered_label_center(REPEAT);
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    settle(&mut world);
    full_again(&mut world);
    assert_eq!(world.state().document, document);
    assert_eq!(camera(&mut world), before);
    world.save_capture("repeat-wheel-native-4058");
    world.escape();
    world.close();
}
