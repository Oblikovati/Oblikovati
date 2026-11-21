// SPDX-License-Identifier: GPL-2.0-only
//! Check the live preview of a different font and a larger height on existing text.
use super::*;
use oblikovati_api::types::SketchTextStyle;
use oblikovati_viewport::ViewportCapture;
use text_format_choices::{GLYPH_CROP_WIDTH, choose_font, choose_format, glyph_region};

#[given("existing bold italic characters are selected in Format Text")]
fn select_existing_bold_italic_text(world: &mut ApplicationWorld) {
    create_bold_italic_text(world);
    text_edit::open_existing(world);
    select_editor_content(world);
    world.text_creation_baseline = Some(world.edited_geometry());
}

fn create_bold_italic_text(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    choose_font(world, "DejaVu Sans");
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    select_editor_content(world);
    click_editor_button(world, "Bold");
    select_editor_content(world);
    click_editor_button(world, "Italic");
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
}

fn select_editor_content(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
}

#[when("the user opens the font list and choose Arial")]
fn apply_arial_to_preview(world: &mut ApplicationWorld) {
    assert_preview_face(world, "DejaVu Sans", 0.305);
    world.sketch_baseline = Some(capture_font_preview(world));
    choose_font(world, "Arial");
}

#[then("the font field shows Arial and the preview changes")]
fn check_arial_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().family, "Arial");
    world.run_in_window("Format Text", "Arial");
    assert_preview_face(world, "Arial", 0.305);
    let arial = capture_font_preview(world);
    let original = world.sketch_baseline.as_ref().unwrap();
    assert_ne!(glyph_region(&arial), glyph_region(original));
    world.sketch_baseline = Some(arial);
    world.save_capture("arial-preview");
}

#[when("the user opens the height list and choose 6.10 mm")]
fn apply_larger_height_to_preview(world: &mut ApplicationWorld) {
    choose_format(world, "Choose height", "6.10 mm");
}

#[then("the preview enlarges and the height field shows 6.10 mm")]
fn check_enlarged_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().height_mm, "6.10");
    world.run_in_window("Format Text", "6.10");
    assert_preview_face(world, "Arial", 0.61);
    let larger = capture_font_preview(world);
    let original = world.sketch_baseline.as_ref().unwrap();
    let (before, after) = (ink_height(original), ink_height(&larger));
    assert!(
        after > before,
        "glyph height {after}px: expected greater than {before}px"
    );
    world.save_capture("arial-enlarged-preview");
    click_editor_button(world, "Cancel");
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
}

fn assert_preview_face(world: &mut ApplicationWorld, family: &str, height: f64) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the live sketch text preview");
    };
    assert_eq!(style.family, family);
    assert_eq!(style.height, height);
    assert_bold_italic(&style);
}

fn assert_bold_italic(style: &SketchTextStyle) {
    assert_eq!(style.content, "ab");
    let [run] = style.formatting_runs.as_deref().unwrap_or_default() else {
        panic!("expected one uniform format run for both characters");
    };
    assert!(run.bold && run.italic);
}

fn capture_font_preview(world: &mut ApplicationWorld) -> ViewportCapture {
    super::super::super::capture_framebuffer(&mut world.app().window.native)
}

fn ink_height(capture: &ViewportCapture) -> usize {
    // The shared glyph crop excludes the editor and axes. Its white preview
    // outline can be measured against the dark canvas without using model size.
    let crop = glyph_region(capture);
    let mut rows = crop
        .as_chunks::<{ GLYPH_CROP_WIDTH * 4 }>()
        .0
        .iter()
        .enumerate()
        .filter_map(|(y, row)| {
            row.as_chunks::<4>()
                .0
                .iter()
                .any(|pixel| pixel[..3] == [255; 3])
                .then_some(y)
        });
    let first = rows.next().expect("expected visible white preview ink");
    rows.next_back().unwrap_or(first) - first + 1
}
