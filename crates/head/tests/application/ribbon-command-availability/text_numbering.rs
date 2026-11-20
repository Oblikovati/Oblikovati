// SPDX-License-Identifier: GPL-2.0-only
//! Drive numbering and symbol insertion through the native Format Text editor.
use super::*;
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_text};
use text_format_choices::{choose_font, glyph_pixels};

#[given("the selected Text preview")]
fn prepare_selected_preview(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    // The fixture selects a font with U+2300. The default font fallback is Martensite #575.
    choose_font(world, "DejaVu Sans");
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    x11_input::press_control_a();
    world.pump(Vec::new());
    expect_preview(world, "ab");
}

#[when("the user opens the numbering dropdown")]
fn open_numbering(world: &mut ApplicationWorld) {
    click_named_row(world, "Numbering");
}

#[then("numeric, lower and upper alphabetic, and lower and upper Roman styles are offered")]
fn check_numbering_styles(world: &mut ApplicationWorld) {
    let list = world.run("Numeric").list;
    for label in [
        "Lower alphabetic",
        "Upper alphabetic",
        "Lower Roman",
        "Upper Roman",
    ] {
        assert_eq!(world.run(label).list, list);
    }
}

#[when("the user chooses numeric numbering")]
fn choose_numeric_numbering(world: &mut ApplicationWorld) {
    let before = glyph_pixels(world);
    click_named_row(world, "Numeric");
    assert!(
        glyph_pixels(world) != before,
        "the numbering preview must change the glyph pixels"
    );
}

#[then("a 1. prefix and indentation appear for the content")]
fn numbered_content_is_visible(world: &mut ApplicationWorld) {
    expect_preview(world, "  1. ab");
    world.expect_text("1. ab");
}

#[when("the user positions the caret at the end and insert diameter from the symbol palette")]
fn insert_diameter_at_the_end(world: &mut ApplicationWorld) {
    let before = glyph_pixels(world);
    x11_input::press_keysym(0xff57);
    world.pump(Vec::new());
    click_named_row(world, "Symbols");
    click_named_row(world, "Diameter");
    assert!(
        glyph_pixels(world) != before,
        "the diameter preview must change the glyph pixels"
    );
}

#[then("the diameter symbol appears in the editor and preview")]
fn numbered_diameter_is_visible(world: &mut ApplicationWorld) {
    expect_preview(world, "  1. ab⌀");
    world.expect_text("ab⌀");
    expect_diameter_outline(world);
    assert!(world.edited_geometry().texts().is_empty());
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("numbered-diameter-preview");
    }
}

fn expect_diameter_outline(world: &mut ApplicationWorld) {
    let Some(ViewportSketchPreview::Text { mut style, .. }) = world.app().owner.sketch_preview()
    else {
        panic!("expected the numbered Text preview");
    };
    style.content = "⌀".into();
    let options = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        deflection: 0.01,
        maximum_curve_points: 4096,
    };
    let edges = sample_sketch_text(&style, [0.; 2], options).unwrap();
    assert!(
        edges.iter().any(|edge| edge.len() > 2),
        "diameter must have curved edges: found {edges:?}"
    );
}

fn click_named_row(world: &mut ApplicationWorld, name: &str) {
    let at = world.run(name).center();
    x11_input::click(diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
}

fn expect_preview(world: &mut ApplicationWorld, expected: &str) {
    assert_eq!(world.state().sketch_text.unwrap().content, expected);
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected a Text preview with content {expected:?}");
    };
    assert_eq!(style.content, expected);
}
