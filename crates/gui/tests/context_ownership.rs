// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::GuiFrameEngine;
use std::process::{Command, Output};
const PROBE_ENV: &str = "OBLIKOVATI_GUI_ACTIVE_FRAME_PROBE";
const PROBE_TEST: &str = "context_creation_refusal_keeps_existing_frame_open";
#[test]
fn context_creation_refusal_keeps_existing_frame_open() {
    if let Some(mode) = std::env::var_os(PROBE_ENV) {
        let refuse_creation = match mode.to_str() {
            Some("baseline") => false,
            Some("refusal") => true,
            _ => panic!("context probe {mode:?}: expected baseline or refusal"),
        };
        active_frame_probe(refuse_creation);
        return;
    }
    for mode in ["baseline", "refusal"] {
        let output = isolated_context_probe(mode);
        assert!(
            output.status.success(),
            "{mode}: expected a retained active frame; {}",
            String::from_utf8_lossy(&output.stderr)
        );
    }
}
fn isolated_context_probe(mode: &str) -> Output {
    // A library C++ assertion aborts the process; isolation keeps its exact failure observable.
    Command::new(std::env::current_exe().unwrap())
        .args(["--exact", PROBE_TEST, "--nocapture"])
        .env(PROBE_ENV, mode)
        .output()
        .unwrap()
}
fn active_frame_probe(refuse_creation: bool) {
    let mut context = imgui::Context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [320., 200.];
    context.fonts().build_rgba32_texture();
    for _ in 0..2 {
        let ui = context.frame();
        if refuse_creation {
            assert!(GuiFrameEngine::new().is_err());
        }
        ui.window("Retained owner")
            .build(|| ui.text("The existing frame still owns its context"));
        context.render();
    }
}
