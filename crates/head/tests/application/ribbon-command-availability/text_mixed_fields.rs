// SPDX-License-Identifier: GPL-2.0-only
//! Inspect linked field sources and commit mixed properties on the same native Part fixture.
use super::super::super::sketch::dimension_label_overlap::place_midpoint_dimensions;
use super::*;
use oblikovati_api::types::{SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER};
use text_expanded_reopen::{
    capture_portable_part, open_sketch_two_editor, reopened_scene_text,
    seed_text_before_mixed_fields,
};
use text_format_choices::choose_format;
use text_linked_journey::{append_text_paragraph, capture_when_requested, complete_text_placement};

const FILE_NAME: &str = "Part6.opd";
const MIXED_PREVIEW: &str = "Part6.opd\n10.00";
const BEFORE_COUNT: usize = 45;

#[given("a fresh empty Format Text editor in Part6")]
fn start_part6_text(world: &mut ApplicationWorld) {
    for _ in 0..6 {
        world.add_part();
    }
    let part = world.active_id();
    assert_eq!(world.document_name(part), "Part6");
    world.add_finished_sketch();
    world.start_sketch();
    place_midpoint_dimensions(world, "10");
    seed_text_before_mixed_fields(world);
    world.seed_dimensions_needed(BEFORE_COUNT);
    world.finish_sketch();
    let path = world.save_part_to_file(part, FILE_NAME);
    capture_portable_part(&path);
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    open_sketch_two_editor(world);
    assert_eq!(reopened_scene_text(world)[3], "10.000");
    world.expect_dimensions_needed(BEFORE_COUNT);
    activate_text(world);
    open_editor(world, [975, 600]);
    assert!(world.state().sketch_text.unwrap().content.is_empty());
}

#[when("the user opens Type and select Standard iProperties")]
fn show_standard_properties(world: &mut ApplicationWorld) {
    choose_format(world, "Parameters - Model", "Standard iProperties");
}

#[then("Source shows the primary Part6 model and Precision is unavailable")]
fn standard_source_is_part6(world: &mut ApplicationWorld) {
    assert_editor_source(world, "Part6");
    assert!(!world.run_is_enabled("Precision"));
}

#[when("the user selects FILENAME, focus the editor, and click Insert")]
fn insert_filename(world: &mut ApplicationWorld) {
    choose_format(world, "Select field", "FILENAME");
    focus_content(world);
    click_editor_button(world, "Insert");
}

#[then(
    "the editor shows a shaded FILENAME field and the canvas shows the current document filename"
)]
fn show_filename_editor_and_preview(world: &mut ApplicationWorld) {
    filename_is_retained(world);
    let editor = world.state().sketch_text_editor_fields.unwrap();
    let span = &editor.projection.fields[0];
    assert_eq!(&editor.projection.content[span.display.clone()], "FILENAME");
    expect_field_preview(world, FILE_NAME);
    capture_when_requested(world, "filename-preview");
}

#[when("the user selects Parameters - Finish")]
fn show_finish_parameters(world: &mut ApplicationWorld) {
    choose_format(world, "Standard iProperties", "Parameters - Finish");
}

#[then("Source is empty and no parameter is available in this fixture")]
fn finish_source_is_unavailable(world: &mut ApplicationWorld) {
    assert_editor_source(world, "Selectfield");
    assert!(!world.run_is_enabled("Field"));
    assert!(!world.run_is_enabled("Insert"));
}

#[when("the user selects Parameters - User")]
fn show_user_parameters(world: &mut ApplicationWorld) {
    choose_format(world, "Parameters - Finish", "Parameters - User");
}

#[then("Source is Part6 and no parameter is available in this fixture")]
fn user_source_has_no_parameters(world: &mut ApplicationWorld) {
    assert_editor_source(world, "Part6");
    assert!(!world.run_is_enabled("Field"));
    assert!(!world.run_is_enabled("Insert"));
}

fn assert_editor_source(world: &ApplicationWorld, next: &str) {
    let text = world.window_text("Format Text").replace(' ', "");
    assert!(
        text.contains(&format!("Source{next}")),
        "expected Source followed by {next:?} in {text:?}"
    );
}

#[then("the inserted filename field remains")]
fn filename_is_retained(world: &mut ApplicationWorld) {
    let editor = world.state().sketch_text_editor_fields.unwrap();
    assert_eq!(editor.projection.content, "FILENAME");
    assert_eq!(editor.fields.len(), 1);
    assert_eq!(editor.fields[0].kind, SKETCH_TEXT_FIELD_FILE_NAME);
    world.run_in_window("Format Text", "FILENAME");
}

#[given("the filename property remains in the fresh editor")]
fn start_mixed_field_editor(world: &mut ApplicationWorld) {
    start_part6_text(world);
    show_standard_properties(world);
    insert_filename(world);
    filename_is_retained(world);
}

#[when("the user selects Parameters - Model and inspect Precision")]
fn open_model_precision(world: &mut ApplicationWorld) {
    choose_format(world, "Standard iProperties", "Parameters - Model");
    click_visible(world, "0.000");
}

#[then("zero through eight decimal examples are offered")]
fn precision_lists_nine_options(world: &mut ApplicationWorld) {
    for digits in 0..=7 {
        let caption = if digits == 0 {
            "0".into()
        } else {
            format!("0.{}", "0".repeat(digits))
        };
        world.exact_run_center(&caption);
    }
    capture_when_requested(world, "model-field-precision");
    let list = world.exact_run_center("0.00000");
    x11_input::scroll(list, -1);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.exact_run_center("0.00000000");
    capture_when_requested(world, "model-field-precision-scrolled");
    x11_input::scroll(list, 1);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[when("the user chooses the two-decimal example")]
fn choose_two_decimals(world: &mut ApplicationWorld) {
    click_visible(world, "0.00");
}

#[when("the user focuses the editor, move to its end, press Return, and insert d0")]
fn append_model_field(world: &mut ApplicationWorld) {
    choose_format(world, "Select field", "d0");
    append_text_paragraph(world);
    click_editor_button(world, "Insert");
}

#[then("the added model field displays 10.00")]
fn model_field_is_two_decimals(world: &mut ApplicationWorld) {
    let editor = world.state().sketch_text_editor_fields.unwrap();
    assert_eq!(editor.projection.content, "FILENAME\n10.00");
    let field = &editor.fields[1];
    assert_eq!(field.kind, SKETCH_TEXT_FIELD_MODEL_PARAMETER);
    assert_eq!(field.parameter, "d0");
    assert_eq!(field.precision.as_deref(), Some(&2));
    expect_field_preview(world, MIXED_PREVIEW);
}

#[when("the user accepts Format Text and press Escape")]
fn commit_mixed_fields(world: &mut ApplicationWorld) {
    complete_text_placement(world);
}

#[then("the current document filename and 10.00 remain in separate paragraphs at Ready")]
fn mixed_fields_remain(world: &mut ApplicationWorld) {
    world.expect_ready();
    let shown = reopened_scene_text(world);
    assert_eq!(shown.len(), 5);
    assert_eq!(shown[4], MIXED_PREVIEW);
    capture_when_requested(world, "mixed-fields-at-ready");
}

#[then("the earlier linked Text still displays 10.000")]
fn earlier_field_remains(world: &mut ApplicationWorld) {
    assert_eq!(reopened_scene_text(world)[3], "10.000");
}

#[then("the status shows 47 dimensions needed")]
fn mixed_text_adds_two_freedoms(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(47);
}

fn click_visible(world: &mut ApplicationWorld, text: &str) {
    x11_input::click(world.exact_run_center(text));
    world.pump(Vec::new());
}

fn expect_field_preview(world: &mut ApplicationWorld, expected: &str) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected a linked Text preview for {expected:?}");
    };
    assert_eq!(style.content, expected);
}
