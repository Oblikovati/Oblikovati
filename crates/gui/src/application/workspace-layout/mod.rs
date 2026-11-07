// SPDX-License-Identifier: GPL-2.0-only
//! Panel visibility and the model browser's dock; the viewport fills whatever they leave free.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, MouseButton, Ui, WindowFlags};

pub(crate) mod body_properties;
pub(crate) mod browser_feature;
pub(crate) mod browser_origin;
#[path = "browser_view.rs"]
pub(crate) mod browser_view;
#[path = "panel_tabs.rs"]
pub(crate) mod panel_tabs;
#[path = "status_bar.rs"]
pub(crate) mod status_bar;
/// The tab strip and the gap under it.
const TAB_STRIP_HEIGHT: f32 = 34.;
const BOUNDARY_WIDTH: f32 = 6.;
const PLUS_CONTROL_WIDTH: f32 = 34.;
const MINIMUM_BROWSER_WIDTH: f32 = 180.;
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
    pub panels: panel_tabs::DockPanels,
    pub tabs_visible: bool,
    dock: BrowserDock,
    dragging: bool,
    window_position: [f32; 2],
    /// GUI frames follow events, and a dropped panel takes its geometry in the next frame.
    pub pending_layout: bool,
    pub switcher: crate::application::document_tabs::WindowSwitcher,
    browser_width_override: Option<f32>,
    view: browser_view::BrowserView,
    /// A navigation command owns the next click on the view, not the browser's selection.
    pub view_clicks_taken: bool,
    /// What a docked ribbon takes from each side of the window.
    ribbon_left: f32,
    ribbon_right: f32,
}
impl Default for WorkspaceLayout {
    fn default() -> Self {
        Self {
            panels: Default::default(),
            tabs_visible: true,
            dock: BrowserDock::default(),
            dragging: false,
            window_position: [0.; 2],
            pending_layout: false,
            switcher: Default::default(),
            browser_width_override: None,
            view: Default::default(),
            view_clicks_taken: false,
            ribbon_left: 0.,
            ribbon_right: 0.,
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
        if !self.panels.dock_visible() {
            self.empty_dock_control(ui, band_height);
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
                self.dock_contents(ui, state, actions);
            });
        self.pending_layout |= self.panels.needs_frame();
        self.finish_drag(ui);
        self.boundary_handle(ui, height, band_height);
    }
    fn dock_contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.panels.plus_control(ui);
        ui.same_line();
        let active = self.panels.tabs(ui);
        if active == panel_tabs::DockPanel::Log {
            self.panels.log_area(ui);
        } else if self.panels.model_open {
            self.view.view_clicks_taken = self.view_clicks_taken;
            crate::browser::browser_contents(ui, state, &mut self.view, actions);
            self.pending_layout |= self.view.body_properties.needs_frame();
        }
    }
    /// With every panel closed, only the plus control remains to bring one back.
    fn empty_dock_control(&mut self, ui: &Ui, band_height: f32) {
        let position = [self.ribbon_left + EDGE_MARGIN, band_height + 2.];
        ui.window("##panel-plus")
            .position(position, Condition::Always)
            .flags(
                WindowFlags::NO_DECORATION
                    | WindowFlags::NO_MOVE
                    | WindowFlags::NO_DOCKING
                    | WindowFlags::NO_SAVED_SETTINGS
                    | WindowFlags::ALWAYS_AUTO_RESIZE
                    | WindowFlags::NO_FOCUS_ON_APPEARING,
            )
            .build(|| self.panels.plus_control(ui));
        self.pending_layout |= self.panels.needs_frame();
    }
    /// Leave room for a docked ribbon; example: `layout.reserve_for_ribbon(196., 0.)`.
    /// Show which origin row is chosen, in the view; example: each frame, under the panels.
    pub(crate) fn draw_selected_origin(&mut self, ui: &Ui, request: &crate::GuiFrameRequest) {
        self.view.layout_scale = request.layout_scale();
        browser_origin::draw_selected_origin(ui, request, self.view.selected_origin_row);
    }
    pub(crate) fn reserve_for_ribbon(&mut self, left: f32, right: f32) {
        self.ribbon_left = left;
        self.ribbon_right = right;
    }
    /// The width the docked browser takes: a user's drag, else the window's default.
    fn browser_width(&self, display_width: f32) -> f32 {
        self.browser_width_override
            .unwrap_or_else(|| crate::browser::model_browser_width(display_width))
    }
    /// A thin window over the boundary, so a drag never competes with the tree's own rows.
    fn boundary_handle(&mut self, ui: &Ui, height: f32, band_height: f32) {
        let display = ui.io().display_size[0];
        let width = self.browser_width(display);
        let x = match self.dock {
            BrowserDock::Left => self.ribbon_left + EDGE_MARGIN + width,
            BrowserDock::Right => {
                display - self.ribbon_right - width - EDGE_MARGIN - BOUNDARY_WIDTH
            }
            BrowserDock::Floating(_) => return,
        };
        let top = band_height + 10.;
        let size = [BOUNDARY_WIDTH, (height - band_height - 18.).max(120.)];
        crate::widget_abi::window_geometry(ui, [x, top], size, Condition::Always);
        // A window's defaults would keep this handle 32 pixels wide and clip its own edges.
        let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
        let _minimum = ui.push_style_var(imgui::StyleVar::WindowMinSize([1., 1.]));
        ui.window("##browser-boundary")
            .flags(boundary_flags())
            .build(|| self.boundary_drag(ui, size, display));
    }
    fn boundary_drag(&mut self, ui: &Ui, size: [f32; 2], display: f32) {
        ui.invisible_button("##browser-boundary-drag", size);
        if ui.is_item_hovered() || ui.is_item_active() {
            ui.set_mouse_cursor(Some(imgui::MouseCursor::ResizeEW));
        }
        if !ui.is_item_active() {
            return;
        }
        let delta = ui.io().mouse_delta[0];
        let signed = if self.dock == BrowserDock::Left {
            delta
        } else {
            -delta
        };
        let grown = self.browser_width(display) + signed;
        self.browser_width_override = Some(grown.clamp(MINIMUM_BROWSER_WIDTH, display * 0.5));
        self.pending_layout = true;
    }
    fn place_browser(&self, ui: &Ui, height: f32, band_height: f32) {
        // While the header is held, Dear ImGui owns the window position.
        if self.dragging {
            return;
        }
        let width = ui.io().display_size[0];
        let docked = [
            self.browser_width(width),
            (height - band_height - 18.).max(120.),
        ];
        let top = band_height + 10.;
        let (position, size) = match self.dock {
            BrowserDock::Left => ([self.ribbon_left + EDGE_MARGIN, top], docked),
            BrowserDock::Right => (
                [width - self.ribbon_right - docked[0] - EDGE_MARGIN, top],
                docked,
            ),
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
    pub(crate) fn tab_span(&self, display_width: f32) -> (f32, f32) {
        let width = display_width - self.ribbon_left - self.ribbon_right;
        let (left, span) = self.span_within(width);
        (left + self.ribbon_left, span)
    }
    fn span_within(&self, width: f32) -> (f32, f32) {
        let reserved = self.browser_width(width) + 2. * EDGE_MARGIN;
        match (self.panels.dock_visible(), self.dock) {
            (true, BrowserDock::Left) => (reserved, (width - reserved).max(1.)),
            (true, BrowserDock::Right) => (EDGE_MARGIN, (width - reserved - EDGE_MARGIN).max(1.)),
            // The plus control that brings a panel back sits in the strip's row.
            (false, _) => {
                let left = EDGE_MARGIN + PLUS_CONTROL_WIDTH;
                (left, (width - left - EDGE_MARGIN).max(1.))
            }
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
            bottom: display[1] - status_bar::STATUS_BAR_HEIGHT - EDGE_MARGIN,
        }
    }
    /// Draw the View tab's panel checklist; example: `layout.user_interface_menu(ui, [108., 18.])`.
    pub(crate) fn user_interface_menu(&mut self, ui: &Ui, size: [f32; 2]) {
        if crate::widget_abi::button(ui, "User Interface", size) {
            ui.open_popup("##user-interface");
        }
        if let Some(_menu) = ui.begin_popup("##user-interface") {
            ui.checkbox("Model Browser", &mut self.panels.model_open);
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

fn boundary_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_DOCKING
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_SCROLLBAR
}
