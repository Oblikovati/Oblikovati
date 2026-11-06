// SPDX-License-Identifier: GPL-2.0-only
//! Panel visibility and the model browser's dock; the viewport fills whatever they leave free.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, MouseButton, Ui, WindowFlags};

/// The tab strip and the gap under it.
const TAB_STRIP_HEIGHT: f32 = 34.;
const EDGE_MARGIN: f32 = 8.;
/// A header released this close to a side edge docks there.
const DOCK_TARGET: f32 = 48.;
const FLOATING_SIZE: [f32; 2] = [280., 360.];

#[derive(Clone, Copy, Debug, Default, PartialEq)]
enum BrowserDock {
    #[default]
    Left,
    Right,
    Floating([f32; 2]),
}
#[derive(Debug)]
pub(crate) struct WorkspaceLayout {
    pub browser_visible: bool,
    pub tabs_visible: bool,
    dock: BrowserDock,
    dragging: bool,
    window_position: [f32; 2],
    /// GUI frames follow events, and a dropped panel takes its geometry in the next frame.
    pub pending_layout: bool,
    pub switcher: crate::application::document_tabs::WindowSwitcher,
}
impl Default for WorkspaceLayout {
    fn default() -> Self {
        Self {
            browser_visible: true,
            tabs_visible: true,
            dock: BrowserDock::default(),
            dragging: false,
            window_position: [0.; 2],
            pending_layout: false,
            switcher: Default::default(),
        }
    }
}
impl WorkspaceLayout {
    /// Draw the browser in its dock; example: `layout.browser_panel(ui, state, 900., 158., actions)`.
    pub(crate) fn browser_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        height: f32,
        band_height: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.pending_layout = false;
        if !self.browser_visible {
            return;
        }
        self.place_browser(ui, height, band_height);
        ui.window("Model")
            // Restoring the panel must not take focus, which would close the menu that restored it.
            .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_FOCUS_ON_APPEARING)
            .resizable(false)
            .collapsible(false)
            .build(|| {
                self.track_header(ui);
                crate::browser::browser_contents(ui, state, actions);
            });
        self.finish_drag(ui);
    }
    fn place_browser(&self, ui: &Ui, height: f32, band_height: f32) {
        // While the header is held, Dear ImGui owns the window position.
        if self.dragging {
            return;
        }
        let width = ui.io().display_size[0];
        let docked = [
            crate::browser::model_browser_width(width),
            (height - band_height - 18.).max(120.),
        ];
        let top = band_height + 10.;
        let (position, size) = match self.dock {
            BrowserDock::Left => ([EDGE_MARGIN, top], docked),
            BrowserDock::Right => ([width - docked[0] - EDGE_MARGIN, top], docked),
            BrowserDock::Floating(position) => (position, FLOATING_SIZE),
        };
        crate::widget_abi::window_geometry(ui, position, size, Condition::Always);
    }
    fn track_header(&mut self, ui: &Ui) {
        self.window_position = ui.window_pos();
        let header = ui.window_pos()[1] + ui.frame_height();
        if ui.is_mouse_clicked(MouseButton::Left)
            && ui.is_window_hovered()
            && ui.io().mouse_pos[1] < header
        {
            self.dragging = true;
        }
    }
    fn finish_drag(&mut self, ui: &Ui) {
        if self.dragging && ui.is_mouse_released(MouseButton::Left) {
            self.dragging = false;
            self.pending_layout = true;
            self.dock = drop_target(
                ui.io().mouse_pos[0],
                ui.io().display_size[0],
                self.window_position,
            );
        }
    }
    /// The horizontal span the document tab strip may use; example: `layout.tab_span(1440.)`.
    pub(crate) fn tab_span(&self, width: f32) -> (f32, f32) {
        let reserved = crate::browser::model_browser_width(width) + 2. * EDGE_MARGIN;
        match (self.browser_visible, self.dock) {
            (true, BrowserDock::Left) => (reserved, (width - reserved).max(1.)),
            (true, BrowserDock::Right) => (EDGE_MARGIN, (width - reserved - EDGE_MARGIN).max(1.)),
            _ => (EDGE_MARGIN, (width - 2. * EDGE_MARGIN).max(1.)),
        }
    }
    /// The screen area left for a document after the browser, tabs and ribbon band.
    /// Example: `layout.document_area([1440., 900.], 120.)`.
    pub(crate) fn document_area(
        &self,
        display: [f32; 2],
        band_height: f32,
    ) -> crate::application::document_creation::workspaces::WorkspaceArea {
        let (left, width) = self.tab_span(display[0]);
        let tabs = if self.tabs_visible {
            TAB_STRIP_HEIGHT
        } else {
            0.
        };
        crate::application::document_creation::workspaces::WorkspaceArea {
            left,
            top: band_height + tabs + EDGE_MARGIN,
            right: left + width,
            bottom: display[1] - EDGE_MARGIN,
        }
    }
    /// Draw the View tab's panel checklist; example: `layout.user_interface_menu(ui, [108., 18.])`.
    pub(crate) fn user_interface_menu(&mut self, ui: &Ui, size: [f32; 2]) {
        if crate::widget_abi::button(ui, "User Interface", size) {
            ui.open_popup("##user-interface");
        }
        if let Some(_menu) = ui.begin_popup("##user-interface") {
            ui.checkbox("Model Browser", &mut self.browser_visible);
            ui.checkbox("Document Tabs", &mut self.tabs_visible);
        }
    }
}
fn drop_target(pointer_x: f32, width: f32, position: [f32; 2]) -> BrowserDock {
    if pointer_x < DOCK_TARGET {
        return BrowserDock::Left;
    }
    if pointer_x > width - DOCK_TARGET {
        return BrowserDock::Right;
    }
    BrowserDock::Floating(position)
}
#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/layout.rs"]
mod tests;
