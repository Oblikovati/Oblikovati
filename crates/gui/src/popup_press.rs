// SPDX-License-Identifier: GPL-2.0-only
//! Preserve button release semantics when a non-modal popup closes on press.
use imgui::{ItemHoveredFlags, MouseButton, Ui};

#[derive(Debug)]
pub(crate) struct PopupPress<T> {
    pending: Option<T>,
}

impl<T> Default for PopupPress<T> {
    fn default() -> Self {
        Self { pending: None }
    }
}

impl<T: Copy + Eq> PopupPress<T> {
    /// Complete one control, for example `press.button(ui, (), clicked)` after drawing it.
    pub(crate) fn button(&mut self, ui: &Ui, target: T, clicked: bool) -> bool {
        let through_popup = self.released(ui, target);
        self.finish_frame(ui);
        clicked || through_popup
    }

    /// Return true on release over the pressed control; for example, a ribbon group (#3849).
    pub(crate) fn released(&mut self, ui: &Ui, target: T) -> bool {
        if !ui.is_item_hovered()
            && ui.is_item_hovered_with_flags(ItemHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP)
            && ui.is_mouse_clicked(MouseButton::Left)
        {
            self.pending = Some(target);
        }
        if !ui.is_mouse_released(MouseButton::Left) || self.pending != Some(target) {
            return false;
        }
        self.pending = None;
        ui.is_item_hovered()
    }

    /// Retire a press when its target disappeared before release.
    pub(crate) fn finish_frame(&mut self, ui: &Ui) {
        if ui.is_mouse_released(MouseButton::Left) {
            self.pending = None;
        }
    }
}
