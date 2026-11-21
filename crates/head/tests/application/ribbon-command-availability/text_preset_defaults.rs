// SPDX-License-Identifier: GPL-2.0-only
//! Exercise named and last-accepted defaults through the native Format Text editor.
use super::*;
use text_fraction::{
    assert_stacked, existing_blue_text, replace_with_fraction, set_fraction_width,
    stack_selected_fraction,
};
use text_presets::{choose_preset_action, open_preset_list, open_preset_settings};

#[test]
fn sketch_text_new_placements_distinguish_named_and_last_used_defaults() {
    let mut world = ApplicationWorld::new();
    reopen_saved_g(&mut world);
    set_named_default_and_cancel(&mut world);
    start_named_default_placement(&mut world);
    named_default_uses_saved_fraction(&mut world);
    start_last_used_placement(&mut world);
    last_used_restores_modified_fraction(&mut world);
    last_used_is_checked_in_settings(&mut world);
    restore_no_preset(&mut world);
    let formats = world.state().sketch_text_font_families.unwrap();
    let family = if formats
        .iter()
        .any(|name| name == text_cancel_defaults::TAHOMA)
    {
        text_cancel_defaults::TAHOMA
    } else {
        ""
    };
    expect_blank_defaults(&mut world, family);
    named_preset_remains_available(&mut world);
    world.close();
}

fn save_g_with_a_different_last_accepted_width(world: &mut ApplicationWorld) {
    existing_blue_text(world);
    replace_with_fraction(world);
    stack_selected_fraction(world);
    text_fraction_preset::name_fraction_preset(world, b'g');
    set_fraction_width(world, b"50");
    expect_fraction_default(world, "*g", "50");
    click_editor_button(world, "OK");
    world.expect_ready();
}

/// Set the default and close the draft, for example with `"Use Current Preset"`.
pub(super) fn set_new_text_default(world: &mut ApplicationWorld, label: &str) {
    open_preset_settings(world);
    choose_preset_action(world, label);
    world.expect_no_text("New Text Default");
    click_editor_button(world, "Cancel");
    assert!(
        world.state().sketch_text.is_none(),
        "closed editor after setting {label}"
    );
}

fn place_with_default(world: &mut ApplicationWorld, anchor: [i32; 2]) {
    x11_input::click(anchor);
    world.pump(Vec::new());
    world.expect_text("Format Text");
}

fn expect_fraction_default(world: &mut ApplicationWorld, caption: &str, width: &str) {
    world.run_in_window("Format Text", caption);
    world.run_in_window("Format Text", width);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected a stacked fraction preview from the new Text default");
    };
    assert_stacked(&style);
    assert_eq!(
        oblikovati_model::sketch::uniform_sketch_text_width(
            &style.content,
            style.formatting_runs.as_deref().unwrap_or_default(),
        ),
        width.parse().ok()
    );
}

#[given("the saved preset g")]
fn reopen_saved_g(world: &mut ApplicationWorld) {
    save_g_with_a_different_last_accepted_width(world);
    world.text_creation_baseline = Some(world.edited_geometry());
    text_edit::open_existing(world);
    world.run_in_window("Format Text", "*g");
}

#[when(
    "the user selects Use Current Preset as the new Text default and cancel the existing-text editor"
)]
fn set_named_default_and_cancel(world: &mut ApplicationWorld) {
    set_new_text_default(world, "Use Current Preset");
}

#[when("the user starts another placement")]
fn start_named_default_placement(world: &mut ApplicationWorld) {
    activate_text(world);
    place_with_default(world, [970, 650]);
}

#[then("g and its fraction content appear automatically")]
fn named_default_uses_saved_fraction(world: &mut ApplicationWorld) {
    expect_fraction_default(world, "g", "100");
}

#[when("the user selects Last Used, cancel, and place again")]
fn start_last_used_placement(world: &mut ApplicationWorld) {
    set_new_text_default(world, "Last Used");
    place_with_default(world, [980, 660]);
}

#[then("the fraction preview returns with the clock-marked *g selector")]
fn last_used_restores_modified_fraction(world: &mut ApplicationWorld) {
    expect_fraction_default(world, "*g", "50");
    text_linked_journey::capture_when_requested(world, "last-used-clock");
}

#[then("Last Used is checked in settings")]
fn last_used_is_checked_in_settings(world: &mut ApplicationWorld) {
    open_preset_settings(world);
    assert!(world.checked("Last Used"));
}

#[when("the user restores No Preset, cancel, and place again")]
fn restore_no_preset(world: &mut ApplicationWorld) {
    choose_preset_action(world, "No Preset");
    world.expect_no_text("New Text Default");
    click_editor_button(world, "Cancel");
    place_with_default(world, [990, 670]);
}

#[then("the editor is empty with Tahoma 3.05 mm, purple and width 100")]
fn no_preset_retains_standard_format(world: &mut ApplicationWorld) {
    expect_blank_defaults(world, text_cancel_defaults::TAHOMA);
    text_linked_journey::capture_when_requested(world, "no-preset-tahoma-and-purple");
}

fn expect_blank_defaults(world: &mut ApplicationWorld, family: &str) {
    let draft = world.state().sketch_text.unwrap();
    assert!(draft.content.is_empty());
    assert_eq!(draft.family, family);
    assert!(draft.formatting.runs.is_none());
    let shown = if family.is_empty() {
        "Systemdefault"
    } else {
        family
    };
    for label in ["NoPreset", shown, "3.05", "100"] {
        world.run_in_window("Format Text", label);
    }
    let [r, g, b, _] = world.pixel_left_of("Color");
    assert!(
        b > r && r > g,
        "expected purple swatch, got RGB {r}, {g}, {b}"
    );
}

#[then("the saved preset g remains listed")]
fn named_preset_remains_available(world: &mut ApplicationWorld) {
    open_preset_list(world);
    assert_eq!(world.run("g").list, world.run("Last Used").list);
    assert_eq!(Some(world.edited_geometry()), world.text_creation_baseline);
}
