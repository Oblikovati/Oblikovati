// SPDX-License-Identifier: GPL-2.0-only
//! Measure anchored Text creation on the two recorded baseline counts.
use super::*;
use oblikovati_model::sketch::Sketch2D;
use text_format_choices::{choose_format, glyph_pixels, reopen_text_sketch_from_file};
use text_linked_journey::{capture_when_requested, complete_text_placement};

#[test]
fn native_point_text_adds_only_anchor_freedoms_to_the_210_dimension_fixture() {
    let mut world = ApplicationWorld::new();
    let before = open_saved_text_fixture(&mut world, 210);
    open_free_text_editor(&mut world, [450, 650]);
    choose_format(&mut world, "Choose height", "6.10 mm");
    type_content(&mut world, b'a');
    complete_text_placement(&mut world);
    let after = expect_text_anchor_addition(&mut world, &before, 212);
    assert_eq!(after.texts()[0].style.content, "a");
    assert_eq!(after.texts()[0].style.height, 0.61);
    capture_when_requested(&mut world, "text-point-ready-212");
    world.close();
}

#[test]
fn native_bold_text_undo_redo_restores_the_same_anchor_and_freedoms() {
    let mut world = ApplicationWorld::new();
    let before = open_saved_text_fixture(&mut world, 28);
    open_free_text_editor(&mut world, [900, 580]);
    assert_empty_text_default(&mut world);
    apply_bold_to_two_typed_characters(&mut world);
    complete_text_placement(&mut world);
    let after = expect_text_anchor_addition(&mut world, &before, 30);
    check_bold_history(&mut world, &before, &after);
    world.close();
}

fn open_saved_text_fixture(world: &mut ApplicationWorld, count: usize) -> Sketch2D {
    world.start_empty_sketch();
    world.seed_dimensions_needed(count);
    let original = world.edited_geometry();
    reopen_text_sketch_from_file(world);
    assert_eq!(world.edited_geometry(), original);
    world.expect_ready();
    world.expect_dimensions_needed(count);
    if let Some(reports) = std::env::var_os("OBK_BDD_REPORTS") {
        let saved = world.saved_path.as_ref().unwrap();
        std::fs::copy(saved, std::path::PathBuf::from(reports).join("Part1.opd")).unwrap();
    }
    original
}

fn open_free_text_editor(world: &mut ApplicationWorld, point: [i32; 2]) {
    activate_text(world);
    open_editor(world, point);
    assert!(world.state().sketch_text.unwrap().content.is_empty());
}

fn assert_empty_text_default(world: &mut ApplicationWorld) {
    let state = world.state();
    let available = state.sketch_text_font_families.unwrap();
    let draft = state.sketch_text.unwrap();
    let family = if available.iter().any(|name| name == "Tahoma") {
        "Tahoma"
    } else {
        ""
    };
    assert_eq!(draft.family, family);
    assert_eq!(draft.height_mm, "3.05");
}

fn apply_bold_to_two_typed_characters(world: &mut ApplicationWorld) {
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    x11_input::press_control_a();
    world.pump(Vec::new());
    let regular = glyph_pixels(world);
    click_editor_button(world, "Bold");
    assert_ne!(glyph_pixels(world), regular);
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.content, "ab");
    let [run] = draft.formatting.runs.as_deref().unwrap() else {
        panic!("expected one Bold run for ab");
    };
    assert_eq!((run.start, run.end, run.bold), (0, 2, true));
    capture_when_requested(world, "text-bold-creation-preview");
}

fn expect_text_anchor_addition(
    world: &mut ApplicationWorld,
    before: &Sketch2D,
    count: usize,
) -> Sketch2D {
    world.expect_ready();
    world.expect_dimensions_needed(count);
    let after = world.edited_geometry();
    let [text] = after.texts() else {
        panic!("expected a single committed Text object");
    };
    let (existing, added) = after.entity_ids().split_at(before.entity_ids().len());
    assert_eq!(existing, before.entity_ids());
    assert_eq!(added, [text.id]);
    let (anchor, earlier_points) = after.points().split_last().unwrap();
    assert_eq!(anchor.id, text.anchor);
    assert_eq!(earlier_points, before.points());
    assert_eq!(after.constraints(), before.constraints());
    let glyph = text_edit::glyph_pixel(world);
    world.expect_segment_drawn(glyph, glyph);
    after
}

fn check_bold_history(world: &mut ApplicationWorld, before: &Sketch2D, after: &Sketch2D) {
    let pixels = glyph_pixels(world);
    capture_when_requested(world, "text-bold-ready-30");
    text_edit::undo_edit(world);
    world.expect_dimensions_needed(28);
    assert_eq!(&world.edited_geometry(), before);
    text_edit::redo_edit(world);
    world.expect_dimensions_needed(30);
    assert_eq!(&world.edited_geometry(), after);
    assert_eq!(glyph_pixels(world), pixels);
    capture_when_requested(world, "text-bold-redo-30");
}
