// SPDX-License-Identifier: GPL-2.0-only
use super::*;

const BLUE: [u8; 3] = [0, 128, 255];

#[test]
fn native_text_palette_blue_reaches_preview_and_committed_glyphs() {
    let mut world = ApplicationWorld::new();
    text_edit::committed_b(&mut world);
    let original = world.edited_geometry();
    text_edit::open_existing(&mut world);
    assert!(blue_ink(&mut world) == 0);
    select_blue(&mut world);
    let draft = world.state().sketch_text.unwrap();
    let selected = draft.formatting.runs.as_deref().unwrap_or_default();
    assert_eq!(
        oblikovati_model::sketch::uniform_sketch_text_color(&draft.content, selected)
            .flatten()
            .map(|color| [color.r, color.g, color.b]),
        Some(BLUE),
        "expected the palette to change the text draft"
    );
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-text-blue-preview");
    }
    assert!(blue_ink(&mut world) > 0);
    click_editor_button(&mut world, "OK");
    assert!(blue_ink(&mut world) > 0);
    let edited = world.edited_geometry();
    let text = &edited.texts()[0];
    assert_eq!(text.id, original.texts()[0].id);
    assert_eq!(text.anchor, original.texts()[0].anchor);
    let run = &text.style.formatting_runs.as_deref().unwrap()[0];
    let color = run.color.as_deref().unwrap();
    assert_eq!([color.r, color.g, color.b], BLUE);
    world.close();
}

fn select_blue(world: &mut ApplicationWorld) {
    focus_content(world);
    x11_input::press_control_a();
    world.pump(Vec::new());
    click_editor_button(world, "Color");
    let swatch = blue_swatch(world);
    x11_input::click(swatch);
    world.pump(Vec::new());
    let confirm = world.run("OK").center();
    x11_input::click(diagnostics::pointer_pixels(confirm));
    world.pump(Vec::new());
    world.expect_no_text("Basic colors");
}

fn blue_swatch(world: &mut ApplicationWorld) -> [i32; 2] {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let [x, _] = diagnostics::window_geometry("Color")
        .unwrap()
        .0
        .map(|v| v as usize);
    let top = world.run("Basic colors").maximum[1] as usize;
    let bottom = world.run("Custom colors").minimum[1] as usize;
    let stride = capture.extent[0] as usize;
    for row in top..bottom {
        for column in x..x + 230 {
            let start = (row * stride + column) * 4;
            if capture.rgba[start..start + 3] == BLUE {
                return [column as i32 + 3, row as i32 + 3];
            }
        }
    }
    panic!("expected the blue basic swatch in the Color dialog");
}

fn blue_ink(world: &mut ApplicationWorld) -> usize {
    text_format_choices::glyph_pixels(world)
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| {
            pixel[2].saturating_sub(pixel[0]) > 70 && pixel[2].saturating_sub(pixel[1]) > 35
        })
        .count()
}
