// SPDX-License-Identifier: GPL-2.0-only
//! Semantics and actions for guarded application-close prompts.

/// A choice in the active guarded-close prompt.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiClosePromptChoice {
    ContinueSketch,
    Save,
    Discard,
    Cancel,
}

/// One visible prompt button and its framebuffer bounds.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiClosePromptButton {
    pub label: &'static str,
    pub choice: GuiClosePromptChoice,
    pub bounds: [f32; 4],
    pub focused: bool,
}

/// The current guarded-close prompt as drawn by the GUI.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiClosePrompt {
    pub title: &'static str,
    pub message: String,
    pub bounds: [f32; 4],
    pub buttons: Vec<GuiClosePromptButton>,
}

/// A visible application-frame or close-prompt control that can receive keyboard focus.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiAccessibleFocus {
    Frame(usize),
    ClosePrompt(GuiClosePromptChoice),
}

pub(crate) fn focused_activation(ui: &imgui::Ui) -> bool {
    let io = ui.io();
    ui.is_item_focused()
        && !io.key_ctrl
        && !io.key_alt
        && !io.key_super
        && (ui.is_key_pressed_no_repeat(imgui::Key::Enter)
            || ui.is_key_pressed_no_repeat(imgui::Key::Space))
}
