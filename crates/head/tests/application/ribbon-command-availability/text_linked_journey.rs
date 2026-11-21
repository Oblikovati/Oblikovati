// SPDX-License-Identifier: GPL-2.0-only
//! Exercise live Text fields against a real dimension, through document save and reopen.
use super::super::super::sketch::dimension_label_overlap::{
    accept_dimension_digits, create_midpoint_dimensions, edit_dimension_with_mouse,
    open_dimension_with_mouse,
};
use super::*;
use text_format_choices::{choose_format, reopen_text_sketch_from_file};

#[test]
fn native_mixed_text_fields_follow_dimension_history_and_file_reopen() {
    let mut world = ApplicationWorld::new();
    create_midpoint_dimensions(&mut world, "10", 1.);
    add_default_numeric_text(&mut world);
    add_mixed_text_fields(&mut world);
    verify_live_dimension_history(&mut world);
    verify_native_file_round_trip(&mut world);
    world.close();
}

#[given("Text displays the inserted d0 value 10.000")]
fn place_linked_dimension_text(world: &mut ApplicationWorld) {
    create_midpoint_dimensions(world, "10", 1.);
    add_default_numeric_text(world);
    world.text_creation_baseline = Some(world.edited_geometry());
}

#[when("the user opens the dimension identified as d0")]
fn open_linked_dimension(world: &mut ApplicationWorld) {
    open_dimension_with_mouse(world, "10", "d0");
}

#[when("the user enters 1 then 2 and accept")]
fn accept_linked_dimension(world: &mut ApplicationWorld) {
    accept_dimension_digits(world, "12");
}

#[then("its source annotation is 12 and Text displays 12.000")]
fn linked_dimension_is_updated(world: &mut ApplicationWorld) {
    world.expect_annotation("12");
    expect_scene_text(world, &["12.000"]);
    capture_when_requested(world, "linked-dimension-12");
}

#[when("the user undoes the source edit")]
fn undo_linked_dimension(world: &mut ApplicationWorld) {
    text_edit::undo_edit(world);
}

#[then("the source and Text return to 10 and 10.000 respectively")]
fn linked_dimension_is_restored(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    expect_scene_text(world, &["10.000"]);
    assert_eq!(Some(world.edited_geometry()), world.text_creation_baseline);
    capture_when_requested(world, "linked-dimension-undo-10");
}

/// Place one live field for the current sketch's `d0`, for example at 10 mm.
pub(super) fn add_default_numeric_text(world: &mut ApplicationWorld) {
    activate_text(world);
    text_drag_placement::drag_text_to(world, [1050, 710], [1250, 810]);
    assert!(world.state().sketch_text.unwrap().content.is_empty());
    focus_content(world);
    insert_model_dimension(world, None);
    world.run_in_window("Format Text", "10.000");
    complete_text_placement(world);
    expect_scene_text(world, &["10.000"]);
}

fn add_mixed_text_fields(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, [950, 620]);
    choose_format(world, "Parameters - Model", "Standard iProperties");
    assert!(!world.run_is_enabled("Precision"));
    choose_format(world, "Select field", "FILENAME");
    focus_content(world);
    click_editor_button(world, "Insert");
    world.run_in_window("Format Text", "FILENAME");
    append_text_paragraph(world);
    inspect_empty_parameter_sources(world);
    insert_model_dimension(world, Some("0.00"));
    let editor = world.state().sketch_text_editor_fields.unwrap();
    assert_eq!(editor.projection.content, "FILENAME\n10.00");
    capture_when_requested(world, "mixed-linked-text-editor");
    complete_text_placement(world);
    expect_scene_text(world, &["10.000", "Part1\n10.00"]);
}

fn inspect_empty_parameter_sources(world: &mut ApplicationWorld) {
    choose_format(world, "Standard iProperties", "Parameters - Finish");
    assert!(!world.run_is_enabled("Field"));
    assert!(!world.window_text("Format Text").contains("SourcePart1"));
    choose_format(world, "Parameters - Finish", "Parameters - User");
    assert!(!world.run_is_enabled("Field"));
    assert!(world.window_text("Format Text").contains("SourcePart1"));
    choose_format(world, "Parameters - User", "Parameters - Model");
}

fn append_text_paragraph(world: &mut ApplicationWorld) {
    focus_content(world);
    for key in [0xff57, 0xff0d] {
        x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
}

fn insert_model_dimension(world: &mut ApplicationWorld, precision: Option<&str>) {
    choose_format(world, "Select field", "d0");
    if let Some(precision) = precision {
        choose_format(world, "0.000", precision);
    }
    click_editor_button(world, "Insert");
}

/// Accept new Text and exit repeated placement, for example after inserting a field.
pub(super) fn complete_text_placement(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
    x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_ready();
}

fn verify_live_dimension_history(world: &mut ApplicationWorld) {
    let before = world.dimensions_needed();
    edit_dimension_with_mouse(world, "10", "d0", "12");
    world.expect_annotation("12");
    expect_scene_text(world, &["12.000", "Part1\n12.00"]);
    capture_when_requested(world, "linked-text-after-dimension-edit");
    change_history(world, b'z');
    world.expect_annotation("10");
    expect_scene_text(world, &["10.000", "Part1\n10.00"]);
    change_history(world, b'y');
    expect_scene_text(world, &["12.000", "Part1\n12.00"]);
    change_history(world, b'z');
    assert_eq!(world.dimensions_needed(), before);
}

fn change_history(world: &mut ApplicationWorld, key: u8) {
    x11_input::press_control_letter(key);
    world.pump(Vec::new());
}

fn verify_native_file_round_trip(world: &mut ApplicationWorld) {
    let authored = world.edited_geometry();
    let degrees = world.dimensions_needed();
    reopen_text_sketch_from_file(world);
    let path = world.saved_path.as_ref().unwrap();
    let file_name = path.file_name().unwrap().to_str().unwrap().to_owned();
    expect_scene_text(world, &["10.000", &format!("{file_name}\n10.00")]);
    assert_eq!(world.edited_geometry(), authored);
    assert_eq!(world.dimensions_needed(), degrees);
    edit_dimension_with_mouse(world, "10", "d0", "12");
    expect_scene_text(world, &["12.000", &format!("{file_name}\n12.00")]);
    capture_when_requested(world, "linked-text-after-native-reopen");
}

/// Compare actual resolved scene text in creation order, for example `&["10.000"]`.
pub(super) fn expect_scene_text(world: &mut ApplicationWorld, expected: &[&str]) {
    let native = &world.app().window.native;
    assert!(native.sketch_display_declines().unwrap().is_empty());
    let scene = &native.sketch_scene_status().unwrap().sketches[0];
    let displayed: Vec<_> = scene
        .geometry
        .texts()
        .iter()
        .map(|text| text.style.content.as_str())
        .collect();
    assert_eq!(displayed, expected);
}

/// Save a named native frame when a report folder is configured, for example after Undo.
pub(super) fn capture_when_requested(world: &mut ApplicationWorld, name: &str) {
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture(name);
    }
}
