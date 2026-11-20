// SPDX-License-Identifier: GPL-2.0-only
//! Native Text creation and history cases from the published Sketch Text feature.
use super::*;

const RECORDED_INSERTION: [i32; 2] = [450, 650];
const EMPTY_INSERTION: [i32; 2] = [900, 580];
const EMPTY_SKETCH_DIMENSIONS: usize = 28;

#[given("an empty text editor is open in an editable sketch")]
#[given("Text has opened an empty Format Text editor in the 28-dimension fixture")]
fn open_empty_fixture_editor(world: &mut ApplicationWorld) {
    if world.app().owner.active_sketch_edit().is_none() {
        world.start_empty_sketch();
    }
    world.seed_dimensions_needed(EMPTY_SKETCH_DIMENSIONS);
    observe_creation_baseline(world);
    activate_text(world);
    open_editor(world, EMPTY_INSERTION);
}

#[when("the user clicks OK without entering content")]
fn accept_empty_content(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
}

#[then("the editor closes without validation or visible text")]
#[then("the editor closes without a validation dialog or visible text")]
fn no_empty_text_entity(world: &mut ApplicationWorld) {
    assert!(!world.shows("Format Text"));
    assert!(world.state().sketch_text.is_none());
    assert!(world.app().owner.sketch_preview().is_none());
    assert_creation_undone(world);
    world.expect_status("Select text insertion point");
}

#[then("the placement prompt remains with 28 dimensions needed")]
fn empty_acceptance_keeps_placing(world: &mut ApplicationWorld) {
    assert_eq!(world.dimensions_needed(), EMPTY_SKETCH_DIMENSIONS);
    world.expect_dimensions_needed(EMPTY_SKETCH_DIMENSIONS);
    world.expect_status("Select text insertion point");
}

#[when("the user chooses the same insertion location")]
fn open_editor_at_same_location(world: &mut ApplicationWorld) {
    open_editor(world, EMPTY_INSERTION);
}

#[then("the new editor is empty")]
fn reopened_draft_is_empty(world: &mut ApplicationWorld) {
    world.expect_text("Format Text");
    assert!(world.state().sketch_text.unwrap().content.is_empty());
}

#[given("the user has committed one valid change through this feature")]
fn commit_from_active_sketch(world: &mut ApplicationWorld) {
    observe_creation_baseline(world);
    activate_text(world);
    open_editor(world, EMPTY_INSERTION);
    type_content(world, b'b');
    click_editor_button(world, "OK");
    observe_committed_text(world, EMPTY_INSERTION);
}

#[given("an editable sketch and Text is active")]
fn activate_text_in_editable_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    observe_creation_baseline(world);
    activate_text(world);
}

fn observe_creation_baseline(world: &mut ApplicationWorld) {
    world.text_creation_baseline = Some(world.edited_geometry());
    world.observed_counts = vec![world.dimensions_needed()];
}

#[when("the user selects insertion location (450,650)")]
fn pick_recorded_insertion(world: &mut ApplicationWorld) {
    open_editor(world, RECORDED_INSERTION);
}

#[when("the user focuses the empty Format Text editor and press lowercase b")]
fn enter_creation_content(world: &mut ApplicationWorld) {
    type_content(world, b'b');
}

#[then("b appears in the editor and live canvas preview")]
fn creation_preview_contains_b(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().content, "b");
    let Some(ViewportSketchPreview::Text {
        style, replaces, ..
    }) = world.app().owner.sketch_preview()
    else {
        panic!("expected the creation preview after typing b");
    };
    assert_eq!(style.content, "b");
    assert!(replaces.is_none());
    assert!(world.edited_geometry().texts().is_empty());
}

#[when("the user clicks OK and press Escape")]
fn commit_and_exit_creation(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    observe_committed_text(world, RECORDED_INSERTION);
}

fn observe_committed_text(world: &mut ApplicationWorld, pointer: [i32; 2]) {
    let expected = world.sketch_point_of(pointer);
    let geometry = world.edited_geometry();
    let [text] = geometry.texts() else {
        panic!("expected one committed text object");
    };
    assert_eq!(text.style.content, "b");
    assert_eq!(geometry.point(text.anchor).unwrap().position, expected);
    let count = world.dimensions_needed();
    world.observed_counts.push(count);
    let glyph = text_edit::glyph_pixel(world);
    world.remember_edited_sketch(vec![glyph]);
}

#[then("b remains visible and the sketch returns Ready")]
fn ready_after_text_creation(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_creation_restored(world);
}

#[when("the user presses Ctrl+Z once")]
fn undo_text_creation(world: &mut ApplicationWorld) {
    x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
}

#[when("the user presses Ctrl+Y once")]
fn redo_text_creation(world: &mut ApplicationWorld) {
    x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
}

#[then("the new b text disappears and the baseline state returns")]
fn assert_creation_undone(world: &mut ApplicationWorld) {
    let current = world.edited_geometry();
    assert_eq!(&current, world.text_creation_baseline.as_ref().unwrap());
    assert_eq!(world.dimensions_needed(), world.observed_counts[0]);
    if let Some(glyph) = world
        .observed_sketch
        .as_ref()
        .and_then(|sketch| sketch.witnesses.first())
        .copied()
    {
        world.expect_segment_absent(glyph, glyph);
    }
}

#[then("b returns with its original visible appearance and position")]
#[then("the visible text and solver count return to their pre-undo state")]
fn assert_creation_restored(world: &mut ApplicationWorld) {
    world.expect_edited_sketch_retained();
    assert_eq!(world.dimensions_needed(), world.observed_counts[1]);
    assert!(
        world
            .app()
            .window
            .native
            .sketch_display_declines()
            .unwrap()
            .is_empty()
    );
    assert!(world.app().owner.sketch_preview().is_none());
    assert!(world.state().sketch_at_ready);
}
