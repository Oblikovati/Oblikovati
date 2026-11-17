// SPDX-License-Identifier: GPL-2.0-only
//! The panels a browser dock can hold, its tabs, and the plus menu that reopens them.
use imgui::{TabItemFlags, Ui};

const PLUS_MENU: &str = "##panel-plus-menu";
const MODEL_BROWSER: &str = "Model Browser";
const LOG_PANEL: &str = "Log Panel";
const FAVORITES_PANEL: &str = "Favorites";
const MODEL_TAB: &str = "Model";
const LOG_TAB: &str = "Log";
const NO_MESSAGES: &str = "Application messages: none yet";
const MESSAGES_HEADING: &str = "Application messages";
/// A status line a user never needs to read back.
const IDLE_STATUS: &str = "Ready";
const MESSAGE_CAPACITY: usize = 200;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
pub(crate) enum DockPanel {
    #[default]
    Model,
    Log,
    Favorites,
}
#[derive(Debug, serde::Deserialize, serde::Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct BrowserPanelSettings {
    model: bool,
    log: bool,
    favorites: bool,
    active: DockPanel,
}
/// What the dock shows; `model_open` is the View tab's Model Browser option.
#[derive(Debug)]
pub(crate) struct DockPanels {
    pub model_open: bool,
    log_open: bool,
    favorites_open: bool,
    pub favorites: crate::application::model_browser::favorites::FavoritesPane,
    active: DockPanel,
    /// A panel just added or left alone takes the tab bar's selection on the next frame.
    select: Option<DockPanel>,
    messages: Vec<String>,
}
impl Default for DockPanels {
    fn default() -> Self {
        Self {
            model_open: true,
            log_open: false,
            favorites_open: false,
            favorites: Default::default(),
            active: DockPanel::Model,
            select: None,
            messages: Vec::new(),
        }
    }
}
impl DockPanels {
    pub(crate) fn settings(&self) -> BrowserPanelSettings {
        BrowserPanelSettings {
            model: self.model_open,
            log: self.log_open,
            favorites: self.favorites_open,
            active: self.active,
        }
    }
    pub(crate) fn restore(&mut self, settings: &BrowserPanelSettings) {
        self.model_open = settings.model;
        self.log_open = settings.log;
        self.favorites_open = settings.favorites;
        self.select = match settings.active {
            DockPanel::Model if settings.model => Some(DockPanel::Model),
            DockPanel::Log if settings.log => Some(DockPanel::Log),
            DockPanel::Favorites if settings.favorites => Some(DockPanel::Favorites),
            _ => self.first_open_panel(),
        };
    }
    pub(crate) fn dock_visible(&self) -> bool {
        self.model_open || self.log_open || self.favorites_open
    }
    pub(crate) fn close_all(&mut self) {
        self.model_open = false;
        self.log_open = false;
        self.favorites_open = false;
        self.select = None;
    }
    /// A frame must follow so a changed selection takes effect without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.select.is_some()
    }
    fn open_panel(&mut self, panel: DockPanel) {
        match panel {
            DockPanel::Model => self.model_open = true,
            DockPanel::Log => self.log_open = true,
            DockPanel::Favorites => {
                self.favorites_open = true;
                self.favorites.reset();
            }
        }
        self.select = Some(panel);
    }
    /// The other panel takes over; with none left the dock closes and the plus control remains.
    fn close_panel(&mut self, panel: DockPanel) {
        match panel {
            DockPanel::Model => self.model_open = false,
            DockPanel::Log => self.log_open = false,
            DockPanel::Favorites => self.favorites_open = false,
        }
        self.select = self.first_open_panel();
    }
    fn first_open_panel(&self) -> Option<DockPanel> {
        [
            (self.model_open, DockPanel::Model),
            (self.log_open, DockPanel::Log),
            (self.favorites_open, DockPanel::Favorites),
        ]
        .into_iter()
        .find_map(|(open, panel)| open.then_some(panel))
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
    /// Draw the plus control and its list; example: at the top of the dock, or alone when it is empty.
    pub(crate) fn plus_control(&mut self, ui: &Ui) {
        if crate::widget_abi::button(ui, "+", [0.; 2]) {
            ui.open_popup(PLUS_MENU);
        }
        let Some(_menu) = ui.begin_popup(PLUS_MENU) else {
            return;
        };
        if ui
            .menu_item_config(MODEL_BROWSER)
            .selected(self.model_open)
            .build()
        {
            self.open_panel(DockPanel::Model);
        }
        if ui
            .menu_item_config(LOG_PANEL)
            .selected(self.log_open)
            .build()
        {
            self.open_panel(DockPanel::Log);
        }
        if ui
            .menu_item_config(FAVORITES_PANEL)
            .selected(self.favorites_open)
            .build()
        {
            self.open_panel(DockPanel::Favorites);
        }
    }
    /// Draw the tabs and answer which panel fills the dock this frame.
    pub(crate) fn tabs(&mut self, ui: &Ui) -> DockPanel {
        let select = self.select.take();
        if let Some(panel) = select {
            self.active = panel;
        }
        let Some(_bar) = ui.tab_bar("##dock-panels") else {
            return self.active;
        };
        if self.model_open {
            self.tab(ui, MODEL_TAB, DockPanel::Model, select);
        }
        if self.log_open {
            self.tab(ui, LOG_TAB, DockPanel::Log, select);
        }
        if self.favorites_open {
            self.tab(ui, FAVORITES_PANEL, DockPanel::Favorites, select);
        }
        self.active
    }
    fn tab(&mut self, ui: &Ui, label: &str, panel: DockPanel, select: Option<DockPanel>) {
        let flags = if select == Some(panel) {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        let mut keep = true;
        let contents = ui.tab_item_with_flags(label, Some(&mut keep), flags);
        // A tab that is about to be selected must not report the old one as chosen.
        if contents.is_some() && select.is_none() {
            self.active = panel;
        }
        drop(contents);
        if !keep {
            self.close_panel(panel);
        }
    }
    /// Draw the log area; example: when the log tab is active.
    pub(crate) fn log_area(&self, ui: &Ui) {
        if self.messages.is_empty() {
            ui.text_disabled(NO_MESSAGES);
            return;
        }
        ui.text_disabled(MESSAGES_HEADING);
        for message in &self.messages {
            ui.text_wrapped(message);
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/panel_tabs.rs"]
mod tests;
