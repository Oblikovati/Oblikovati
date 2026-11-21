// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_uppercase_only_changes_selected_sketch_text() {
    let mut world = ApplicationWorld::new();
    open_unselected_lowercase(&mut world);
    uppercase_without_selection(&mut world);
    lowercase_remains(&mut world);
    uppercase_selected_content(&mut world);
    uppercase_preview_changes(&mut world);
    click_editor_button(&mut world, "OK");
    let edited = world.edited_geometry();
    let original = world.text_creation_baseline.as_ref().unwrap();
    assert_eq!(edited.texts()[0].style.content, "AB");
    assert_eq!(edited.texts()[0].id, original.texts()[0].id);
    assert_eq!(edited.texts()[0].anchor, original.texts()[0].anchor);
    world.close();
}

#[given("existing lowercase ab is unselected")]
fn open_unselected_lowercase(world: &mut ApplicationWorld) {
    create_lowercase_label(world);
    world.text_creation_baseline = Some(world.edited_geometry());
    text_edit::open_existing(world);
    expect_case_preview(world, "ab");
    world.last_capture = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}

#[when("the user chooses UPPERCASE")]
fn uppercase_without_selection(world: &mut ApplicationWorld) {
    click_editor_button(world, "UPPERCASE");
}

#[then("the lowercase content remains")]
fn lowercase_remains(world: &mut ApplicationWorld) {
    expect_case_preview(world, "ab");
    let current = text_format_choices::glyph_pixels(world);
    let original = world.last_capture.as_ref().unwrap();
    assert_eq!(current, text_format_choices::glyph_region(original));
}

#[when("the user focuses and select the content and choose UPPERCASE")]
fn uppercase_selected_content(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
    click_editor_button(world, "UPPERCASE");
}

#[then("AB appears in the editor and preview")]
fn uppercase_preview_changes(world: &mut ApplicationWorld) {
    expect_case_preview(world, "AB");
    let current = text_format_choices::glyph_pixels(world);
    let original = world.last_capture.as_ref().unwrap();
    assert_ne!(current, text_format_choices::glyph_region(original));
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("uppercase-selected-preview");
    }
}

fn create_lowercase_label(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, "DejaVu Sans");
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
}

/// Select and convert the test label, for example from `ab` to `AB`.
pub(super) fn uppercase_selection(world: &mut ApplicationWorld) {
    uppercase_selected_content(world);
    expect_case_preview(world, "AB");
}

fn expect_case_preview(world: &mut ApplicationWorld, expected: &str) {
    assert_eq!(world.state().sketch_text.unwrap().content, expected);
    world.run_in_window("Format Text", expected);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected {expected:?} in the native viewport");
    };
    assert_eq!(style.content, expected);
}
