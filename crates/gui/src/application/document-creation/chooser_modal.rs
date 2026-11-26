// SPDX-License-Identifier: GPL-2.0-only
//! Present the template chooser and return to it from a docked Projects window.
use super::{
    GuiCommandState, GuiDrawingLayoutAction, GuiShellAction, TemplateChooser, has_layout_sources,
};
use crate::icons::GuiRibbonIcons;
use imgui::{Condition, Key, Ui};

const TITLE: &str = "New document";

impl TemplateChooser {
    pub(crate) fn request(&mut self) {
        if self.projects_return_geometry.is_none() {
            self.clear_gallery_navigation();
            self.requested = true;
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
    ) {
        self.sync_active_project(state);
        if ui.io().key_ctrl && ui.is_key_pressed(Key::N) && !state.transaction_open {
            self.request();
        }
        if self.projects_return_geometry.is_some() && self.projects.is_requested() {
            self.projects.panel(ui, state, actions);
        } else {
            self.chooser(ui, state, actions, icons);
        }
        self.component.panel(ui, state, actions);
    }

    fn chooser(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &GuiRibbonIcons,
    ) {
        self.open_requested_chooser(ui, state, actions);
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            self.projects.close();
            self.clear_gallery_navigation();
            return;
        };
        self.contents(ui, state, icons);
        self.buttons(ui, state, actions);
        if self.projects.opens_in_dock(ui) {
            // A docked tab needs pointer input from the workspace outside the chooser modal.
            self.projects_return_geometry = Some((ui.window_pos(), ui.window_size()));
            ui.close_current_popup();
        } else {
            self.projects.panel(ui, state, actions);
        }
    }

    fn open_requested_chooser(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some((position, size)) = self.projects_return_geometry.take() {
            ui.open_popup(TITLE);
            crate::widget_abi::window_geometry(ui, position, size, Condition::Always);
        } else if std::mem::take(&mut self.requested) {
            if has_layout_sources(state) {
                actions.push(GuiShellAction::DrawingLayout(
                    GuiDrawingLayoutAction::CaptureSources,
                ));
            }
            self.reveal_project_folder(state);
            self.restore_project_selection(state);
            ui.open_popup(TITLE);
            dialog_geometry(ui);
        }
    }

    /// A frame must follow so a nested dialog opens or returns without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        let chooser_needs_frame = if self.projects_return_geometry.is_some() {
            !self.projects.is_requested()
        } else {
            self.gallery_scroll_frames > 0
        };
        self.projects.needs_frame() || self.component.needs_frame() || chooser_needs_frame
    }
}

fn dialog_geometry(ui: &Ui) {
    let size = [
        820_f32.min(ui.io().display_size[0] - 24.),
        400_f32.min(ui.io().display_size[1] - 24.),
    ];
    let position = std::array::from_fn(|axis| (ui.io().display_size[axis] - size[axis]) * 0.5);
    crate::widget_abi::window_geometry(ui, position, size, Condition::Appearing);
}
