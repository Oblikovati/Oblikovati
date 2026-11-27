// SPDX-License-Identifier: GPL-2.0-only
#![cfg(test)]

use super::text_line_advance;
use imgui::{Condition, MouseButton, Ui};

#[test]
fn double_spacing_moves_native_rows_and_preserves_glyph_size() {
    let mut context = editor_context();
    let mut content = "Z\nZ".to_owned();
    let single = render_rows(&mut context, 1., &mut content);
    let double = render_rows(&mut context, 2., &mut content);
    assert_eq!(single[0], double[0]);
    let gap = single[2] - single[0];
    assert!(gap > 0.);
    assert_eq!(double[2] - double[0], 2. * gap);
    assert_eq!(single[1] - single[0], double[1] - double[0]);
}

#[test]
fn spaced_row_hit_test_places_the_native_caret_on_the_second_line() {
    let mut context = editor_context();
    let mut content = "Z\nZ".to_owned();
    let [_, _, top, bottom] = render_rows(&mut context, 2., &mut content);
    context
        .io_mut()
        .add_mouse_pos_event([50., top.midpoint(bottom)]);
    render_rows(&mut context, 2., &mut content);
    context
        .io_mut()
        .add_mouse_button_event(MouseButton::Left, true);
    render_rows(&mut context, 2., &mut content);
    context
        .io_mut()
        .add_mouse_button_event(MouseButton::Left, false);
    render_rows(&mut context, 2., &mut content);
    context.io_mut().add_input_character('Q');
    render_rows(&mut context, 2., &mut content);
    assert_eq!(content, "Z\nZQ");
}

fn editor_context() -> crate::test_context::TestContext {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [450., 300.];
    context.fonts().build_rgba32_texture();
    context
}

fn render_rows(context: &mut imgui::Context, factor: f32, content: &mut String) -> [f32; 4] {
    let ui = context.frame();
    let mut corners = [[0.; 2]; 2];
    crate::widget_abi::window_geometry(ui, [10., 10.], [300., 200.], Condition::Always);
    ui.window("Spacing test")
        .build(|| corners = draw_input(ui, factor, content));
    let mut ys: Vec<_> = context
        .render()
        .draw_lists()
        .flat_map(|list| list.vtx_buffer())
        .filter(|vertex| corners.contains(&vertex.uv))
        .map(|vertex| vertex.pos[1])
        .collect();
    ys.sort_by(f32::total_cmp);
    ys.try_into()
        .expect("expected two glyph quads with two matching corners each")
}

fn draw_input(ui: &Ui, factor: f32, content: &mut String) -> [[f32; 2]; 2] {
    // The test window and built atlas keep the current native font and glyph live.
    let glyph = unsafe { &*imgui::sys::ImFont_FindGlyph(imgui::sys::igGetFont(), 'Z' as u32) };
    let corners = [[glyph.U0, glyph.V0], [glyph.U1, glyph.V1]];
    let normal = ui.current_font_size();
    {
        let _advance = text_line_advance(ui, factor);
        ui.input_text_multiline("##rows", content, [200., 100.])
            .build();
    }
    assert_eq!(normal, ui.current_font_size());
    corners
}
