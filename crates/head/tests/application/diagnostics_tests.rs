// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use imgui::{Condition, Context};
use std::sync::Mutex;
// Dear ImGui allows one active context per process.
static HEADLESS_CONTEXT: Mutex<()> = Mutex::new(());
/// Draw one headless frame with the named widgets; returns runs from its draw data.
fn runs_of(build: impl FnOnce(&imgui::Ui), scale: f32) -> Vec<RenderedRun> {
    let _exclusive = HEADLESS_CONTEXT
        .lock()
        .unwrap_or_else(|poison| poison.into_inner());
    let mut context = Context::create();
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
    runs_of_draw_data(context.render(), &lookup, font_size)
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
