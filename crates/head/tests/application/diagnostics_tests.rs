// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use imgui::Condition;
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
