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

#[test]
fn scalar_background_text_keeps_literal_glyphs_behind_window_geometry() {
    let mut context = probe_context();
    let ui = context.frame();
    let text = "d0 = 10 mm##literal";
    background_literal_text(ui, [80.25, 100.5], [1., 0., 0., 1.], text);
    window_geometry(ui, [0., 0.], [400., 300.], Condition::Always);
    ui.window("Annotation layer probe").build(|| {
        literal_text(ui, [80., 100.], [0., 0., 1., 1.], "Window");
    });
    assert_background_draws(&mut context, text);
}
fn assert_background_draws(context: &mut imgui::Context, text: &str) {
    let draws = context.render();
    let lists: Vec<&imgui::DrawList> = draws.draw_lists().collect();
    let red = ImColor32::from_rgba(255, 0, 0, 255).to_bits().to_ne_bytes();
    let vertices = lists[0].vtx_buffer();
    assert_eq!(
        vertices.len(),
        text.chars().filter(|value| !value.is_whitespace()).count() * 4
    );
    assert!(vertices.iter().all(|vertex| vertex.col == red));
    assert!(
        lists.len() > 1,
        "expected window geometry after the background annotation"
    );
    assert!(
        lists[1..]
            .iter()
            .flat_map(|list| list.vtx_buffer())
            .all(|vertex| vertex.col != red)
    );
}

#[test]
fn fitted_dialog_retains_valid_user_placement_and_refits_after_viewport_resize() {
    let mut context = probe_context();
    assert_fitted_dialog(&mut context, [16., 168.], [608., 160.]);
    let ui = context.frame();
    window_geometry(ui, [37., 51.], [350., 200.], Condition::Always);
    ui.window("Fitted").build(|| {});
    context.render();
    assert_fitted_dialog(&mut context, [37., 51.], [350., 200.]);
    context.io_mut().display_size = [200., 120.];
    assert_fitted_dialog(&mut context, [16., 16.], [168., 88.]);
}
fn assert_fitted_dialog(context: &mut imgui::Context, position: [f32; 2], size: [f32; 2]) {
    let ui = context.frame();
    fit_dialog(ui, "Fitted", [320., 168.], [620., 160.]);
    ui.window("Fitted").build(|| {
        assert_eq!(ui.window_pos(), position);
        assert_eq!(ui.window_size(), size);
    });
    context.render();
}

#[test]
fn collapsed_dialog_refits_its_full_size_before_it_is_expanded() {
    let mut context = probe_context();
    let ui = context.frame();
    window_geometry(ui, [37., 51.], [350., 200.], Condition::Always);
    ui.window("Collapsed")
        .collapsed(true, Condition::Always)
        .build(|| {});
    context.render();
    context.io_mut().display_size = [200., 120.];
    let ui = context.frame();
    fit_dialog(ui, "Collapsed", [320., 168.], [620., 160.]);
    ui.window("Collapsed")
        .collapsed(false, Condition::Always)
        .build(|| {
            assert_eq!(ui.window_pos(), [16., 16.]);
            assert_eq!(ui.window_size(), [168., 88.]);
        });
    context.render();
}

#[test]
fn decorated_buttons_keep_the_normal_button_identity_and_hit_bounds() {
    let mut context = probe_context();
    let plain = button_item_probe(&mut context, false, true);
    let decorated = button_item_probe(&mut context, true, true);
    let icon_only = button_item_probe(&mut context, true, false);
    assert_eq!(icon_only, plain);
    assert_eq!(decorated, plain);
    assert_eq!(decorated.0, [108., 58.]);
}
fn button_item_probe(
    context: &mut imgui::Context,
    decorated: bool,
    show_caption: bool,
) -> ([f32; 2], u32) {
    let ui = context.frame();
    window_geometry(ui, [0., 0.], [400., 300.], Condition::Always);
    let mut result = ([0.; 2], 0);
    ui.window("Icon identity").build(|| {
        cursor_position(ui, [31., 47.]);
        draw_probe_button(ui, decorated, show_caption);
        // The sole live context owns LastItemData; no pointer survives this observation.
        let identity = unsafe { imgui::sys::igGetCurrentContext().as_ref() }
            .unwrap()
            .LastItemData
            .ID;
        result = (ui.item_rect_size(), identity);
    });
    context.render();
    result
}

#[test]
fn icon_button_captions_are_opaque_and_inside_the_clickable_cell() {
    for large in [true, false] {
        check_icon_caption(large);
    }
}
fn check_icon_caption(large: bool) {
    let mut context = probe_context();
    draw_colored_caption(&mut context, large);
    context.render();
    draw_colored_caption(&mut context, large);
    let ink = colored_caption_points(context.render());
    assert_eq!(
        ink.len(),
        "Rectangle".len() * 4,
        "expected every opaque caption glyph"
    );
    assert!(
        ink.iter()
            .all(|position| position[0] >= 31. && position[0] <= 139.)
    );
    let (low, high) = if large { (89., 105.) } else { (47., 65.) };
    assert!(
        ink.iter()
            .all(|position| position[1] >= low && position[1] <= high)
    );
}
fn colored_caption_points(draws: &imgui::DrawData) -> Vec<[f32; 2]> {
    let color = ImColor32::from_rgba(51, 102, 153, 255)
        .to_bits()
        .to_ne_bytes();
    draws
        .draw_lists()
        .flat_map(|list| list.vtx_buffer())
        .filter(|vertex| vertex.col == color)
        .map(|vertex| vertex.pos)
        .collect()
}
fn draw_probe_button(ui: &Ui, decorated: bool, show_caption: bool) {
    if decorated {
        let mut presentation = caption_presentation(true);
        presentation.show_caption = show_caption;
        icon_button(ui, "Rectangle", presentation);
    } else {
        button(ui, "Rectangle", [108., 58.]);
    }
}
fn caption_presentation(large: bool) -> IconButtonPresentation {
    IconButtonPresentation {
        texture: crate::GuiTextureKey::from_raw(2).unwrap(),
        size: if large { [108., 58.] } else { [108., 18.] },
        icon: if large { [40.; 2] } else { [18.; 2] },
        large,
        show_caption: true,
    }
}
fn draw_colored_caption(context: &mut imgui::Context, large: bool) {
    let ui = context.frame();
    window_geometry(ui, [0., 0.], [400., 300.], Condition::Always);
    ui.window("Caption visibility").build(|| {
        cursor_position(ui, [31., 47.]);
        let _color = ui.push_style_color(imgui::StyleColor::Text, [0.2, 0.4, 0.6, 1.]);
        icon_button(ui, "Rectangle", caption_presentation(large));
    });
}

#[test]
fn caption_free_icon_keeps_its_image_without_text() {
    for large in [true, false] {
        let mut context = probe_context();
        for _ in 0..2 {
            let ui = context.frame();
            window_geometry(ui, [0., 0.], [400., 300.], Condition::Always);
            ui.window("Icon only").build(|| {
                cursor_position(ui, [31., 47.]);
                let _color = ui.push_style_color(imgui::StyleColor::Text, [0.2, 0.4, 0.6, 1.]);
                let mut presentation = caption_presentation(large);
                presentation.show_caption = false;
                icon_button(ui, "Rectangle", presentation);
            });
            let draws = context.render();
            assert!(colored_caption_points(draws).is_empty());
        }
        let draws = context.render();
        assert!(draws.draw_lists().flat_map(|list| list.commands()).any(|command| {
            matches!(command, imgui::DrawCmd::Elements { cmd_params, .. } if cmd_params.texture_id.id() == 2)
        }), "expected the icon texture to remain in the draw data");
    }
}
