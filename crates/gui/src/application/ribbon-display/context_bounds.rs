// SPDX-License-Identifier: GPL-2.0-only
//! Keep expanded panels open while their context menus receive pointer input.
use super::{RibbonDisplayState, RibbonPanel, RibbonTab};
use imgui::Ui;

impl RibbonDisplayState {
    pub(crate) fn request_expansion_context(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        self.context_expansion = Some((tab, panel));
        self.context_command = None;
        self.context_had_submenu = false;
        self.context_requested = true;
    }
    pub(crate) fn request_expansion_command_context(
        &mut self,
        tab: RibbonTab,
        panel: RibbonPanel,
        command: &'static str,
    ) {
        self.request_expansion_context(tab, panel);
        self.context_command = Some(command);
    }
    pub(crate) fn expansion_context_escape(
        &self,
        ui: &Ui,
        tab: RibbonTab,
        panel: RibbonPanel,
    ) -> bool {
        self.context_expansion == Some((tab, panel))
            && self.context_bounds.is_some()
            && self.submenu_bounds.is_none()
            && self.context_had_submenu
            && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
    }
    pub(super) fn capture_context_bounds(&mut self, ui: &Ui) {
        self.context_bounds = Some(window_bounds(ui));
        self.submenu_bounds = None;
    }
    pub(super) fn capture_submenu_bounds(&mut self, ui: &Ui) {
        self.submenu_bounds = Some(window_bounds(ui));
        self.context_had_submenu = true;
    }
    pub(crate) fn context_contains(&self, point: [f32; 2]) -> bool {
        [self.context_bounds, self.submenu_bounds]
            .into_iter()
            .flatten()
            .any(|bounds| {
                (0..2).all(|axis| point[axis] >= bounds[0][axis] && point[axis] < bounds[1][axis])
            })
    }
    /// Forward a tab press that dismisses its context menu, if the menu does not cover the tab.
    pub(crate) fn context_tab_press(&self, ui: &Ui) -> bool {
        if self.context_bounds.is_none() || !ui.is_mouse_clicked(imgui::MouseButton::Left) {
            return false;
        }
        let point = ui.io().mouse_pos;
        let min = ui.item_rect_min();
        let max = ui.item_rect_max();
        !self.context_contains(point)
            && (0..2).all(|axis| point[axis] >= min[axis] && point[axis] <= max[axis])
    }
    pub(crate) fn take_docking_selection(&mut self) -> bool {
        std::mem::take(&mut self.docking_selected)
    }
}

fn window_bounds(ui: &Ui) -> [[f32; 2]; 2] {
    let position = ui.window_pos();
    let size = ui.window_size();
    [position, [position[0] + size[0], position[1] + size[1]]]
}
