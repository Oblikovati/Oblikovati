// SPDX-License-Identifier: GPL-2.0-only
//! Panel visibility and the model browser's dock; the viewport fills whatever they leave free.
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::dockable_window::TabbedDockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, StyleVar, Ui, WindowFlags};

pub(crate) mod body_properties;
pub(crate) mod browser_feature;
pub(crate) mod browser_origin;
#[path = "browser_view.rs"]
pub(crate) mod browser_view;
pub(crate) mod document_dock_tabs;
#[path = "panel_tabs.rs"]
pub(crate) mod panel_tabs;
#[path = "status_bar.rs"]
pub(crate) mod status_bar;
/// The document tab row and the gap above it.
use crate::document_tabs::TAB_STRIP_HEIGHT;

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
    pub document_dock_tabs: document_dock_tabs::DocumentDockTabs,
    pub tool_visibility: ToolVisibility,
    pub tabs_visible: bool,
    pub view_cube_visible: bool,
    pub navigation_bar_visible: bool,
    pub status_bar_visible: bool,
    pub marking_menu_enabled: bool,
    pub selection_toolbar_visible: bool,
    outer_center: Option<crate::application::document_creation::workspaces::WorkspaceArea>,
    outer_bottom_docked: bool,
    pub(crate) outer_bottom_span: Option<[f32; 2]>,
    pub(crate) outer_dock_anchor: u32,
    /// GUI frames follow events, and a dropped panel takes its geometry in the next frame.
    pub pending_layout: bool,
    pub switcher: crate::application::document_tabs::WindowSwitcher,
    view: browser_view::BrowserView,
    origin_selection_document: Option<u64>,
    /// A navigation command owns the next click on the view, not the browser's selection.
    pub view_clicks_taken: bool,
    pub part_context_open: bool,
    /// What a docked ribbon takes from each side of the window.
    ribbon_left: f32,
    ribbon_right: f32,
}
impl Default for WorkspaceLayout {
    fn default() -> Self {
        Self {
            panels: Default::default(),
            document_dock_tabs: Default::default(),
            tool_visibility: Default::default(),
            tabs_visible: true,
            view_cube_visible: true,
            navigation_bar_visible: true,
            status_bar_visible: true,
            marking_menu_enabled: true,
            selection_toolbar_visible: true,
            outer_center: None,
            outer_bottom_docked: false,
            outer_bottom_span: None,
            outer_dock_anchor: 0,
            pending_layout: false,
            switcher: Default::default(),
            view: Default::default(),
            origin_selection_document: None,
            view_clicks_taken: false,
            part_context_open: false,
            ribbon_left: 0.,
            ribbon_right: 0.,
        }
    }
}
impl WorkspaceLayout {
    /// Inspect test-only browser geometry; example: `layout.test_browser_view()`.
    #[cfg(test)]
    pub(crate) fn test_browser_view(&self) -> &browser_view::BrowserView {
        &self.view
    }
    pub(crate) fn auto_expand_representations(&self) -> bool {
        self.view.auto_expand_representations()
    }
    pub(crate) fn set_auto_expand_representations(&mut self, enabled: bool) {
        self.view.set_auto_expand_representations(enabled);
    }
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.panels.register_dockables(registry)?;
        self.view.body_properties.register_dockable(registry)?;
        self.view.feature_properties.register_dockable(registry)?;
        self.view.geometry.register_dockable(registry)?;
        self.view.relationships.register_dockable(registry)?;
        self.view.part_view_scale.register_dockable(registry)?;
        self.view.find.register_dockable(registry)
    }
    /// Clear a browser origin selection when its document changes; example: `layout.sync_origin_document(Some(1))`.
    pub(crate) fn sync_origin_document(&mut self, document: Option<u64>) {
        if self.origin_selection_document != document {
            self.view.selected_origin_row = None;
            self.view.focused_origin_row = None;
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
    /// Open the Model Browser's tree filter; example: `layout.request_browser_search()`.
    pub(crate) fn request_browser_search(&mut self) {
        self.panels.open_panel(panel_tabs::DockPanel::Model);
        self.view.open_search();
        self.pending_layout = true;
    }
    /// Open the tree options from the Model dock tab.
    pub(crate) fn request_browser_menu(&mut self) {
        self.view.menu_requested = true;
        self.pending_layout = true;
    }
    /// Open the Find criteria dialog; example: `layout.request_find()`.
    pub(crate) fn request_find(&mut self) {
        self.view.find.request();
    }
    fn draw_find(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if let Some(browser) = &state.browser {
            self.view.find.panel(ui, state, browser, actions);
        }
    }
    fn draw_browser_dialogs(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(browser) = &state.browser else {
            return;
        };
        let _document = ui.push_id(format!("document-{:?}", state.document));
        self.view.dialogs(ui, state, browser, actions);
        self.pending_layout |= self.view.needs_frame();
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState, home_active: bool) {
        self.panels.sync_dock_states(home_active);
        self.view.sync_before_docking(state);
    }
    pub(crate) fn draw_status(
        &self,
        ui: &Ui,
        request: &crate::GuiFrameRequest,
        (alias, distance): (Option<&str>, Option<&str>),
    ) {
        if self.status_bar_visible {
            status_bar::status_bar(ui, request, alias, distance);
        }
    }
    pub(crate) fn browser_consumed_arrow(&self) -> bool {
        self.view.arrow_consumed
    }
    pub(crate) fn feature_keyboard_focus(&self) -> bool {
        self.view.feature_keyboard_focus
    }
    /// Draw the mini-toolbar when its workspace preference and current tool allow it.
    /// Example: `layout.selection_toolbar(ui, request, actions, menu_open)`.
    pub(crate) fn selection_toolbar(
        &self,
        ui: &Ui,
        request: &crate::GuiFrameRequest,
        actions: &mut Vec<GuiShellAction>,
        suppressed: bool,
    ) {
        if self.selection_toolbar_visible && !suppressed {
            browser_feature::selection_toolbar(ui, request, actions);
        }
    }
    /// Draw the browser in its dock and route pointer input in the document area.
    pub(crate) fn browser_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        presentation: (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &crate::icons::GuiRibbonIcons,
        ),
        part_context_open: bool,
        selector_context_gesture: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.pending_layout |= self.panels.visibility_changed_since_layout();
        self.view.arrow_consumed = false;
        self.view.feature_keyboard_focus = false;
        self.view.document_area = Some(presentation.0);
        self.view.icons = Some(presentation.1.clone());
        self.view.view_clicks_taken = self.view_clicks_taken;
        browser_origin::follow_view_click(
            ui,
            state,
            &mut self.view,
            part_context_open,
            selector_context_gesture,
            actions,
        );
        let focus = self.panels.take_tab_focus();
        if self.panels.model_visible() {
            let mut open = true;
            panel_tabs::ModelBrowserWindow::tabbed_dockable_window(ui, "Model", "Model", &mut open)
                .flags(WindowFlags::NO_FOCUS_ON_APPEARING)
                .focused(
                    focus == Some(panel_tabs::DockPanel::Model)
                        || (state.browser.is_some() && self.view.focus_search),
                )
                .build(|| {
                    crate::browser::browser_contents(ui, state, &mut self.view, actions);
                    self.pending_layout |= self.view.needs_frame();
                });
            self.panels.set_open(panel_tabs::DockPanel::Model, open);
        }
        if self.panels.log_open {
            let mut open = true;
            panel_tabs::LogPanelWindow::tabbed_dockable_window(ui, "Log", "Model", &mut open)
                .flags(WindowFlags::NO_FOCUS_ON_APPEARING)
                .focused(focus == Some(panel_tabs::DockPanel::Log))
                .build(|| self.panels.log_area(ui));
            self.panels.set_open(panel_tabs::DockPanel::Log, open);
        }
        if self.panels.favorites_visible() {
            let mut open = true;
            panel_tabs::FavoritesPanelWindow::tabbed_dockable_window(
                ui,
                "Favorites",
                "Model",
                &mut open,
            )
            .flags(WindowFlags::NO_FOCUS_ON_APPEARING)
            .focused(focus == Some(panel_tabs::DockPanel::Favorites))
            .build(|| self.panels.favorites.panel(ui));
            self.panels.set_open(panel_tabs::DockPanel::Favorites, open);
        }
        self.draw_find(ui, state, actions);
        self.draw_browser_dialogs(ui, state, actions);
    }
    /// Keep the panel menu available after the last browser tab closes.
    /// Example: `layout.empty_browser_menu(ui, document_area)` before rendering.
    pub(crate) fn empty_browser_menu(
        &self,
        ui: &Ui,
        area: WorkspaceArea,
    ) -> Option<crate::widget_abi::DockTabAction> {
        let panels = &self.panels;
        if panels.model_open || panels.log_open || panels.favorites_visible() {
            return None;
        }
        let _padding = ui.push_style_var(StyleVar::WindowPadding([3., 3.]));
        let _minimum = ui.push_style_var(StyleVar::WindowMinSize([1., 1.]));
        let side = ui.frame_height() + 6.;
        ui.window("##panel-plus")
            .position([area.left + 8., area.top + 4.], Condition::Always)
            .size([side, side], Condition::Always)
            .flags(panel_launcher_flags())
            .build(|| crate::widget_abi::workspace_panel_menu(ui))
            .flatten()
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
        browser_origin::draw_selected_origin(
            ui,
            request,
            self.view.selected_origin_row,
            shown,
            &self.view,
        );
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
                - self.status_height()
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
    /// Reserve space only while the status bar is visible; example: `layout.status_height()`.
    pub(crate) fn status_height(&self) -> f32 {
        if self.status_bar_visible {
            status_bar::STATUS_BAR_HEIGHT
        } else {
            0.
        }
    }
    /// Find the top of the bottom tab row; example: `layout.tab_top(900.)`.
    pub(crate) fn tab_top(&self, height: f32) -> f32 {
        let bottom = if self.outer_bottom_docked {
            height - self.status_height()
        } else {
            self.outer_center
                .map_or(height - self.status_height(), |area| area.bottom)
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
            ui.checkbox("ViewCube", &mut self.view_cube_visible);
            ui.checkbox("Navigation Bar", &mut self.navigation_bar_visible);
            ui.checkbox("Mini-Toolbar", &mut self.selection_toolbar_visible);
            ui.checkbox("Status Bar", &mut self.status_bar_visible);
            ui.checkbox("Marking Menu", &mut self.marking_menu_enabled);
            let mut model_open = self.panels.model_open;
            if ui.checkbox("Model Browser", &mut model_open) {
                self.panels
                    .set_open(panel_tabs::DockPanel::Model, model_open);
            }
            deferred_rule_panel_checkboxes(ui);
            let mut log_open = self.panels.log_open;
            if ui.checkbox("Log Panel", &mut log_open) {
                self.panels.set_open(panel_tabs::DockPanel::Log, log_open);
            }
            ui.checkbox("Document Tabs", &mut self.tabs_visible);
            let mut favorites = self.panels.favorites_visible();
            if ui.checkbox("Favorites", &mut favorites) {
                self.panels.toggle_favorites();
            }
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
fn deferred_rule_panel_checkboxes(ui: &Ui) {
    // Rule tools have no desktop workflow in this release (#4185).
    let _disabled = ui.begin_disabled(true);
    for label in ["Rule Log", "Rule Automation"] {
        ui.checkbox(label, &mut false);
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text("Rule panels are unavailable in this build.");
        }
    }
}

fn panel_launcher_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_DOCKING
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
}

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/layout.rs"]
mod tests;
