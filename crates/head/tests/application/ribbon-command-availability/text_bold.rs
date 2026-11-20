// SPDX-License-Identifier: GPL-2.0-only
//! Check the published Bold preview and Cancel through X11 input.
use super::*;

#[given("unbolded c text and Ready sketch state")]
fn prepare_unbolded_text(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, "DejaVu Sans");
    type_content(world, b'c');
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    world.expect_ready();
    let sketch = world.edited_geometry();
    assert_eq!(sketch.texts()[0].style.content, "c");
    assert!(sketch.texts()[0].style.formatting_runs.is_none());
    world.text_creation_baseline = Some(sketch);
    world.observed_counts = vec![world.dimensions_needed()];
}

#[when("the user chooses Edit text, focus the editor, and press Ctrl+A")]
fn select_existing_text(world: &mut ApplicationWorld) {
    text_edit::open_existing(world);
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
}

#[when("the user clicks the Bold control")]
fn click_bold(world: &mut ApplicationWorld) {
    let plain_button = world.pixel_left_of("Bold");
    let regular_editor = editor_ink(world);
    let regular_canvas = text_format_choices::glyph_pixels(world);
    click_editor_button(world, "Bold");
    x11_input::move_pointer([480, 540]);
    world.pump(Vec::new());
    assert_ne!(world.pixel_left_of("Bold"), plain_button);
    assert!(
        editor_ink(world) > regular_editor,
        "expected more foreground glyph pixels in Bold"
    );
    assert_ne!(text_format_choices::glyph_pixels(world), regular_canvas);
}

#[then("Bold is highlighted and c thickens in the editor and live preview")]
fn expect_bold_preview(world: &mut ApplicationWorld) {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the existing Text preview");
    };
    let runs = style
        .formatting_runs
        .as_ref()
        .expect("expected one Bold run");
    assert_eq!(runs.len(), 1);
    assert!(runs[0].bold);
    assert_eq!((runs[0].start, runs[0].end), (0, 1));
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("bold-text-preview");
    }
}

#[then("the character content remains c")]
fn expect_editor_content(world: &mut ApplicationWorld) {
    assert_eq!(world.state().sketch_text.unwrap().content, "c");
}

#[then("the original thinner c returns and the sketch is Ready")]
fn expect_cancelled_preview(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert!(world.app().owner.sketch_preview().is_none());
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
}

#[then("the baseline solver count is retained")]
fn expect_solver_count(world: &mut ApplicationWorld) {
    assert_eq!(world.dimensions_needed(), world.observed_counts[0]);
}

fn editor_ink(world: &mut ApplicationWorld) -> usize {
    let run = world.run("c").clone();
    let [left, top] = diagnostics::pointer_pixels(run.minimum);
    let [right, bottom] = diagnostics::pointer_pixels(run.maximum);
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let stride = capture.extent[0] as usize * 4;
    capture
        .rgba
        .chunks_exact(stride)
        .skip(top as usize)
        .take((bottom - top) as usize)
        .flat_map(|row| row[left as usize * 4..right as usize * 4].chunks_exact(4))
        .filter(|color| color[..3].iter().all(|value| *value > 140))
        .count()
}
