// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::*;
mod parameter_journey;
#[test]
fn mouse_and_keyboard_create_and_edit_a_size_parameter_without_draft_leaks() {
    let engine = oblikovati_gui::GuiFrameEngine::new().unwrap();
    parameter_journey::exercise_parameter_journey(engine, cfg!(target_os = "macos"));
}
