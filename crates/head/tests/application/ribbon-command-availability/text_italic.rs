// SPDX-License-Identifier: GPL-2.0-only
//! Edit a uniform italic face and verify the original and edited outlines through history.
use super::*;
use oblikovati_model::sketch::Sketch2D;
use oblikovati_viewport::ViewportCapture;

const BEFORE_TEXT: usize = 31;
const WITH_TEXT: usize = 33;

#[test]
fn native_italic_text_history_keeps_bold_content_and_anchor() {
    let mut world = ApplicationWorld::new();
    existing_bold_text(&mut world);
    select_bold_text(&mut world);
    accept_italic(&mut world);
    expect_italic_ready(&mut world);
    text_edit::undo_edit(&mut world);
    expect_upright_bold(&mut world);
    text_edit::redo_edit(&mut world);
    expect_italic_again(&mut world);
    world.close();
}

#[given("existing bold characters")]
fn existing_bold_text(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    // A free insertion point adds two degrees of freedom to the real sketch fixture.
    world.seed_dimensions_needed(BEFORE_TEXT);
    activate_text(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, "DejaVu Sans");
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    select_all_text(world);
    click_editor_button(world, "Bold");
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    record_upright_fixture(world);
}

fn record_upright_fixture(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(world.dimensions_needed(), WITH_TEXT);
    let original = world.edited_geometry();
    let [text] = original.texts() else {
        panic!("expected one text entity");
    };
    let [run] = text.style.formatting_runs.as_deref().unwrap() else {
        panic!("expected one Bold run");
    };
    assert_eq!((run.bold, run.italic), (true, false));
    world.text_creation_baseline = Some(original);
    world.sketch_baseline = Some(capture(world));
}

#[when("the user reopens Text, focus the editor and select both characters")]
fn select_bold_text(world: &mut ApplicationWorld) {
    text_edit::open_existing(world);
    focus_content(world);
    select_all_text(world);
}

fn select_all_text(world: &mut ApplicationWorld) {
    x11_input::press_control_a();
    world.pump(Vec::new());
}

#[when("the user applies Italic and accept")]
fn accept_italic(world: &mut ApplicationWorld) {
    let before = text_format_choices::glyph_pixels(world);
    click_editor_button(world, "Italic");
    assert_ne!(text_format_choices::glyph_pixels(world), before);
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("bold-italic-preview");
    }
    click_editor_button(world, "OK");
}

#[then("slanted bold characters remain at Ready with 33 dimensions needed")]
fn expect_italic_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    verify_italic_recipe(world);
    let slanted = capture(world);
    let upright = world.sketch_baseline.as_ref().unwrap();
    assert_ne!(
        text_format_choices::glyph_region(&slanted),
        text_format_choices::glyph_region(upright)
    );
    world.last_capture = Some(slanted);
}

#[then("upright bold characters remain")]
fn expect_upright_bold(world: &mut ApplicationWorld) {
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
    assert_eq!(world.dimensions_needed(), WITH_TEXT);
    let current = text_format_choices::glyph_pixels(world);
    assert_eq!(
        current,
        text_format_choices::glyph_region(world.sketch_baseline.as_ref().unwrap())
    );
}

#[then("the slanted appearance returns without changing the 33-dimension count")]
fn expect_italic_again(world: &mut ApplicationWorld) {
    verify_italic_recipe(world);
    let current = text_format_choices::glyph_pixels(world);
    assert_eq!(
        current,
        text_format_choices::glyph_region(world.last_capture.as_ref().unwrap())
    );
}

fn verify_italic_recipe(world: &mut ApplicationWorld) {
    assert_eq!(world.dimensions_needed(), WITH_TEXT);
    world.expect_dimensions_needed(WITH_TEXT);
    assert_eq!(
        world.edited_geometry(),
        expected_italic(world.text_creation_baseline.as_ref().unwrap())
    );
}

fn expected_italic(before: &Sketch2D) -> Sketch2D {
    let text = &before.texts()[0];
    let mut style = text.style.clone();
    style.formatting_runs.as_mut().unwrap()[0].italic = true;
    let mut geometry = before.clone();
    geometry.set_text_style(text.id, style).unwrap();
    geometry
}

fn capture(world: &mut ApplicationWorld) -> ViewportCapture {
    super::super::super::capture_framebuffer(&mut world.app().window.native)
}
