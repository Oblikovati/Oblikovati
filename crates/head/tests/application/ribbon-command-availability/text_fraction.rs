// SPDX-License-Identifier: GPL-2.0-only
//! Check that stacking selected text creates a blue fraction in the actual viewport.
use super::*;
use oblikovati_api::types::{SketchTextFraction, SketchTextStyle};
use text_format_choices::{GLYPH_CROP_WIDTH, glyph_pixels};

const EXISTING_DOF: usize = 35;
const FRACTION: SketchTextFraction = SketchTextFraction { start: 0, end: 3 };

#[given("the existing blue Text is open in Format Text")]
pub(super) fn existing_blue_text(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(EXISTING_DOF - 2);
    activate_text(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, "DejaVu Sans");
    type_content(world, b'a');
    text_color::select_blue(world);
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
    world.expect_dimensions_needed(EXISTING_DOF);
    world.text_creation_baseline = Some(world.edited_geometry());
    text_edit::open_existing(world);
}

#[when("the user replaces its content with the characters 1, slash, and 2")]
pub(super) fn replace_with_fraction(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    for &byte in b"1/2" {
        x11_input::press_keysym(u32::from(byte));
        world.pump(Vec::new());
    }
    assert_eq!(world.state().sketch_text.unwrap().content, "1/2");
    assert!(!world.run_is_enabled("Stack"));
    world.observed_counts = vec![blue_rule(&glyph_pixels(world))];
}

#[when("the user selects the fraction and click the enabled stack control")]
pub(super) fn stack_selected_fraction(world: &mut ApplicationWorld) {
    x11_input::press_control_a();
    world.pump(Vec::new());
    assert!(world.run_is_enabled("Stack"));
    click_editor_button(world, "Stack");
}

#[then("the preview displays 1 above 2")]
fn fraction_preview(world: &mut ApplicationWorld) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the fraction preview");
    };
    assert_stacked(&style);
    let pixels = glyph_pixels(world);
    let separator = blue_rule(&pixels);
    assert!(
        separator > world.observed_counts[0],
        "blue fraction separator {separator}px: expected a longer horizontal stroke than the unstacked text"
    );
    world.save_capture("blue-stacked-fraction-preview");
}

#[when("the user accepts the dialog")]
fn accept_stacked_fraction(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
}

#[then("the stacked fraction remains and the status shows 35 dimensions needed")]
fn accepted_fraction(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(EXISTING_DOF);
    let accepted = world.edited_geometry();
    let original = world.text_creation_baseline.as_ref().unwrap();
    assert_eq!(accepted.texts()[0].id, original.texts()[0].id);
    assert_eq!(accepted.texts()[0].anchor, original.texts()[0].anchor);
    assert_stacked(&accepted.texts()[0].style);
    assert!(blue_rule(&glyph_pixels(world)) > world.observed_counts[0]);
}

pub(super) fn assert_stacked(style: &SketchTextStyle) {
    assert_eq!(style.content, "1/2");
    assert_eq!(style.stacked_fractions.as_deref(), Some(&[FRACTION][..]));
    let color = oblikovati_model::sketch::uniform_sketch_text_color(
        &style.content,
        style.formatting_runs.as_deref().unwrap_or_default(),
    )
    .flatten()
    .unwrap();
    assert_eq!([color.r, color.g, color.b], [0, 128, 255]);
}

fn blue_rule(pixels: &[u8]) -> usize {
    pixels
        .as_chunks::<{ GLYPH_CROP_WIDTH * 4 }>()
        .0
        .iter()
        .map(|row| longest_blue_run(row.as_chunks::<4>().0))
        .max()
        .unwrap_or_default()
}

fn longest_blue_run(row: &[[u8; 4]]) -> usize {
    row.split(|pixel| {
        pixel[2].saturating_sub(pixel[0]) <= 70 || pixel[2].saturating_sub(pixel[1]) <= 35
    })
    .map(<[_]>::len)
    .max()
    .unwrap_or_default()
}

const WIDTH_WARNING: &str = "Width is out of the expected range. Enter a positive width.";

#[given("the existing fraction editor shows width 50")]
fn existing_narrow_fraction(world: &mut ApplicationWorld) {
    existing_blue_text(world);
    replace_with_fraction(world);
    stack_selected_fraction(world);
    set_fraction_width(world, b"50");
    click_editor_button(world, "OK");
    world.text_creation_baseline = Some(world.edited_geometry());
    text_edit::open_existing(world);
    world.run_in_window("Format Text", "50");
    world.observed_counts = vec![blue_rule(&glyph_pixels(world))];
}

#[when("the user focuses width, select its value, enter 0, and press Tab")]
fn enter_invalid_width(world: &mut ApplicationWorld) {
    set_fraction_width(world, b"0");
}

pub(super) fn set_fraction_width(world: &mut ApplicationWorld, digits: &[u8]) {
    text_format_choices::focus_text_number(world, "Width(%)");
    x11_input::press_control_a();
    world.pump(Vec::new());
    text_format_choices::enter_text_number(world, digits);
}

#[then("a separate out-of-expected-range warning appears")]
fn width_warning_is_separate(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Text Width").is_some());
    world.expect_text(WIDTH_WARNING);
    world.run_in_window("Format Text", "Width(%)");
    assert_eq!(blue_rule(&glyph_pixels(world)), world.observed_counts[0]);
    world.save_capture("stacked-fraction-width-warning");
}

#[then("zero is selected for correction")]
fn zero_width_has_focus(world: &mut ApplicationWorld) {
    world.expect_no_text(WIDTH_WARNING);
    let zero = width_input_run(world, "0");
    assert!(world.text_run_is_selected(&zero));
}

#[when("the user enters 5 then 0 and press Tab")]
fn replace_invalid_width_without_reselecting(world: &mut ApplicationWorld) {
    // Inspect the raw input before Tab can normalize a value like 050.
    text_format_choices::type_digits(world, b"5");
    width_input_run(world, "5");
    text_format_choices::type_digits(world, b"0");
    width_input_run(world, "50");
    text_format_choices::tab_out_of_number(world);
}

#[then("width 50 is displayed without another warning")]
fn retained_width_without_warning(world: &mut ApplicationWorld) {
    width_input_run(world, "50");
    world.expect_no_text(WIDTH_WARNING);
    assert_eq!(blue_rule(&glyph_pixels(world)), world.observed_counts[0]);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the retained fraction preview");
    };
    assert_stacked(&style);
    click_editor_button(world, "Cancel");
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
}

fn width_input_run(world: &ApplicationWorld, text: &str) -> diagnostics::RenderedRun {
    let label = world.run_in_window("Format Text", "Width(%)");
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == text
                && run.minimum[0] < label.minimum[0]
                && (run.center()[1] - label.center()[1]).abs() <= 3.
        })
        .cloned()
        .unwrap_or_else(|| panic!("expected width field {text:?}"))
}
