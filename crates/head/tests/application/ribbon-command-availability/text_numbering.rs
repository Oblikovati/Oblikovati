// SPDX-License-Identifier: GPL-2.0-only
//! Drive numbering and symbol insertion through the native Format Text editor.
use super::*;
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_text};
use text_format_choices::{choose_font, glyph_pixels};

const PARAGRAPH_FIXTURE_FREEDOMS: usize = 40;
const PARAGRAPH_INSERTION: [i32; 2] = [900, 580];
const NUMBERED_PARAGRAPHS: &str = "  1. a\n  2. b";

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
    if !world.shows("Numeric") {
        open_numbering(world);
    }
    click_named_row(world, "Numeric");
    assert!(
        glyph_pixels(world) != before,
        "the numbering preview must change the glyph pixels"
    );
}

#[given("the restored No Preset editor is empty")]
fn open_empty_paragraph_editor(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(PARAGRAPH_FIXTURE_FREEDOMS);
    activate_text(world);
    open_editor(world, PARAGRAPH_INSERTION);
    world.expect_text("No Preset");
}

#[when("the user focuses the editor and press a, Return, and b")]
fn enter_two_real_paragraphs(world: &mut ApplicationWorld) {
    type_content(world, b'a');
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    expect_preview(world, "a\nb");
}

#[when("the user selects both paragraphs and click Bullets")]
fn bullet_both_paragraphs(world: &mut ApplicationWorld) {
    x11_input::press_control_a();
    world.pump(Vec::new());
    let before = glyph_pixels(world);
    click_named_row(world, "Bullets");
    assert_ne!(
        glyph_pixels(world),
        before,
        "bullet outlines must change the preview"
    );
}

#[then("each paragraph has a bullet marker")]
fn expect_bullets_in_both_rows(world: &mut ApplicationWorld) {
    expect_preview(world, "  • a\n  • b");
    world.expect_text("• a");
    world.expect_text("• b");
    expect_symbol_outline(world, "•");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("bulleted-paragraphs");
    }
}

#[then("the editor shows 1. for a and 2. for b")]
fn expect_sequential_numbering(world: &mut ApplicationWorld) {
    expect_preview(world, NUMBERED_PARAGRAPHS);
    world.expect_text("1. a");
    world.expect_text("2. b");
}

#[then("the numbered Text remains and Ready shows 42 dimensions needed")]
fn expect_numbered_text_at_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(world.dimensions_needed(), 42);
    world.expect_dimensions_needed(42);
    let geometry = world.edited_geometry();
    let [text] = geometry.texts() else {
        panic!("expected a single numbered Text entity at Ready");
    };
    assert_eq!(text.style.content, NUMBERED_PARAGRAPHS);
    assert_eq!(
        geometry.point(text.anchor).unwrap().position,
        world.sketch_point_of(PARAGRAPH_INSERTION)
    );
    assert!(world.app().owner.sketch_preview().is_none());
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("numbered-paragraphs-at-ready");
    }
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
    expect_symbol_outline(world, "⌀");
    assert!(world.edited_geometry().texts().is_empty());
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("numbered-diameter-preview");
    }
}

fn expect_symbol_outline(world: &mut ApplicationWorld, symbol: &str) {
    let Some(ViewportSketchPreview::Text { mut style, .. }) = world.app().owner.sketch_preview()
    else {
        panic!("expected a Text preview containing {symbol:?}");
    };
    style.content = symbol.into();
    let options = SketchSamplingOptionsV1 {
        working_scale: 0.1,
        deflection: 0.01,
        maximum_curve_points: 4096,
    };
    let edges = sample_sketch_text(&style, [0.; 2], options).unwrap();
    assert!(
        edges.iter().any(|edge| edge.len() > 2),
        "symbol {symbol:?} must have curved edges: found {edges:?}"
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
