// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn probe_context() -> imgui::Context {
    let mut context = imgui::Context::create();
    context.set_ini_filename(None);
    context.set_log_filename(None);
    context.io_mut().display_size = [640., 480.];
    context.io_mut().delta_time = 1. / 60.;
    context.fonts().build_rgba32_texture();
    context
}
#[test]
fn scalar_window_payload_retains_every_condition_and_fractional_coordinate() {
    let mut context = probe_context();
    for condition in [
        Condition::Always,
        Condition::Once,
        Condition::FirstUseEver,
        Condition::Appearing,
    ] {
        assert_window_payload(&mut context, condition);
    }
}
fn assert_window_payload(context: &mut imgui::Context, condition: Condition) {
    let ui = context.frame();
    window_geometry(ui, [117.25, 229.5], [363.75, 171.25], condition);
    // This owner holds the sole live context; read the record before Begin consumes it.
    let current = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.unwrap();
    let position = current.NextWindowData.PosVal;
    let size = current.NextWindowData.SizeVal;
    assert_eq!([position.x, position.y], [117.25, 229.5]);
    assert_eq!([size.x, size.y], [363.75, 171.25]);
    assert_eq!(current.NextWindowData.PosCond, condition as i32);
    assert_eq!(current.NextWindowData.SizeCond, condition as i32);
    context.render();
}
#[test]
fn never_condition_does_not_publish_next_window_settings() {
    let mut context = probe_context();
    let ui = context.frame();
    window_geometry(ui, [1., 2.], [200., 100.], Condition::Never);
    // The active owner retains the context through Render; no native pointers escape this test.
    let current = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.unwrap();
    assert_eq!(current.NextWindowData.Flags, 0);
    context.render();
}
#[test]
fn scalar_widgets_retain_cursor_dimensions_and_literal_draw_color() {
    let mut context = probe_context();
    let ui = context.frame();
    window_geometry(ui, [0., 0.], [400., 300.], Condition::Always);
    ui.window("Scalar widget probe").build(|| {
        cursor_position(ui, [31., 47.]);
        assert_eq!(ui.cursor_pos(), [31., 47.]);
        assert!(!button(ui, "Button", [101., 33.]));
        assert_eq!(ui.item_rect_size(), [101., 33.]);
        assert!(!selectable(ui, "Row", false, true));
        literal_text(ui, [80., 100.], [0.2, 0.4, 0.6, 1.], "Part##literal");
    });
    assert_literal_draw_color(&mut context);
}
fn assert_literal_draw_color(context: &mut imgui::Context) {
    let draws = context.render();
    let color = ImColor32::from_rgba_f32s(0.2, 0.4, 0.6, 1.)
        .to_bits()
        .to_ne_bytes();
    assert!(
        draws
            .draw_lists()
            .flat_map(|list| list.vtx_buffer())
            .any(|vertex| vertex.col == color)
    );
}
