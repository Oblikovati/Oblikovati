// SPDX-License-Identifier: GPL-2.0-only
//! Keep expanded panels open while their context menus receive pointer input.
use super::{RibbonDisplayState, RibbonPanel, RibbonTab};
use imgui::Ui;

pub(super) fn submenu_row_pressed(ui: &Ui, label: &str, submenu_open: bool) -> bool {
    let selected = crate::widget_abi::dialog_selectable(ui, label, false);
    // ImGui consumes the first click to dismiss a child popup before its parent row activates.
    let hovered =
        ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP);
    selected || submenu_open && hovered && ui.is_mouse_clicked(imgui::MouseButton::Left)
}

impl RibbonDisplayState {
    /// Open the ribbon menu at a title or elsewhere, for example `request_context_menu(None)`.
    pub(crate) fn request_context_menu(&mut self, title: Option<RibbonPanel>) {
        self.context_title = title;
        self.context_expansion = None;
        self.context_command = None;
        self.context_had_submenu = false;
        self.context_requested = true;
    }
    pub(crate) fn request_expansion_context(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        self.context_expansion = Some((tab, panel));
        self.context_title = None;
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
    pub(crate) fn take_context_relocated(&mut self) -> Option<(RibbonTab, RibbonPanel)> {
        self.context_relocated.take()
    }
    pub(super) fn finish_title_context_selection(&mut self) {
        self.close_context_menu = true;
        self.close_unpinned_expansion =
            self.context_expansion.is_some() && self.context_command.is_none();
    }
    pub(crate) fn take_context_expansion_close(&mut self) -> bool {
        std::mem::take(&mut self.close_unpinned_expansion)
    }
    pub(super) fn expanded_command_menu(
        &mut self,
        ui: &Ui,
        customization: &mut crate::application::ribbon_customization::RibbonCustomization,
        (state, toolbar): (
            &crate::GuiCommandState,
            &mut crate::application::quick_access_toolbar::QuickAccessToolbar,
        ),
    ) {
        let Some(label) = self.context_command else {
            return;
        };
        let recognized =
            label == crate::move_body_ribbon::LABEL || label == crate::shell_ribbon::LABEL;
        if !recognized {
            muted_context_item(ui, "Add to Quick Access Toolbar");
            muted_context_item(ui, "Move to Main Panel");
            return;
        }
        if label == crate::move_body_ribbon::LABEL {
            if crate::move_body_ribbon::menu_item(
                ui,
                "Add to Quick Access Toolbar",
                !toolbar.has_ribbon_command(state, label),
            ) {
                toolbar.add_ribbon_command(state, label);
            }
        } else {
            muted_context_item(ui, "Add to Quick Access Toolbar");
        }
        if crate::move_body_ribbon::menu_item(ui, "Move to Main Panel", true) {
            let result = if label == crate::shell_ribbon::LABEL {
                customization.relocate_builtin(crate::shell_ribbon::ID, false)
            } else {
                crate::move_body_ribbon::set_main(customization, true)
            };
            match result {
                Ok(()) => self.context_relocated = self.context_expansion,
                Err(error) => customization.report_command_error(error),
            }
        }
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

fn muted_context_item(ui: &Ui, label: &str) {
    ui.menu_item_config(label).enabled(false).build();
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("This command is not available in this build.");
    }
}
