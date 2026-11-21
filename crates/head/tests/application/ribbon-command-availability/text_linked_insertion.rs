// SPDX-License-Identifier: GPL-2.0-only
//! Insert live fields using the native Format Text source and precision controls.
use super::*;
use oblikovati_api::{
    client::Client,
    types::{
        SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER,
        SKETCH_TEXT_FIELD_PLACEHOLDER,
    },
};
use text_format_choices::choose_format;

#[test]
fn native_linked_model_field_insertion_keeps_selected_precision() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let client = Client::new(world.app().owner.document_host());
    client
        .parameters()
        .add(&text_linked_viewport::dimension("10 mm"))
        .unwrap();
    activate_text(&mut world);
    open_editor(&mut world, [900, 580]);
    focus_content(&mut world);
    choose_format(&mut world, "Select field", "d0");
    choose_format(&mut world, "0.000", "0.00");
    click_editor_button(&mut world, "Insert");
    assert_field_preview(&mut world, "10.00");
    world.run_in_window("Format Text", "10.00");
    edit_around_field(&mut world);
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-text-numeric-field-editor");
    }
    accept_field(&mut world, "a10.00z");
    let saved = world.edited_geometry();
    let field = &saved.texts()[0].style.linked_fields.as_ref().unwrap()[0];
    assert_eq!(field.kind, SKETCH_TEXT_FIELD_MODEL_PARAMETER);
    assert_eq!(field.parameter, "d0");
    assert_eq!(field.precision.as_deref(), Some(&2));
    assert_eq!(field.at, 1);
    world.close();
}

fn edit_around_field(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    focus_content(world);
    for key in [0xff50, u32::from(b'a'), 0xff57, u32::from(b'z')] {
        x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
    let state = world.state();
    let draft = state.sketch_text.unwrap();
    assert_eq!(draft.content, format!("a{SKETCH_TEXT_FIELD_PLACEHOLDER}z"));
    let editor = state.sketch_text_editor_fields.unwrap();
    assert_eq!(editor.projection.content, "a10.00z");
    world.run_in_window("Format Text", "a10.00z");
}

#[test]
fn native_linked_file_name_field_offers_no_numeric_precision() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    activate_text(&mut world);
    open_editor(&mut world, [900, 580]);
    focus_content(&mut world);
    choose_format(&mut world, "Parameters - Model", "Standard iProperties");
    assert!(!world.run_is_enabled("Precision"));
    choose_format(&mut world, "Select field", "FILENAME");
    click_editor_button(&mut world, "Insert");
    assert_field_preview(&mut world, "Part1");
    let editor = world.state().sketch_text_editor_fields.unwrap();
    assert_eq!(editor.projection.content, "FILENAME");
    world.run_in_window("Format Text", "FILENAME");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-text-filename-field-editor");
    }
    remove_shown_file_name(&mut world);
    click_editor_button(&mut world, "Insert");
    assert_field_preview(&mut world, "Part1");
    accept_field(&mut world, "Part1");
    let saved = world.edited_geometry();
    let field = &saved.texts()[0].style.linked_fields.as_ref().unwrap()[0];
    assert_eq!(field.kind, SKETCH_TEXT_FIELD_FILE_NAME);
    assert!(field.precision.is_none());
    world.close();
}

fn remove_shown_file_name(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_keysym(0xff57);
    world.pump(Vec::new());
    x11_input::press_keysym(0xff08);
    world.pump(Vec::new());
    let draft = world.state().sketch_text.unwrap();
    assert!(draft.content.is_empty());
    assert!(draft.formatting.linked_fields.is_none());
}

fn assert_field_preview(world: &mut ApplicationWorld, expected: &str) {
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.content, SKETCH_TEXT_FIELD_PLACEHOLDER);
    assert_eq!(draft.formatting.linked_fields.unwrap().len(), 1);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected a linked field preview after Insert");
    };
    assert_eq!(style.content, expected);
    assert!(world.edited_geometry().texts().is_empty());
}

fn accept_field(world: &mut ApplicationWorld, expected: &str) {
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
    let native = &world.app().window.native;
    let scene = &native.sketch_scene_status().unwrap().sketches[0];
    assert_eq!(scene.geometry.texts()[0].style.content, expected);
    assert!(native.sketch_display_declines().unwrap().is_empty());
}
