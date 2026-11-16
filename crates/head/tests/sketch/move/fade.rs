// SPDX-License-Identifier: GPL-2.0-only
//! Drive the Precise Input Auto Fade control in the native Move window.
use super::{ApplicationWorld, move_pin, move_precise};
use cucumber::{given, then, when};

const BAR: &str = "Precise Input";
const FADE: &str = "Auto Fade";
const PIN: &str = "Pin Mini-Toolbar Position";

#[given("the mini-toolbar is visible with Pin checked and Auto Fade unchecked")]
fn pinned_toolbar(world: &mut ApplicationWorld) {
    move_precise::select_ellipse(world);
    move_pin::open_settings(world);
    assert!(!world.checked(PIN));
    world.click_text(PIN);
    move_pin::open_settings(world);
    assert!(world.checked(PIN));
    assert!(!world.checked(FADE));
}

#[when("Auto Fade is enabled and the pointer remains away")]
fn fade_away(world: &mut ApplicationWorld) {
    world.click_text(FADE);
    world.hover([350, 750]);
    world.pump(Vec::new());
}

#[then("the mode selector and lower toolbar row disappear while numeric fields remain")]
fn reduced_toolbar(world: &mut ApplicationWorld) {
    let text = world.window_text(BAR);
    assert!(
        !text.contains("Mode") && !text.contains("ResettoOrigin"),
        "{text:?}"
    );
    assert!(text.contains('X') && text.contains('Y'), "{text:?}");
}

#[when("the pointer hovers over the Y input")]
fn hover_y(world: &mut ApplicationWorld) {
    let position = world.precise_field_position("Y");
    world.hover(position);
}

#[then("the full toolbar reappears")]
fn expanded_toolbar(world: &mut ApplicationWorld) {
    let text = world.window_text(BAR);
    assert!(
        text.contains("Mode") && text.contains("ResettoOrigin"),
        "{text:?}"
    );
}

#[then("reopened Settings shows Auto Fade checked")]
fn checked_fade(world: &mut ApplicationWorld) {
    let button = world.run_in_window(BAR, "...").center();
    world.hover(super::diagnostics::pointer_pixels(button));
    move_pin::open_settings(world);
    assert!(world.checked(FADE));
}

#[when("Auto Fade is disabled and Settings is reopened")]
fn disable_fade(world: &mut ApplicationWorld) {
    world.click_text(FADE);
    move_pin::open_settings(world);
}

#[then("Auto Fade is unchecked")]
fn unchecked_fade(world: &mut ApplicationWorld) {
    assert!(!world.checked(FADE));
}

#[test]
fn auto_fade_preserves_editing_and_pinned_position() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    pinned_toolbar(&mut world);
    let pinned = move_pin::bar_position();
    fade_away(&mut world);
    reduced_toolbar(&mut world);
    world.save_capture("auto-fade-reduced");
    assert_eq!(move_pin::bar_position(), pinned);
    hover_y(&mut world);
    expanded_toolbar(&mut world);
    world.save_capture("auto-fade-restored");
    world.click_precise_field("Y");
    world.type_text("42");
    world.hover([350, 750]);
    expanded_toolbar(&mut world);
    world.type_text("7");
    assert!(world.window_text(BAR).contains("427"));
    assert_eq!(move_pin::bar_position(), pinned);
    world.close();
}
