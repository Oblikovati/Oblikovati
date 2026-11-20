// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_uppercase_only_changes_selected_sketch_text() {
    let mut world = ApplicationWorld::new();
    create_lowercase_label(&mut world);
    let original = world.edited_geometry();
    text_edit::open_existing(&mut world);
    let lowercase = text_format_choices::glyph_pixels(&mut world);
    click_editor_button(&mut world, "UPPERCASE");
    assert_eq!(world.state().sketch_text.unwrap().content, "ab");
    assert_eq!(text_format_choices::glyph_pixels(&mut world), lowercase);
    uppercase_selection(&mut world);
    assert_ne!(text_format_choices::glyph_pixels(&mut world), lowercase);
    click_editor_button(&mut world, "OK");
    let edited = world.edited_geometry();
    assert_eq!(edited.texts()[0].style.content, "AB");
    assert_eq!(edited.texts()[0].id, original.texts()[0].id);
    assert_eq!(edited.texts()[0].anchor, original.texts()[0].anchor);
    world.close();
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

fn uppercase_selection(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
    click_editor_button(world, "UPPERCASE");
    assert_eq!(world.state().sketch_text.unwrap().content, "AB");
    world.run_in_window("Format Text", "AB");
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected uppercase text in the native viewport");
    };
    assert_eq!(style.content, "AB");
}
