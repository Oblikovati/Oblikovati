// SPDX-License-Identifier: GPL-2.0-only
//! Drive context editing in the published Sketch Text scenarios.
use super::*;
use crate::HeadWindow;
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_text};
use oblikovati_model::sketch::SketchText2D;

const INSERT: [i32; 2] = [900, 580];

#[test]
fn sketch_text_context_replaces_the_same_entity_and_restores_history() {
    let mut world = ApplicationWorld::new();
    committed_b(&mut world);
    let original = world.edited_geometry();
    let baseline = world.state().dimensions_needed;
    cancel_replacement(&mut world);
    assert_eq!(world.edited_geometry(), original);
    open_existing(&mut world);
    accept_c(&mut world);
    assert_eq!(world.state().dimensions_needed, baseline);
    let edited = world.edited_geometry();
    assert_eq!(edited.texts()[0].id, original.texts()[0].id);
    assert_eq!(edited.texts()[0].anchor, original.texts()[0].anchor);
    undo_edit(&mut world);
    assert_eq!(world.edited_geometry(), original);
    redo_edit(&mut world);
    assert_eq!(world.edited_geometry(), edited);
    world.close();
}

#[given("committed b text and Ready sketch state")]
pub(super) fn committed_b(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, INSERT);
    type_content(world, b'b');
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    assert_committed(world, "b");
}

#[given("the b text remains after a canceled c replacement")]
fn cancel_replacement(world: &mut ApplicationWorld) {
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        committed_b(world);
    }
    open_existing(world);
    replace_with_c(world);
    click_editor_button(world, "Cancel");
    assert_committed(world, "b");
}

#[when("the user right-clicks the glyph and choose Edit text")]
#[when("the user chooses Edit text again")]
pub(super) fn open_existing(world: &mut ApplicationWorld) {
    open_accepted_text(world, 0);
}

/// Open a specific accepted Text through its native context menu.
/// For example, index 1 edits the second Text after a saved sketch reopens.
pub(super) fn open_accepted_text(world: &mut ApplicationWorld, index: usize) {
    let at = accepted_text_pixel(world, index);
    x11_input::right_click(at);
    world.pump(Vec::new());
    assert_eq!(
        world
            .state()
            .sketch_context
            .as_ref()
            .and_then(|context| context.text),
        Some(world.edited_geometry().texts()[index].id)
    );
    x11_input::click(diagnostics::pointer_pixels(world.run("Edit Text").center()));
    world.pump(Vec::new());
    world.expect_text("Format Text");
}

pub(super) fn glyph_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    accepted_text_pixel(world, 0)
}

fn accepted_text_pixel(world: &mut ApplicationWorld, index: usize) -> [i32; 2] {
    let geometry = world.edited_geometry();
    let text = &geometry.texts()[index];
    let anchor = geometry.point(text.anchor).unwrap().position;
    let paths = sample_sketch_text(
        &text.style,
        anchor,
        SketchSamplingOptionsV1 {
            working_scale: 1.,
            deflection: 0.001,
            maximum_curve_points: 4096,
        },
    )
    .unwrap();
    let plane = world.app().owner.active_sketch_edit().unwrap().plane;
    world.app().window.sketch_pixel(plane, paths[0][0]).unwrap()
}

#[then("Format Text opens containing b and its current formatting")]
#[then("Format Text contains b")]
fn original_draft(world: &mut ApplicationWorld) {
    let style = committed_text(world).style;
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.content, "b");
    assert_eq!(draft.family, style.family);
    assert_eq!(draft.height_mm.parse::<f64>().unwrap() / 10., style.height);
    assert_eq!(
        draft.rotation_degrees.parse::<f64>().unwrap().to_radians(),
        style.rotation
    );
}

#[when("the user focuses the editor, press Ctrl+A, and type c")]
fn replace_with_c(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
    x11_input::press_keysym(u32::from(b'c'));
    world.pump(Vec::new());
    assert_replacement_preview(world);
}

#[then("c replaces b in the editor and live canvas preview")]
fn assert_replacement_preview(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().content, "c");
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let text = committed_text(world);
    assert_eq!(text.style.content, "b");
    let Some(ViewportSketchPreview::Text {
        style, replaces, ..
    }) = world.app().owner.sketch_preview()
    else {
        panic!("expected a replacement text preview");
    };
    assert_eq!(style.content, "c");
    assert_eq!(replaces, Some((edit.sketch, text.id)));
}

#[when("the user focuses the editor, press Ctrl+A, type c, and click OK")]
fn accept_c(world: &mut ApplicationWorld) {
    replace_with_c(world);
    click_editor_button(world, "OK");
    assert_committed(world, "c");
}

#[then("the original b returns on the canvas")]
#[then("b returns without removing the text object")]
fn original_text_is_visible(world: &mut ApplicationWorld) {
    assert_committed(world, "b");
}

#[then("the sketch returns directly to Ready with its baseline solver count")]
#[then("the solver count remains at its pre-edit value")]
fn baseline_count(world: &mut ApplicationWorld) {
    assert_eq!(world.state().dimensions_needed, Some(2));
    assert!(world.state().sketch_at_ready);
}

#[then("no new text object is created")]
fn no_extra_text(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().texts().len(), 1);
}

#[then("c replaces b and the sketch returns directly to Ready")]
#[then("c returns at the same visible location")]
fn replacement_text_is_visible(world: &mut ApplicationWorld) {
    assert_committed(world, "c");
}

#[when("the user presses Ctrl+Z")]
pub(super) fn undo_edit(world: &mut ApplicationWorld) {
    x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
}

#[when("the user presses Ctrl+Y")]
pub(super) fn redo_edit(world: &mut ApplicationWorld) {
    x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
}

fn committed_text(world: &mut ApplicationWorld) -> SketchText2D {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.texts().len(), 1);
    geometry.texts()[0].clone()
}

fn assert_committed(world: &mut ApplicationWorld, content: &str) {
    let text = committed_text(world);
    assert_eq!(text.style.content, content);
    let plane = world.app().owner.active_sketch_edit().unwrap().plane;
    let anchor = world
        .app()
        .window
        .sketch_point(INSERT, plane)
        .unwrap()
        .unwrap();
    assert_eq!(
        world.edited_geometry().point(text.anchor).unwrap().position,
        anchor
    );
    assert!(world.state().sketch_at_ready);
    assert!(world.app().owner.sketch_preview().is_none());
    assert!(
        world
            .app()
            .window
            .native
            .sketch_display_declines()
            .unwrap()
            .is_empty()
    );
}
