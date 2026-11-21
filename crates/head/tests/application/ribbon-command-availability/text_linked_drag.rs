// SPDX-License-Identifier: GPL-2.0-only
//! Sketch2 placement by mouse drag followed by typed model-field insertion.
use super::*;
use oblikovati_api::types::{SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER};
use text_expanded_reopen::{capture_portable_part, open_sketch_two_editor, reopened_scene_text};
use text_linked_journey::{
    capture_when_requested, complete_text_placement, insert_model_dimension,
};

const DRAG_FROM: [i32; 2] = [1050, 800];
const DRAG_TO: [i32; 2] = [1250, 900];
const WINDOW_PX: [u32; 2] = [1440, 1000];
const BEFORE_TEXT_FREEDOMS: usize = 36;

#[given("Text is active in Part6 Sketch2")]
fn activate_text_in_part6_sketch_two(world: &mut ApplicationWorld) {
    x11_input::resize_application_window(WINDOW_PX);
    world.pump(Vec::new());
    assert_eq!(x11_input::application_bounds()[2..], [1440, 1000]);
    let part = text_mixed_fields::begin_part6_sketch_two(world);
    world.seed_dimensions_needed(BEFORE_TEXT_FREEDOMS);
    world.finish_sketch();
    let path = world.save_part_to_file(part, "Part6.opd");
    capture_portable_part(&path);
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    open_sketch_two_editor(world);
    world.expect_dimensions_needed(BEFORE_TEXT_FREEDOMS);
    assert!(world.edited_geometry().texts().is_empty());
    activate_text(world);
}

#[when("the user drags from 1050,800 to 1250,900 and release")]
fn drag_on_sketch_canvas(world: &mut ApplicationWorld) {
    world.sketch_point_of(DRAG_TO);
    text_drag_placement::drag_text_to(world, DRAG_FROM, DRAG_TO);
}

#[then("an empty Format Text editor opens")]
fn drag_opens_empty_editor(world: &mut ApplicationWorld) {
    world.expect_text("Format Text");
    assert!(world.state().sketch_text.unwrap().content.is_empty());
    assert!(world.edited_geometry().texts().is_empty());
}

#[when("the user focuses the editor and insert the displayed d0 model parameter")]
fn insert_d0_into_dragged_text(world: &mut ApplicationWorld) {
    focus_content(world);
    insert_model_dimension(world, None);
}

#[then("a shaded 10.000 field appears without digit entry")]
fn confirm_projected_drag_field(world: &mut ApplicationWorld) {
    let editor = world.state().sketch_text_editor_fields.unwrap();
    assert_eq!(editor.authored, SKETCH_TEXT_FIELD_PLACEHOLDER);
    let [span] = editor.projection.fields.as_slice() else {
        panic!("expected one shaded model field");
    };
    assert_eq!(&editor.projection.content[span.display.clone()], "10.000");
    world.run_in_window("Format Text", "10.000");
    capture_when_requested(world, "linked-d0-drag-editor");
}

#[when("the user accepts and exit repeated placement")]
fn finish_dragged_text(world: &mut ApplicationWorld) {
    complete_text_placement(world);
}

#[then("the displayed value remains at Ready with 38 dimensions needed")]
fn confirm_committed_dragged_field(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(38);
    assert_eq!(reopened_scene_text(world), ["10.000"]);
    let geometry = world.edited_geometry();
    let text = &geometry.texts()[0];
    assert_eq!(
        geometry.point(text.anchor).unwrap().position,
        world.sketch_point_of(DRAG_FROM)
    );
    let field = &text.style.linked_fields.as_ref().unwrap()[0];
    assert_eq!(field.kind, SKETCH_TEXT_FIELD_MODEL_PARAMETER);
    assert_eq!(field.parameter, "d0");
    capture_when_requested(world, "linked-d0-drag-at-ready");
}
