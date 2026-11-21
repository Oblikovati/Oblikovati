// SPDX-License-Identifier: GPL-2.0-only
//! Exercise fraction replacement while a separate live dimension field remains visible.
use super::super::super::sketch::dimension_label_overlap::create_midpoint_dimensions;
use super::*;
use text_linked_journey::{
    add_default_numeric_text, capture_when_requested, complete_text_placement, expect_scene_text,
};

const FRACTION_TEXT_SLOT: usize = 1;
const DIAMETER_LABEL: &str = "AB⌀";

#[test]
fn native_fraction_history_restores_diameter_without_changing_linked_text() {
    let mut world = ApplicationWorld::new();
    accepted_fraction_after_diameter(&mut world);
    let fraction_pixels = text_format_choices::glyph_pixels(&mut world);
    text_edit::undo_edit(&mut world);
    previous_diameter_returns(&mut world);
    assert_ne!(
        text_format_choices::glyph_pixels(&mut world),
        fraction_pixels
    );
    text_edit::redo_edit(&mut world);
    restored_fraction_returns(&mut world);
    linked_dimension_still_displays(&mut world);
    assert_eq!(
        text_format_choices::glyph_pixels(&mut world),
        fraction_pixels
    );
    world.close();
}

#[given("an accepted stacked fraction")]
fn accepted_fraction_after_diameter(world: &mut ApplicationWorld) {
    create_midpoint_dimensions(world, "10", 1.);
    add_default_numeric_text(world);
    // The other Text adds one point with two freedoms to the actual sketch.
    world.seed_dimensions_needed(33);
    commit_diameter_label(world);
    world.expect_dimensions_needed(35);
    expect_scene_text(world, &["10.000", DIAMETER_LABEL]);
    world.text_creation_baseline = Some(world.edited_geometry());
    text_edit::open_accepted_text(world, FRACTION_TEXT_SLOT);
    text_fraction::replace_with_fraction(world);
    text_fraction::stack_selected_fraction(world);
    click_editor_button(world, "OK");
    world.expect_ready();
    expect_scene_text(world, &["10.000", "1/2"]);
}

fn commit_diameter_label(world: &mut ApplicationWorld) {
    activate_text(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, "DejaVu Sans");
    focus_content(world);
    for &letter in b"ab" {
        x11_input::press_keysym(u32::from(letter));
        world.pump(Vec::new());
    }
    text_case::uppercase_selection(world);
    text_numbering::append_diameter_symbol(world);
    assert_eq!(world.state().sketch_text.unwrap().content, DIAMETER_LABEL);
    text_color::select_blue(world);
    complete_text_placement(world);
}

#[then("the previous AB diameter content returns")]
fn previous_diameter_returns(world: &mut ApplicationWorld) {
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
    expect_scene_text(world, &["10.000", DIAMETER_LABEL]);
    capture_when_requested(world, "diameter-after-fraction-undo");
}

#[then("the stacked fraction returns with 35 dimensions needed")]
fn restored_fraction_returns(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_at_ready);
    world.expect_dimensions_needed(35);
    assert_eq!(world.dimensions_needed(), 35);
    let geometry = world.edited_geometry();
    let baseline = world.text_creation_baseline.as_ref().unwrap();
    let fraction = &geometry.texts()[FRACTION_TEXT_SLOT];
    assert_eq!(fraction.id, baseline.texts()[FRACTION_TEXT_SLOT].id);
    assert_eq!(fraction.anchor, baseline.texts()[FRACTION_TEXT_SLOT].anchor);
    text_fraction::assert_stacked(&fraction.style);
    capture_when_requested(world, "stacked-fraction-after-redo");
}

#[then("the linked Text remains 10.000")]
fn linked_dimension_still_displays(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(
        geometry.texts()[0],
        world.text_creation_baseline.as_ref().unwrap().texts()[0]
    );
    expect_scene_text(world, &["10.000", "1/2"]);
}
