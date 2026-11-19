// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use imgui::Condition;
#[test]
fn quarter_turned_glyph_keeps_its_letter_and_visible_bounds() {
    let vertices = [
        DrawVert {
            pos: [20., 10.],
            uv: [0.1, 0.2],
            col: [255; 4],
        },
        DrawVert {
            pos: [20., 20.],
            uv: [0.2, 0.2],
            col: [255; 4],
        },
        DrawVert {
            pos: [10., 20.],
            uv: [0.2, 0.3],
            col: [255; 4],
        },
        DrawVert {
            pos: [10., 10.],
            uv: [0.1, 0.3],
            col: [255; 4],
        },
    ];
    let lookup = BTreeMap::from([([0.1_f32, 0.2, 0.2, 0.3].map(f32::to_bits), 'F')]);
    let glyph = quad_glyph(0, &vertices, [0., 0., 30., 30.], &lookup).unwrap();
    assert_eq!(
        (glyph.letter, glyph.minimum, glyph.maximum),
        ('F', [10., 10.], [20., 20.])
    );
    assert!(quad_glyph(0, &vertices, [15., 0., 30., 30.], &lookup).is_none());
}
#[test]
fn clip_accepts_small_glyph_overhang_but_rejects_hidden_letters() {
    let vertex = |pos| DrawVert {
        pos,
        uv: [0., 0.],
        col: [255; 4],
    };
    let a = vertex([10., 10.]);
    let b = vertex([20., 20.]);
    assert!(inside_clip(a, b, [11., 10., 30., 30.]));
    assert!(!inside_clip(a, b, [14., 10., 30., 30.]));
    assert!(!inside_clip(a, b, [21., 10., 30., 30.]));
}
#[test]
fn clipped_glyph_uv_keeps_the_matching_letter_only_when_most_is_visible() {
    let atlas = [0.1_f32, 0.2, 0.2, 0.3];
    let lookup = BTreeMap::from([(atlas.map(f32::to_bits), 'p')]);
    assert_eq!(glyph_for_uv([0.1, 0.2, 0.2, 0.29], &lookup), Some('p'));
    assert_eq!(glyph_for_uv([0.1, 0.2, 0.2, 0.25], &lookup), None);
    assert_eq!(glyph_for_uv([0.3, 0.2, 0.4, 0.3], &lookup), None);
}
#[test]
fn pointer_pixels_follow_framebuffer_scale_on_each_axis() {
    let mut context = crate::test_context::create();
    context.io_mut().display_framebuffer_scale = [1.5, 2.];
    assert_eq!(pointer_pixels([20., 30.]), [30, 60]);
}
/// Draw one headless frame with the named widgets; returns runs from its draw data.
fn runs_of(build: impl FnOnce(&imgui::Ui), scale: f32) -> Vec<RenderedRun> {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.fonts().build_rgba32_texture();
    let io = context.io_mut();
    io.display_size = [400., 300.];
    io.font_global_scale = scale;
    io.delta_time = 0.016;
    let ui = context.frame();
    ui.window("probe")
        .position([10., 10.], Condition::Always)
        .size([300., 200.], Condition::Always)
        .build(|| build(ui));
    let lookup = glyph_lookup();
    let font_size = current_font_size();
    frame_of_draw_data(context.render(), &lookup, font_size).runs
}
fn has(runs: &[RenderedRun], text: &str) -> bool {
    runs.iter().any(|run| run.text.contains(text))
}
#[test]
fn visible_text_survives_tree_arrows_between_rows() {
    let runs = runs_of(
        |ui| {
            ui.text("First");
            if let Some(_node) = ui.tree_node_config("Branch").default_open(true).push() {
                ui.bullet_text("Leaf");
            }
            ui.text("Last");
        },
        1.,
    );
    for text in ["First", "Branch", "Leaf", "Last"] {
        assert!(has(&runs, text), "{text}: {runs:?}");
    }
}
#[test]
fn text_of_a_skipped_widget_is_absent() {
    let runs = runs_of(
        |ui| {
            if false {
                ui.text("Hidden");
            }
            ui.text("Shown");
        },
        1.,
    );
    assert!(has(&runs, "Shown") && !has(&runs, "Hidden"), "{runs:?}");
}
#[test]
fn text_outside_its_clip_rectangle_is_absent() {
    let runs = runs_of(
        |ui| {
            ui.set_cursor_pos([0., 400.]);
            ui.text("Below");
            ui.set_cursor_pos([20., 40.]);
            ui.text("Inside");
        },
        1.,
    );
    assert!(has(&runs, "Inside") && !has(&runs, "Below"), "{runs:?}");
}
#[test]
fn scaled_text_keeps_its_letters_and_grows() {
    let width = |scale| {
        let runs = runs_of(|ui| ui.text("Scale"), scale);
        let run = runs
            .iter()
            .find(|run| run.text == "Scale")
            .cloned()
            .unwrap();
        run.maximum[0] - run.minimum[0]
    };
    assert!(width(2.) > width(1.) * 1.5);
}
#[test]
fn separate_buttons_are_separate_runs() {
    let runs = runs_of(
        |ui| {
            ui.button("Save");
            ui.same_line();
            ui.button("Cancel");
        },
        1.,
    );
    assert!(runs.iter().any(|run| run.text == "Save"), "{runs:?}");
    assert!(runs.iter().any(|run| run.text == "Cancel"), "{runs:?}");
}
#[test]
fn a_checked_box_has_a_mark_and_a_cleared_one_has_none() {
    let marks = |mut value: bool| {
        let mut context = crate::test_context::create();
        context.set_ini_filename(None);
        context.fonts().build_rgba32_texture();
        context.io_mut().display_size = [400., 300.];
        context.io_mut().delta_time = 0.016;
        let ui = context.frame();
        let window = ui.window("probe");
        let window = window
            .position([0., 0.], Condition::Always)
            .size([200., 100.], Condition::Always);
        window.build(|| {
            ui.checkbox("Option", &mut value);
        });
        let lookup = glyph_lookup();
        let font_size = current_font_size();
        frame_of_draw_data(context.render(), &lookup, font_size)
            .marks
            .len()
    };
    assert!(marks(true) > marks(false));
}

#[test]
fn window_occlusions_identifies_real_ui_rectangles() {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.fonts().build_rgba32_texture();
    context.io_mut().display_size = [400., 300.];
    context.io_mut().delta_time = 0.016;
    context
        .frame()
        .window("occluding panel")
        .position([10., 10.], Condition::Always)
        .size([120., 80.], Condition::Always)
        .build(|| {});
    context.render();
    assert!(window_occlusions().contains(&[10., 10., 130., 90.]));
    assert_eq!(
        window_pixel_bounds("occluding panel"),
        Some([10., 10., 130., 90.])
    );
    context.io_mut().display_framebuffer_scale = [2., 2.];
    assert_eq!(
        window_pixel_bounds("occluding panel"),
        Some([20., 20., 260., 180.])
    );
    assert!(window_pixel_bounds("absent panel").is_none());
}

#[test]
fn dock_header_bounds_exclude_only_the_scaled_strip() {
    let node = sys::ImGuiDockNode {
        Pos: sys::ImVec2 { x: 0., y: 158. },
        Size: sys::ImVec2 { x: 360., y: 720. },
        ..Default::default()
    };
    let scale = sys::ImVec2 { x: 2., y: 1.5 };
    assert_eq!(
        dock_header_bounds(&node, 177., scale),
        Some([0., 237., 720., 265.5])
    );
    assert_eq!(dock_header_bounds(&node, 158., scale), None);
}

#[test]
fn dock_header_is_not_solid_but_does_not_hide_model_pixels() {
    let node = sys::ImGuiDockNode {
        Pos: sys::ImVec2 { x: 0., y: 0. },
        Size: sys::ImVec2 { x: 16., y: 32. },
        ..Default::default()
    };
    let scale = sys::ImVec2 { x: 1., y: 1. };
    let header = dock_header_bounds(&node, 8., scale).unwrap();
    let mut image = blank_visibility_capture();
    paint_visibility_patch(&mut image, [2, 1]);
    assert!(!super::super::solid_visibility::visible_solid_fill(
        &image,
        &[header]
    ));
    paint_visibility_patch(&mut image, [20, 20]);
    assert!(super::super::solid_visibility::visible_solid_fill(
        &image,
        &[header]
    ));
}

fn blank_visibility_capture() -> oblikovati_viewport::ViewportCapture {
    oblikovati_viewport::ViewportCapture {
        view_id: 0,
        scene_generation: 0,
        camera_generation: 0,
        surface_generation: 0,
        overlay_generation: 0,
        extent: [32, 32],
        rgba: vec![40; 32 * 32 * 4],
    }
}

fn paint_visibility_patch(image: &mut oblikovati_viewport::ViewportCapture, at: [usize; 2]) {
    for y in at[1]..at[1] + 5 {
        for x in at[0]..at[0] + 5 {
            let offset = (y * 32 + x) * 4;
            image.rgba[offset..offset + 4].copy_from_slice(&[63, 153, 255, 255]);
        }
    }
}

fn checkbox_frame(scale: f32, mut left: bool, mut right: bool) -> RenderedFrame {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.fonts().build_rgba32_texture();
    context.io_mut().display_size = [800., 400.];
    context.io_mut().font_global_scale = scale;
    context.io_mut().delta_time = 0.016;
    let ui = context.frame();
    ui.window("probe")
        .position([0., 0.], Condition::Always)
        .size([750., 300.], Condition::Always)
        .build(|| {
            ui.checkbox("Left", &mut left);
            ui.same_line();
            ui.checkbox("Right", &mut right);
        });
    let lookup = glyph_lookup();
    let font_size = current_font_size();
    frame_of_draw_data(context.render(), &lookup, font_size)
}
#[test]
fn checkbox_marks_belong_to_their_own_label_at_multiple_scales() {
    for scale in [1.0, 1.5, 2.0] {
        for (left, right) in [(false, false), (false, true), (true, false), (true, true)] {
            let frame = checkbox_frame(scale, left, right);
            for (label, expected) in [("Left", left), ("Right", right)] {
                let run = frame.runs.iter().find(|run| run.text == label).unwrap();
                assert_eq!(
                    run_is_checked(&frame, run),
                    expected,
                    "scale {scale}, label {label}, marks {:?}",
                    frame.marks
                );
            }
        }
    }
}

fn popup_frame(scale: f32, mut left: bool, mut right: bool, selected: bool) -> RenderedFrame {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.fonts().build_rgba32_texture();
    context.io_mut().display_size = [800., 400.];
    context.io_mut().font_global_scale = scale;
    context.io_mut().delta_time = 0.016;
    let mut frame = RenderedFrame::default();
    for pass in 0..2 {
        let ui = context.frame();
        ui.window("probe")
            .position([0., 0.], Condition::Always)
            .size([750., 300.], Condition::Always)
            .build(|| {
                if pass == 0 {
                    ui.open_popup("settings");
                }
                if let Some(_popup) = ui.begin_popup("settings") {
                    ui.checkbox("PopupLeft", &mut left);
                    ui.same_line();
                    ui.checkbox("PopupRight", &mut right);
                    ui.menu_item_config("MenuOption").selected(selected).build();
                }
            });
        let lookup = glyph_lookup();
        let font_size = current_font_size();
        frame = frame_of_draw_data(context.render(), &lookup, font_size);
    }
    frame
}
#[test]
fn popup_checkbox_columns_and_menu_marks_stay_distinct_at_multiple_scales() {
    for scale in [1.0, 1.5, 2.0] {
        for (left, right, selected) in [
            (false, true, false),
            (true, false, true),
            (false, false, true),
        ] {
            let frame = popup_frame(scale, left, right, selected);
            for (label, expected) in [
                ("PopupLeft", left),
                ("PopupRight", right),
                ("MenuOption", selected),
            ] {
                let run = frame
                    .runs
                    .iter()
                    .find(|run| run.text == label)
                    .unwrap_or_else(|| panic!("missing {label}: {:?}", frame.runs));
                assert_eq!(
                    run_is_checked(&frame, run),
                    expected,
                    "scale {scale}, label {label}, runs {:?}",
                    frame.runs
                );
            }
        }
    }
}
