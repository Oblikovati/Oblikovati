// SPDX-License-Identifier: GPL-2.0-only
//! Preserve the source and presentation of a stacked fraction in a named preset.
use super::*;
use oblikovati_api::types::SketchTextStyle;
use text_format_choices::{choose_font, choose_format};
use text_fraction::{
    assert_stacked, existing_blue_text, replace_with_fraction, set_fraction_width,
    stack_selected_fraction,
};
use text_presets::{
    choose_preset_action, click_in_text_editor, confirm_preset_name, open_preset_list,
    open_preset_settings, replace_preset_name,
};

const SECOND_PLACEMENT: [i32; 2] = [970, 650];

#[given("the existing stacked fraction is open in Format Text")]
fn open_styled_fraction(world: &mut ApplicationWorld) {
    existing_blue_text(world);
    replace_with_fraction(world);
    stack_selected_fraction(world);
    choose_font(world, "Arial");
    choose_format(world, "Choose height", "6.10 mm");
    set_fraction_width(world, b"50");
    click_editor_button(world, "OK");
    let geometry = world.edited_geometry();
    assert_preset_style(&geometry.texts()[0].style);
    world.text_creation_baseline = Some(geometry);
    text_edit::open_existing(world);
    world.run_in_window("Format Text", "NoPreset");
}

#[when("the user creates and confirm inline preset name f")]
fn name_existing_fraction(world: &mut ApplicationWorld) {
    name_fraction_preset(world, b'f');
}

/// Name a fraction preset from the current draft, for example with `b'g'`.
pub(super) fn name_fraction_preset(world: &mut ApplicationWorld, name: u8) {
    click_in_text_editor(world, "+");
    replace_preset_name(world, name);
    confirm_preset_name(world);
    world.run_in_window("Format Text", &char::from(name).to_string());
}

#[when("the user cancels the unchanged existing-text editor")]
fn leave_first_fraction_unchanged(world: &mut ApplicationWorld) {
    click_editor_button(world, "Cancel");
    world.expect_ready();
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
}

#[when("the user starts a fresh Text placement")]
fn start_second_fraction(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, SECOND_PLACEMENT);
}

#[then("the fresh editor initially shows No Preset and empty content")]
fn fresh_fraction_is_empty(world: &mut ApplicationWorld) {
    world.run_in_window("Format Text", "NoPreset");
    assert!(world.state().sketch_text.unwrap().content.is_empty());
    assert!(!second_fraction_is_blue(world));
}

#[when("the user selects f from the preset list")]
fn choose_fraction_preset(world: &mut ApplicationWorld) {
    open_preset_list(world);
    choose_preset_action(world, "f");
}

#[then("the fraction content and blue formatting appear in the fresh placement")]
fn fraction_preview_is_restored(world: &mut ApplicationWorld) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the fraction preview from preset f");
    };
    assert_preset_style(&style);
    assert!(second_fraction_is_blue(world));
    assert_eq!(world.edited_geometry().texts().len(), 1);
    world.save_capture("fraction-preset-preview");
}

#[then("the controls show Arial, 6.10 mm, and width 50")]
fn fraction_fields_are_restored(world: &mut ApplicationWorld) {
    for text in ["Arial", "6.10", "50"] {
        world.run_in_window("Format Text", text);
    }
}

#[when("the user inspects and dismiss preset settings without changing them")]
fn inspect_fraction_preset_settings(world: &mut ApplicationWorld) {
    open_preset_settings(world);
    for action in ["Save Current", "Rename Current", "Delete Current"] {
        assert!(world.run_is_enabled(action), "expected {action}");
    }
    click_in_text_editor(world, "Width(%)");
    world.expect_no_text("Delete Current");
    world.run_in_window("Format Text", "f");
}

#[when("the user accepts the dialog and press Escape")]
fn accept_second_fraction(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
}

#[then("a second fraction remains with 37 dimensions needed")]
fn both_fractions_remain(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(37);
    let geometry = world.edited_geometry();
    let [first, second] = geometry.texts() else {
        panic!("expected two committed fraction text entities");
    };
    let original = world.text_creation_baseline.as_ref().unwrap();
    assert_eq!(first, &original.texts()[0]);
    assert_eq!(second.style, first.style);
    assert_ne!(second.id, first.id);
    assert_ne!(second.anchor, first.anchor);
    assert!(second_fraction_is_blue(world));
}

fn assert_preset_style(style: &SketchTextStyle) {
    assert_stacked(style);
    assert_eq!(style.family, "Arial");
    assert_eq!(style.height, 0.61);
    let width = oblikovati_model::sketch::uniform_sketch_text_width(
        &style.content,
        style.formatting_runs.as_deref().unwrap_or_default(),
    );
    assert_eq!(width, Some(50.));
}

fn second_fraction_is_blue(world: &mut ApplicationWorld) -> bool {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let stride = capture.extent[0] as usize * 4;
    let [x, y] = SECOND_PLACEMENT.map(|value| value as usize);
    capture
        .rgba
        .chunks_exact(stride)
        .skip(y - 36)
        .take(60)
        .any(|row| {
            row[(x - 4) * 4..(x + 38) * 4]
                .as_chunks::<4>()
                .0
                .iter()
                .any(|pixel| {
                    pixel[2].saturating_sub(pixel[0]) > 70 && pixel[2].saturating_sub(pixel[1]) > 35
                })
        })
}
