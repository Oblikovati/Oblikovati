// SPDX-License-Identifier: GPL-2.0-only
//! Preserve accepted sketch text when a preset is removed from the session.
use super::*;
use text_fraction::{
    assert_stacked, open_blue_text_with_freedoms, replace_with_fraction, stack_selected_fraction,
};
use text_presets::{
    choose_preset_action, click_in_text_editor, open_preset_list, open_preset_settings,
};

const FIRST_TEXT_FREEDOMS: usize = 43;
const FINAL_FREEDOMS: usize = 47;

#[given("preset g exists and is selected")]
fn create_and_apply_preset(world: &mut ApplicationWorld) {
    open_blue_text_with_freedoms(world, FIRST_TEXT_FREEDOMS);
    replace_with_fraction(world);
    stack_selected_fraction(world);
    click_editor_button(world, "OK");
    text_edit::open_existing(world);
    text_fraction_preset::name_fraction_preset(world, b'g');
    click_editor_button(world, "Cancel");
    add_second_fraction(world);
    add_unrelated_text(world);
}

fn add_second_fraction(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, [970, 650]);
    open_preset_list(world);
    choose_preset_action(world, "g");
    world.run_in_window("Format Text", "g");
    click_editor_button(world, "OK");
    leave_repeated_placement(world);
}

fn add_unrelated_text(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, [1040, 570]);
    focus_content(world);
    for &byte in b"label" {
        x11_input::press_keysym(u32::from(byte));
        world.pump(Vec::new());
    }
    click_editor_button(world, "OK");
    leave_repeated_placement(world);
}

fn leave_repeated_placement(world: &mut ApplicationWorld) {
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
}

#[given("the reopened second fraction shows g")]
fn reopen_second_fraction_with_preset(world: &mut ApplicationWorld) {
    assert_accepted_text(world);
    let saved = world.edited_geometry();
    text_format_choices::reopen_text_sketch_from_file(world);
    assert_eq!(world.edited_geometry(), saved);
    world.text_creation_baseline = Some(saved);
    text_edit::open_accepted_text(world, 1);
    world.run_in_window("Format Text", "g");
    expect_stacked_preview(world);
}

#[when("the user opens preset settings and choose Delete Current")]
fn delete_current_preset(world: &mut ApplicationWorld) {
    open_preset_settings(world);
    choose_preset_action(world, "Delete Current");
}

#[then("No Preset replaces g without an additional application confirmation window")]
fn preset_is_deleted_immediately(world: &mut ApplicationWorld) {
    world.run_in_window("Format Text", "NoPreset");
    world.expect_no_text("Delete Current");
    assert!(
        !world
            .window_text("Format Text")
            .split_whitespace()
            .any(|run| run == "g")
    );
    assert!(world.run_is_enabled("Cancel"));
}

#[then("the fraction content remains in the editor")]
fn expect_stacked_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().content, "1/2");
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected an editable stacked fraction preview");
    };
    assert_stacked(&style);
}

#[then("g is absent and No Preset and Last Used remain")]
fn deleted_preset_is_absent_from_list(world: &mut ApplicationWorld) {
    let last = world.run("Last Used").list;
    let list = world.frame.runs.iter().filter(|run| run.list == last);
    assert_eq!(
        list.map(|run| run.text.as_str()).collect::<Vec<_>>(),
        ["NoPreset", "LastUsed"]
    );
    world.save_capture("preset-g-removed");
}

#[when("the user dismisses the list and cancel the unchanged editor")]
fn cancel_edit_after_removing_preset(world: &mut ApplicationWorld) {
    click_in_text_editor(world, "Content");
    click_editor_button(world, "Cancel");
}

#[then("both accepted fractions and the other text remain")]
fn accepted_geometry_is_unchanged(world: &mut ApplicationWorld) {
    assert_accepted_text(world);
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
    assert!(world.app().owner.sketch_preview().is_none());
}

#[then("Ready shows 47 dimensions needed")]
fn restored_sketch_remains_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(FINAL_FREEDOMS);
}

fn assert_accepted_text(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    let [first, second, label] = geometry.texts() else {
        panic!("expected two accepted fractions and the unrelated text label");
    };
    assert_stacked(&first.style);
    assert_stacked(&second.style);
    assert_ne!(first.anchor, second.anchor);
    assert_eq!(label.style.content, "label");
    restored_sketch_remains_ready(world);
}
