// SPDX-License-Identifier: GPL-2.0-only
//! Drive native font, height and alignment controls and inspect their glyph output.
use super::*;
use oblikovati_api::types::{
    SketchTextStyle, TEXT_ALIGN_CENTER, TEXT_ALIGN_MIDDLE, TEXT_LINE_SPACING_DOUBLE,
};

const MONOSPACE: &str = "DejaVu Sans Mono";
pub(super) const GLYPH_CROP_WIDTH: usize = 90;

#[test]
fn sketch_text_format_choices_preview_and_preserve_the_original_on_cancel() {
    let mut world = ApplicationWorld::new();
    text_edit::committed_b(&mut world);
    let original = world.edited_geometry();
    let original_pixels = glyph_pixels(&mut world);
    text_edit::open_existing(&mut world);
    choose_formats(&mut world);
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-text-format-preview");
    }
    click_editor_button(&mut world, "Cancel");
    assert_eq!(world.edited_geometry(), original);
    assert_eq!(glyph_pixels(&mut world), original_pixels);
    text_edit::open_existing(&mut world);
    choose_formats(&mut world);
    click_editor_button(&mut world, "OK");
    check_committed_format_history(&mut world, original);
    check_formatted_text_reopens(&mut world);
    world.close();
}

fn choose_formats(world: &mut ApplicationWorld) {
    let before = glyph_pixels(world);
    choose_font(world, MONOSPACE);
    let font_pixels = glyph_pixels(world);
    assert_ne!(font_pixels, before, "font changed the glyph pixels");
    choose_format(world, "Choose height", "6.10 mm");
    let taller = glyph_pixels(world);
    assert_ne!(taller, font_pixels, "height changed the glyph pixels");
    choose_format(world, "Left", "Center");
    choose_format(world, "Baseline", "Middle");
    assert_ne!(
        glyph_pixels(world),
        taller,
        "alignment changed the glyph pixels"
    );
    choose_format(world, "Single", "Double");
    assert_preview_style(world);
}

pub(super) fn choose_font(world: &mut ApplicationWorld, family: &str) {
    let families = world.state().sketch_text_font_families.unwrap();
    assert!(
        families.iter().any(|installed| installed == family),
        "Sketch Text fixture: expected installed font family {family:?}"
    );
    click_format_row(world, "Choose font");
    let label = world.run("Filter fonts").clone();
    x11_input::click(diagnostics::pointer_pixels([
        label.minimum[0] - 70.,
        label.center()[1],
    ]));
    world.pump(Vec::new());
    for letter in family.split_ascii_whitespace().next().unwrap().bytes() {
        x11_input::press_keysym(u32::from(letter.to_ascii_lowercase()));
    }
    world.pump(Vec::new());
    click_format_row(world, family);
    assert_eq!(world.state().sketch_text.unwrap().family, family);
}

pub(super) fn choose_format(world: &mut ApplicationWorld, caption: &str, selection: &str) {
    click_format_row(world, caption);
    click_format_row(world, selection);
}

fn click_format_row(world: &mut ApplicationWorld, label: &str) {
    let center = world.run(label).center();
    x11_input::click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

pub(super) fn glyph_pixels(world: &mut ApplicationWorld) -> Vec<u8> {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    glyph_region(&capture)
}

pub(super) fn glyph_region(capture: &oblikovati_viewport::ViewportCapture) -> Vec<u8> {
    let left = 860 * 4;
    let right = left + GLYPH_CROP_WIDTH * 4;
    let stride = capture.extent[0] as usize * 4;
    capture
        .rgba
        .chunks_exact(stride)
        .skip(545)
        .take(72)
        .flat_map(|row| row[left..right].iter().copied())
        .collect()
}

fn assert_preview_style(world: &mut ApplicationWorld) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the formatted preview");
    };
    assert_format(&style);
}

fn assert_format(style: &SketchTextStyle) {
    assert_eq!(style.family, MONOSPACE);
    assert_eq!(style.height, 6.10 / 10.);
    assert_eq!(style.h_align, TEXT_ALIGN_CENTER);
    assert_eq!(style.v_align, TEXT_ALIGN_MIDDLE);
    assert_eq!(
        style.line_spacing.as_ref().unwrap().mode,
        TEXT_LINE_SPACING_DOUBLE
    );
}

fn check_committed_format_history(
    world: &mut ApplicationWorld,
    original: oblikovati_model::sketch::Sketch2D,
) {
    let committed = world.edited_geometry();
    let text = &committed.texts()[0];
    assert_format(&text.style);
    assert_eq!(text.id, original.texts()[0].id);
    assert_eq!(
        committed.point(text.anchor).unwrap(),
        original.point(text.anchor).unwrap()
    );
    text_edit::undo_edit(world);
    assert_eq!(world.edited_geometry(), original);
    text_edit::redo_edit(world);
    assert_eq!(world.edited_geometry(), committed);
    text_edit::open_existing(world);
    assert_preview_style(world);
}

fn check_formatted_text_reopens(world: &mut ApplicationWorld) {
    let committed = world.edited_geometry();
    click_editor_button(world, "Cancel");
    reopen_text_sketch_from_file(world);
    assert_eq!(world.edited_geometry(), committed);
    text_edit::open_existing(world);
    assert_preview_style(world);
}

pub(super) fn reopen_text_sketch_from_file(world: &mut ApplicationWorld) {
    world.finish_sketch();
    x11_input::press_control_letter(b's');
    world.pump(Vec::new());
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    let row = world.run("Sketch1").center();
    x11_input::right_click(diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    click_format_row(world, "Edit Sketch");
}

#[test]
fn sketch_text_rejects_zero_height_without_losing_the_last_valid_preview() {
    let mut world = ApplicationWorld::new();
    let family = open_recorded_height_fraction(&mut world);
    let original = world.edited_geometry();
    let pixels = glyph_pixels(&mut world);
    reject_zero_in_fraction_editor(&mut world);
    assert_eq!(glyph_pixels(&mut world), pixels);
    accept_typed_six_as_preview(&mut world, family);
    click_editor_button(&mut world, "Cancel");
    world.expect_ready();
    world.expect_dimensions_needed(47);
    assert_eq!(world.edited_geometry(), original);
    text_linked_journey::capture_when_requested(&mut world, "height-fraction-cancel-at-ready");
    world.close();
}

fn open_recorded_height_fraction(world: &mut ApplicationWorld) -> &'static str {
    text_fraction::open_blue_text_with_freedoms(world, 47);
    text_fraction::replace_with_fraction(world);
    text_fraction::stack_selected_fraction(world);
    let catalogue = world.state().sketch_text_font_families.unwrap();
    let family = if catalogue.iter().any(|family| family == "Arial") {
        "Arial"
    } else {
        "DejaVu Sans"
    };
    choose_font(world, family);
    choose_format(world, "Choose height", "6.10 mm");
    text_fraction::set_fraction_width(world, b"50");
    click_editor_button(world, "OK");
    let authored = world.edited_geometry();
    reopen_text_sketch_from_file(world);
    assert_eq!(world.edited_geometry(), authored);
    capture_height_validation_fixture(world);
    text_edit::open_existing(world);
    world.run_in_window("Format Text", "6.10");
    family
}

fn capture_height_validation_fixture(world: &ApplicationWorld) {
    let Some(reports) = std::env::var_os("OBK_BDD_REPORTS") else {
        return;
    };
    let saved = world.saved_path.as_ref().unwrap();
    std::fs::copy(saved, std::path::PathBuf::from(reports).join("Part1.opd")).unwrap();
}

fn reject_zero_in_fraction_editor(world: &mut ApplicationWorld) {
    let valid_preview = world.app().owner.sketch_preview().unwrap();
    focus_text_number(world, "Height(mm)");
    x11_input::press_control_a();
    world.pump(Vec::new());
    enter_text_number(world, b"0");
    world.expect_text("Enter a size greater than 0.01 mm.");
    assert_eq!(world.state().sketch_text.unwrap().height_mm, "6.10");
    assert_eq!(world.app().owner.sketch_preview(), Some(valid_preview));
    text_linked_journey::capture_when_requested(world, "height-fraction-warning");
    acknowledge_number_warning(world, "Text Height");
}

fn accept_typed_six_as_preview(world: &mut ApplicationWorld, family: &str) {
    enter_text_number(world, b"6");
    world.expect_no_text("Enter a size greater than 0.01 mm.");
    world.run_in_window("Format Text", "6");
    world.run_in_window("Format Text", &family.replace(' ', ""));
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.height_mm, "6");
    assert_eq!(draft.family, family);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the fraction preview at the corrected height");
    };
    text_fraction::assert_stacked(&style);
    assert_eq!(style.height, 0.6);
    text_linked_journey::capture_when_requested(world, "height-fraction-typed-six");
}

#[test]
fn sketch_text_width_rejects_zero_and_restores_the_native_editor() {
    let mut world = ApplicationWorld::new();
    prepare_narrow_text(&mut world);
    let original = world.edited_geometry();
    text_edit::open_existing(&mut world);
    let preview = world.app().owner.sketch_preview().unwrap();
    focus_text_number(&mut world, "Width(%)");
    x11_input::press_control_a();
    world.pump(Vec::new());
    enter_text_number(&mut world, b"0");
    world.expect_text("Width is out of the expected range. Enter a positive width.");
    assert_eq!(world.app().owner.sketch_preview(), Some(preview.clone()));
    acknowledge_number_warning(&mut world, "Text Width");
    type_digits(&mut world, b"50");
    world.run_in_window("Format Text", "50");
    tab_out_of_number(&mut world);
    world.expect_no_text("Width is out of the expected range. Enter a positive width.");
    assert_eq!(world.app().owner.sketch_preview(), Some(preview));
    click_editor_button(&mut world, "Cancel");
    assert_eq!(world.edited_geometry(), original);
    world.close();
}

fn prepare_narrow_text(world: &mut ApplicationWorld) {
    text_edit::committed_b(world);
    text_edit::open_existing(world);
    let full = glyph_pixels(world);
    focus_text_number(world, "Width(%)");
    x11_input::press_control_a();
    world.pump(Vec::new());
    enter_text_number(world, b"50");
    assert_ne!(glyph_pixels(world), full);
    click_editor_button(world, "OK");
    let geometry = world.edited_geometry();
    let style = &geometry.texts()[0].style;
    let [run] = style.formatting_runs.as_deref().unwrap() else {
        panic!("expected one width run");
    };
    assert_eq!(run.width_percent.as_deref(), Some(&50.));
}

pub(super) fn focus_text_number(world: &mut ApplicationWorld, name: &str) {
    let label = world.run_in_window("Format Text", name);
    let position = [label.minimum[0] - 70., label.center()[1]];
    x11_input::click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
}

pub(super) fn enter_text_number(world: &mut ApplicationWorld, digits: &[u8]) {
    type_digits(world, digits);
    tab_out_of_number(world);
}

pub(super) fn type_digits(world: &mut ApplicationWorld, digits: &[u8]) {
    for digit in digits {
        x11_input::press_keysym(u32::from(*digit));
        world.pump(Vec::new());
    }
}

pub(super) fn tab_out_of_number(world: &mut ApplicationWorld) {
    x11_input::press_keysym(0xff09);
    world.pump(Vec::new());
}

fn acknowledge_number_warning(world: &mut ApplicationWorld, title: &str) {
    let button = world.run_in_window(title, "OK").center();
    x11_input::click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
}
