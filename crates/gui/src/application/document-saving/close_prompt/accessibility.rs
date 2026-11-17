// SPDX-License-Identifier: GPL-2.0-only
//! Capture real guarded-close controls for native accessibility.
use super::*;
impl ClosePrompt {
    pub(crate) fn answer_accessibly(&mut self, choice: crate::GuiClosePromptChoice) {
        self.accessible_choice = Some(choice);
    }
    pub(crate) fn focus_accessibly(&mut self, choice: crate::GuiClosePromptChoice) {
        self.accessible_focus = Some(choice);
    }
    pub(crate) fn semantics(&self) -> Option<crate::GuiClosePrompt> {
        self.accessible_prompt.clone()
    }
    pub(super) fn begin_semantics(&mut self, ui: &Ui, title: &'static str, message: String) {
        let [x, y] = ui.window_pos();
        let [width, height] = ui.window_size();
        self.accessible_prompt = Some(crate::GuiClosePrompt {
            title,
            message,
            bounds: [x, y, x + width, y + height],
            buttons: Vec::new(),
        });
    }
    pub(super) fn prompt_button(
        &mut self,
        ui: &Ui,
        label: &'static str,
        choice: crate::GuiClosePromptChoice,
    ) -> bool {
        if self.accessible_focus == Some(choice) {
            self.accessible_focus = None;
            ui.set_keyboard_focus_here();
        }
        let clicked = crate::widget_abi::button(ui, label, [80., 0.]);
        let clicked = clicked || crate::accessibility::focused_activation(ui);
        let [x1, y1] = ui.item_rect_min();
        let [x2, y2] = ui.item_rect_max();
        if let Some(prompt) = &mut self.accessible_prompt {
            prompt.buttons.push(crate::GuiClosePromptButton {
                label,
                choice,
                bounds: [x1, y1, x2, y2],
                focused: ui.is_item_focused(),
            });
        }
        clicked
    }
    pub(super) fn save_changes_buttons(&mut self, ui: &Ui) -> Option<CloseAnswer> {
        let pending = self.accessible_choice.take();
        let mut chosen = None;
        for (label, choice, answer) in [
            ("Yes", crate::GuiClosePromptChoice::Save, CloseAnswer::Save),
            (
                "No",
                crate::GuiClosePromptChoice::Discard,
                CloseAnswer::Discard,
            ),
        ] {
            if self.prompt_button(ui, label, choice) || pending == Some(choice) {
                chosen = Some(answer);
            }
            ui.same_line();
        }
        if self.prompt_button(ui, "Cancel", crate::GuiClosePromptChoice::Cancel)
            || pending == Some(crate::GuiClosePromptChoice::Cancel)
            || self.escape.pressed(ui)
        {
            Some(CloseAnswer::Keep)
        } else {
            chosen
        }
    }
}
