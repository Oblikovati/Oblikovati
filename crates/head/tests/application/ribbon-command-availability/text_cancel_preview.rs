// SPDX-License-Identifier: GPL-2.0-only
//! Check draft glyph removal against a real sketch with 210 remaining freedoms.
use super::*;
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_text};
use text_cancel_defaults::{TAHOMA, configure_blank_font_default, expect_blank_formatting_default};
use text_linked_journey::capture_when_requested;

const ORIGINAL_FREEDOMS: usize = 210;
const INSERTION: [i32; 2] = [450, 650];

#[test]
fn native_text_cancel_removes_visible_preview_and_preserves_solver_count() {
    let mut world = ApplicationWorld::new();
    build_preview_fixture(&mut world, "DejaVu Sans");
    open_preview_editor(&mut world);
    expect_blank_formatting_default(&mut world, "DejaVu Sans");
    type_lowercase_a(&mut world);
    preview_glyph_is_visible(&mut world);
    click_editor_button(&mut world, "Cancel");
    pending_glyph_is_gone(&mut world);
    placement_still_active(&mut world);
    x11_input::press_escape();
    world.pump(Vec::new());
    cancelled_sketch_is_unchanged(&mut world);
    world.close();
}

#[given("an editable sketch with 210 dimensions needed")]
fn configure_recorded_preview_fixture(world: &mut ApplicationWorld) {
    build_preview_fixture(world, TAHOMA);
}

fn build_preview_fixture(world: &mut ApplicationWorld, family: &str) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(ORIGINAL_FREEDOMS);
    configure_blank_font_default(world, family);
    x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_ready();
    world.expect_dimensions_needed(ORIGINAL_FREEDOMS);
    world.text_creation_baseline = Some(world.edited_geometry());
    world.sketch_baseline = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}

#[when("the user starts Text and select insertion location (450,650)")]
fn open_preview_editor(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, INSERTION);
}

#[then("Format Text opens with an empty editor and Tahoma, 3.05 mm and width 100")]
fn recorded_preview_defaults(world: &mut ApplicationWorld) {
    expect_blank_formatting_default(world, TAHOMA);
}

#[when("the user focuses the editor and press lowercase a")]
fn type_lowercase_a(world: &mut ApplicationWorld) {
    type_content(world, b'a');
}

#[then("a appears in the editor and the live canvas preview")]
fn preview_glyph_is_visible(world: &mut ApplicationWorld) {
    world.run_in_window("Format Text", "a");
    let witness = preview_glyph_pixel(world);
    world.observed_view = Some(witness);
    world.expect_segment_drawn(witness, witness);
    capture_when_requested(world, "text-pending-preview");
}

fn preview_glyph_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let Some(ViewportSketchPreview::Text {
        style,
        anchor,
        working_scale,
        ..
    }) = world.app().owner.sketch_preview()
    else {
        panic!("expected an active preview for lowercase a");
    };
    assert_eq!(style.content, "a");
    assert_eq!(anchor, world.sketch_point_of(INSERTION));
    let sampling = SketchSamplingOptionsV1 {
        working_scale,
        deflection: 0.001,
        maximum_curve_points: 4096,
    };
    let paths = sample_sketch_text(&style, anchor, sampling).unwrap();
    world.sketch_pixel_of(paths[0][0])
}

#[then("Format Text closes and the pending text preview disappears")]
fn pending_glyph_is_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Format Text");
    assert!(world.state().sketch_text.is_none());
    assert!(world.app().owner.sketch_preview().is_none());
    let witness = world.observed_view.unwrap();
    world.expect_segment_absent(witness, witness);
}

#[then("Text remains available for placement")]
fn placement_still_active(world: &mut ApplicationWorld) {
    world.expect_status("Select text insertion point");
    assert!(!world.state().sketch_at_ready);
}

#[then("the creation command exits without retained text or solver-count change")]
fn cancelled_sketch_is_unchanged(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert!(world.state().sketch_at_ready);
    world.expect_dimensions_needed(ORIGINAL_FREEDOMS);
    assert_eq!(world.dimensions_needed(), ORIGINAL_FREEDOMS);
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
    assert!(world.app().owner.sketch_preview().is_none());
    capture_when_requested(world, "text-preview-cancelled-sketch");
}
