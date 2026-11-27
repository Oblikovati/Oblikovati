// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published multiline formatting scenario with X11 keyboard and pointer input.
use super::*;
use oblikovati_api::types::{TEXT_ALIGN_CENTER, TEXT_ALIGN_MIDDLE, TEXT_LINE_SPACING_DOUBLE};

const FIRST: &str = "ooo";
const SECOND: &str = "oooo";

#[given("the existing Text content is selected")]
fn existing_multiline_text_is_selected(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    type_content(world, b'o');
    enter_keys(world, [b'o' as u32, b'o' as u32, 0xff0d]);
    enter_keys(world, [b'o' as u32; 4]);
    click_editor_button(world, "OK");
    enter_keys(world, [0xff1b]);
    text_edit::open_existing(world);
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
    assert_eq!(world.state().sketch_text.unwrap().content, "ooo\noooo");
}

fn enter_keys(world: &mut ApplicationWorld, symbols: impl IntoIterator<Item = u32>) {
    for symbol in symbols {
        x11_input::press_keysym(symbol);
        world.pump(Vec::new());
    }
}

#[when("the user selects center horizontal and middle vertical alignment")]
fn select_center_and_middle(world: &mut ApplicationWorld) {
    let original = text_format_choices::glyph_pixels(world);
    let original_fill = [world.pixel_left_of("Left"), world.pixel_left_of("Baseline")];
    click_row(world, "Left");
    click_row(world, "Center");
    click_row(world, "Baseline");
    click_row(world, "Middle");
    x11_input::move_pointer([475, 540]);
    world.pump(Vec::new());
    assert_ne!(text_format_choices::glyph_pixels(world), original);
    assert_ne!(world.pixel_left_of("Center"), original_fill[0]);
    assert_ne!(world.pixel_left_of("Middle"), original_fill[1]);
}

#[then("the selected controls are highlighted and canvas placement changes")]
fn preview_is_centered(world: &mut ApplicationWorld) {
    world.run_in_window("Format Text", "Center");
    world.run_in_window("Format Text", "Middle");
    let style = preview_style(world);
    assert_eq!(style.h_align, TEXT_ALIGN_CENTER);
    assert_eq!(style.v_align, TEXT_ALIGN_MIDDLE);
}

#[when("the user opens line spacing")]
fn open_line_spacing(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(editor_row_gap(world));
    click_row(world, "Single");
}

#[then("Single is checked and Double, 1.5 Lines, Multiple and Exactly are offered")]
fn expect_line_spacing_choices(world: &mut ApplicationWorld) {
    let list = world.run("Double").list;
    for name in ["Single", "1.5 Lines", "Multiple", "Exactly"] {
        assert_eq!(
            world.run(name).list,
            list,
            "expected {name} in the line-spacing menu"
        );
    }
    assert!(world.checked("Single"));
}

#[when("the user chooses Double")]
fn choose_double_spacing(world: &mut ApplicationWorld) {
    click_row(world, "Double");
}

#[then("the displayed editor rows have increased separation")]
fn verify_double_spacing(world: &mut ApplicationWorld) {
    let before = world.observed_row_gap.unwrap();
    assert!(before > 0.);
    assert_eq!(editor_row_gap(world), 2. * before);
    assert_eq!(
        preview_style(world).line_spacing.unwrap().mode,
        TEXT_LINE_SPACING_DOUBLE
    );
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("double-line-spacing");
    }
}

fn editor_row_gap(world: &ApplicationWorld) -> f32 {
    world.run(SECOND).center()[1] - world.run(FIRST).center()[1]
}

fn click_row(world: &mut ApplicationWorld, name: &str) {
    x11_input::click(diagnostics::pointer_pixels(world.run(name).center()));
    world.pump(Vec::new());
}

fn preview_style(world: &mut ApplicationWorld) -> oblikovati_api::types::SketchTextStyle {
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the native Text preview");
    };
    *style
}
