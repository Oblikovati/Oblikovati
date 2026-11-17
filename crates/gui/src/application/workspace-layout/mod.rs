// SPDX-License-Identifier: GPL-2.0-only
//! Panel visibility and the model browser's dock; the viewport fills whatever they leave free.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui, WindowFlags};

pub(crate) mod body_properties;
pub(crate) mod browser_feature;
pub(crate) mod browser_origin;
#[path = "browser_view.rs"]
pub(crate) mod browser_view;
#[path = "panel_tabs.rs"]
pub(crate) mod panel_tabs;
#[path = "status_bar.rs"]
pub(crate) mod status_bar;
/// The document tab row and the gap above it.
const TAB_STRIP_HEIGHT: f32 = 26.;
const EDGE_MARGIN: f32 = 8.;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct ToolVisibility {
    pub material: bool,
    pub appearance: bool,
    pub parameters: bool,
    pub selection_filter: bool,
    pub history: bool,
}

#[derive(Debug)]
pub(crate) struct WorkspaceLayout {
    pub panels: panel_tabs::DockPanels,
    dock_state: crate::dockable_lifecycle::DockableState,
    pub tool_visibility: ToolVisibility,
    pub tabs_visible: bool,
    outer_center: Option<crate::application::document_creation::workspaces::WorkspaceArea>,
    outer_bottom_docked: bool,
    pub(crate) outer_bottom_span: Option<[f32; 2]>,
    /// GUI frames follow events, and a dropped panel takes its geometry in the next frame.
    pub pending_layout: bool,
    pub switcher: crate::application::document_tabs::WindowSwitcher,
    view: browser_view::BrowserView,
    origin_selection_document: Option<u64>,
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
            dock_state: Default::default(),
            tool_visibility: Default::default(),
            tabs_visible: true,
            outer_center: None,
            outer_bottom_docked: false,
            outer_bottom_span: None,
            pending_layout: false,
            switcher: Default::default(),
            view: Default::default(),
            origin_selection_document: None,
            view_clicks_taken: false,
            ribbon_left: 0.,
            ribbon_right: 0.,
        }
    }
}
impl DockableWindow for WorkspaceLayout {}
impl WorkspaceLayout {
    /// Inspect test-only browser geometry; example: `layout.test_browser_view()`.
    #[cfg(test)]
    pub(crate) fn test_browser_view(&self) -> &browser_view::BrowserView {
        &self.view
    }
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Model", &self.dock_state)?;
        self.view.body_properties.register_dockable(registry)?;
        self.view.feature_properties.register_dockable(registry)?;
        self.view.geometry.register_dockable(registry)?;
        self.view.relationships.register_dockable(registry)?;
        self.view.find.register_dockable(registry)
    }
    /// Clear a browser origin selection when its document changes; example: `layout.sync_origin_document(Some(1))`.
    pub(crate) fn sync_origin_document(&mut self, document: Option<u64>) {
        if self.origin_selection_document != document {
            self.view.selected_origin_row = None;
            self.view.origin_folder_chosen = false;
            self.origin_selection_document = document;
        }
    }
    /// Consume a selected origin plane; example: `layout.take_selected_origin_plane()`.
    pub(crate) fn take_selected_origin_plane(&mut self) -> Option<crate::GuiOriginPlaneKind> {
        let row = self.view.selected_origin_row?;
        let label = *crate::browser::ORIGIN_ROWS.get(row)?;
        let plane = [
            crate::GuiOriginPlaneKind::Xy,
            crate::GuiOriginPlaneKind::Xz,
            crate::GuiOriginPlaneKind::Yz,
        ]
        .into_iter()
        .find(|plane| plane.label() == label)?;
        self.view.selected_origin_row = None;
        Some(plane)
    }
    /// Open the shared browser search; example: `layout.request_find()`.
    pub(crate) fn request_find(&mut self) {
        self.view.find.request();
    }
    fn draw_find(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if let Some(browser) = &state.browser {
            self.view.find.panel(ui, state, browser, actions);
        }
    }
    pub(crate) fn browser_consumed_arrow(&self) -> bool {
        self.view.arrow_consumed
    }
    pub(crate) fn feature_keyboard_focus(&self) -> bool {
        self.view.feature_keyboard_focus
    }
    /// Draw the browser in its dock; example: `layout.browser_panel(ui, state, 900., 158., actions)`.
    pub(crate) fn browser_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        bounds: (f32, f32),
        document_area: crate::application::document_creation::workspaces::WorkspaceArea,
        part_context_open: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.view.arrow_consumed = false;
        self.view.feature_keyboard_focus = false;
        let (_, band_height) = bounds;
        self.view.document_area = Some(document_area);
        self.view.view_clicks_taken = self.view_clicks_taken;
        browser_origin::follow_view_click(ui, state, &mut self.view, part_context_open, actions);
        self.dock_state.set_open(self.panels.dock_visible());
        if !self.panels.dock_visible() {
            self.empty_dock_control(ui, band_height);
            self.draw_find(ui, state, actions);
            return;
        }
        let mut open = true;
        // Restoring the panel must not take focus, which would close the menu that restored it.
        Self::dockable_window(ui, "Model", &mut open)
            .flags(WindowFlags::NO_FOCUS_ON_APPEARING)
            .build(|| self.dock_contents(ui, state, actions));
        if !open {
            self.panels.close_all();
            self.dock_state.set_open(false);
        }
        self.pending_layout |= self.panels.needs_frame();
        self.draw_find(ui, state, actions);
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
        } else if active == panel_tabs::DockPanel::Favorites {
            self.panels.favorites.panel(ui);
        } else if self.panels.model_open {
            crate::browser::browser_contents(ui, state, &mut self.view, actions);
            self.pending_layout |= self.view.needs_frame();
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
        let document = request.commands.document.unwrap_or(0);
        let visible = self.view.visible_origin_rows.iter();
        let shown = visible
            .filter(|(owner, _)| *owner == document)
            .map(|(_, row)| *row);
        browser_origin::draw_selected_origin(ui, request, self.view.selected_origin_row, shown);
    }
    pub(crate) fn reserve_for_ribbon(&mut self, left: f32, right: f32) {
        self.ribbon_left = left;
        self.ribbon_right = right;
    }
    /// The panel dock fills the region below the ribbon and above the status bar.
    /// Example: `layout.outer_area([1440., 900.], 120.)` after reserving ribbon margins.
    pub(crate) fn outer_area(
        &self,
        display: [f32; 2],
        band_height: f32,
    ) -> crate::application::document_creation::workspaces::WorkspaceArea {
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: self.ribbon_left,
            top: band_height,
            right: display[0] - self.ribbon_right,
            bottom: display[1]
                - status_bar::STATUS_BAR_HEIGHT
                - if self.outer_bottom_docked {
                    self.tab_height()
                } else {
                    0.
                },
        }
    }
    /// Record the outer dock's central area for workspace layout.
    /// Example: `layout.set_outer_center(area, root)` after drawing the outer dock.
    pub(crate) fn set_outer_center(
        &mut self,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        root: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        let bottom_docked = area.bottom < root.bottom - 1.;
        self.pending_layout |= bottom_docked != self.outer_bottom_docked;
        self.outer_bottom_docked = bottom_docked;
        self.outer_center = Some(area);
    }
    /// The horizontal span left by docked panels.
    /// Example: `layout.tab_span(1440.)` after the dockspace has been drawn.
    pub(crate) fn tab_span(&self, display_width: f32) -> (f32, f32) {
        let left = self.outer_center.map_or(self.ribbon_left, |area| area.left);
        let right = self
            .outer_center
            .map_or(display_width - self.ribbon_right, |area| area.right);
        (left, (right - left).max(1.))
    }
    /// Align the tab row with the dock directly above it.
    /// Example: `layout.document_tab_span(1440., viewport)` after drawing the dockspaces.
    pub(crate) fn document_tab_span(
        &self,
        display_width: f32,
        viewport: crate::application::document_creation::workspaces::WorkspaceArea,
    ) -> (f32, f32) {
        if self.outer_bottom_docked
            && let Some([left, right]) = self.outer_bottom_span
        {
            let start = left.max(self.ribbon_left);
            let end = right.min(display_width - self.ribbon_right);
            if start < end {
                return (start, end - start);
            }
        }
        (viewport.left, (viewport.right - viewport.left).max(1.))
    }
    /// Reserve the tab row only while tabs are visible; example: `layout.tab_height()`.
    pub(crate) fn tab_height(&self) -> f32 {
        if self.tabs_visible {
            TAB_STRIP_HEIGHT
        } else {
            0.
        }
    }
    /// Find the top of the bottom tab row; example: `layout.tab_top(900.)`.
    pub(crate) fn tab_top(&self, height: f32) -> f32 {
        let bottom = if self.outer_bottom_docked {
            height - status_bar::STATUS_BAR_HEIGHT
        } else {
            self.outer_center
                .map_or(height - status_bar::STATUS_BAR_HEIGHT, |area| area.bottom)
        };
        (bottom - self.tab_height()).max(0.)
    }
    /// The screen area left for a document after the browser, tabs and ribbon band.
    /// Example: `layout.document_area([1440., 900.], 120.)`.
    pub(crate) fn document_area(
        &self,
        display: [f32; 2],
        band_height: f32,
    ) -> crate::application::document_creation::workspaces::WorkspaceArea {
        let (left, width) = self.tab_span(display[0]);
        crate::application::document_creation::workspaces::WorkspaceArea {
            left,
            top: self.outer_center.map_or(band_height, |area| area.top),
            right: left + width,
            bottom: if self.outer_bottom_docked {
                self.outer_center
                    .map_or(self.tab_top(display[1]), |area| area.bottom)
            } else {
                self.tab_top(display[1])
            },
        }
    }
    /// Draw the View tab's panel checklist; example: `layout.user_interface_menu(ui, [108., 18.], None)`.
    pub(crate) fn user_interface_menu(
        &mut self,
        ui: &Ui,
        size: [f32; 2],
        icons: Option<&crate::icons::GuiRibbonIcons>,
    ) {
        let clicked = icons.map_or_else(
            || crate::widget_abi::button(ui, "User Interface", size),
            |icons| {
                icons.button(
                    ui,
                    "User Interface",
                    crate::icons::IconGlyph::UserInterface,
                    size,
                )
            },
        );
        if clicked {
            ui.open_popup("##user-interface");
        }
        if let Some(_menu) = ui.begin_popup("##user-interface") {
            ui.checkbox("Model Browser", &mut self.panels.model_open);
            ui.checkbox("Document Tabs", &mut self.tabs_visible);
            ui.separator();
            ui.checkbox("Material Browser", &mut self.tool_visibility.material);
            ui.checkbox("Appearance Browser", &mut self.tool_visibility.appearance);
            ui.checkbox("Parameters", &mut self.tool_visibility.parameters);
            ui.checkbox(
                "Selection Filter",
                &mut self.tool_visibility.selection_filter,
            );
            ui.checkbox("History Browser", &mut self.tool_visibility.history);
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/layout.rs"]
mod tests;
