// SPDX-License-Identifier: GPL-2.0-only
//! Independent dock windows that ImGui combines into one native tab group.

const IDLE_STATUS: &str = "Ready";
const MESSAGE_CAPACITY: usize = 200;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
pub(crate) enum DockPanel {
    #[default]
    Model,
    Log,
    Favorites,
}

/// Window marker for the model browser's native ImGui dock tab.
pub(crate) struct ModelBrowserWindow;
impl crate::dockable_window::DockableWindow for ModelBrowserWindow {}

/// Window marker for application messages' native ImGui dock tab.
pub(crate) struct LogPanelWindow;
impl crate::dockable_window::DockableWindow for LogPanelWindow {}

/// Window marker for favorites' native ImGui dock tab.
pub(crate) struct FavoritesPanelWindow;
impl crate::dockable_window::DockableWindow for FavoritesPanelWindow {}

#[derive(Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct BrowserPanelSettings {
    model: bool,
    log: bool,
    favorites: bool,
    /// Retained for existing saved layouts; ImGui's ini now owns active-tab selection.
    #[serde(default)]
    active: DockPanel,
}

#[derive(Debug)]
pub(crate) struct DockPanels {
    pub model_open: bool,
    pub log_open: bool,
    favorites_open: bool,
    pending_tab_focus: Option<DockPanel>,
    laid_out_visibility: [bool; 3],
    pub favorites: crate::application::model_browser::favorites::FavoritesPane,
    model_dock: crate::dockable_lifecycle::DockableState,
    log_dock: crate::dockable_lifecycle::DockableState,
    favorites_dock: crate::dockable_lifecycle::DockableState,
    messages: Vec<String>,
}

impl Default for DockPanels {
    fn default() -> Self {
        Self {
            model_open: true,
            log_open: false,
            favorites_open: false,
            pending_tab_focus: None,
            laid_out_visibility: [true, false, false],
            favorites: Default::default(),
            model_dock: Default::default(),
            log_dock: Default::default(),
            favorites_dock: Default::default(),
            messages: Vec::new(),
        }
    }
}

impl DockPanels {
    /// Request a follow-up layout after a panel changes; example: opening Log returns true once.
    pub(crate) fn visibility_changed_since_layout(&mut self) -> bool {
        let current = [self.model_open, self.log_open, self.favorites_open];
        let changed = current != self.laid_out_visibility;
        self.laid_out_visibility = current;
        changed
    }

    pub(crate) fn settings(&self) -> BrowserPanelSettings {
        BrowserPanelSettings {
            model: self.model_open,
            log: self.log_open,
            favorites: self.favorites_open,
            active: DockPanel::Model,
        }
    }

    pub(crate) fn restore(&mut self, settings: &BrowserPanelSettings) {
        self.model_open = settings.model;
        self.log_open = settings.log;
        self.favorites_open = settings.favorites;
        let _active_tab_is_restored_from_imgui_ini = settings.active;
        self.sync_dock_states(false);
    }

    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<ModelBrowserWindow>("Model", &self.model_dock)?;
        registry.register::<LogPanelWindow>("Log", &self.log_dock)?;
        registry.register::<FavoritesPanelWindow>("Favorites", &self.favorites_dock)
    }

    pub(crate) fn sync_dock_states(&self, home_in_front: bool) {
        if self.model_open != self.model_dock.is_requested() {
            self.model_dock.set_open(self.model_open);
        }
        self.model_dock.set_suspended(home_in_front);
        self.log_dock.set_open(self.log_open);
        self.favorites_dock.set_open(self.favorites_open);
    }

    pub(crate) fn model_visible(&self) -> bool {
        self.model_dock.is_open()
    }

    pub(crate) fn open_panel(&mut self, panel: DockPanel) {
        self.set_open(panel, true);
        self.pending_tab_focus = Some(panel);
    }

    /// Consume focus requested by the panel menu, for example when selecting Log Panel.
    pub(crate) fn take_tab_focus(&mut self) -> Option<DockPanel> {
        self.pending_tab_focus.take()
    }

    pub(crate) fn set_open(&mut self, panel: DockPanel, open: bool) {
        match panel {
            DockPanel::Model => {
                self.model_open = open;
                self.model_dock.set_open(open);
            }
            DockPanel::Log => {
                self.log_open = open;
                self.log_dock.set_open(open);
            }
            DockPanel::Favorites => {
                if open && !self.favorites_open {
                    self.favorites.reset();
                }
                self.favorites_open = open;
                self.favorites_dock.set_open(open);
            }
        }
    }

    /// Toggle the Favorites pane from the interface menu; example: `panels.toggle_favorites()`.
    pub(crate) fn toggle_favorites(&mut self) {
        self.set_open(DockPanel::Favorites, !self.favorites_open);
    }

    /// Read the Favorites pane visibility; example: `panels.favorites_visible()`.
    pub(crate) fn favorites_visible(&self) -> bool {
        self.favorites_open
    }

    /// Record a status message for the log; an unchanged or idle status adds nothing.
    pub(crate) fn record_status(&mut self, status: Option<&str>) {
        let Some(text) = status.filter(|text| *text != IDLE_STATUS && !text.is_empty()) else {
            return;
        };
        if self.messages.last().is_some_and(|last| last == text) {
            return;
        }
        if self.messages.len() == MESSAGE_CAPACITY {
            self.messages.remove(0);
        }
        self.messages.push(text.to_string());
    }

    /// Draw application messages in the Log Panel window.
    pub(crate) fn log_area(&self, ui: &imgui::Ui) {
        if self.messages.is_empty() {
            ui.text_disabled("Application messages: none yet");
            return;
        }
        ui.text_disabled("Application messages");
        for message in &self.messages {
            ui.text_wrapped(message);
        }
    }
}

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/panel_tabs.rs"]
mod tests;
