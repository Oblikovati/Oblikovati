// SPDX-License-Identifier: GPL-2.0-only
//! How much of the ribbon stays open, and the temporary panel that shows what a mode hides.
use crate::ribbon_panels::RibbonPanel;
use imgui::Ui;

const CYCLE: [RibbonDisplay; 4] = [
    RibbonDisplay::Full,
    RibbonDisplay::PanelButtons,
    RibbonDisplay::PanelTitles,
    RibbonDisplay::Tabs,
];
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum RibbonDisplay {
    #[default]
    Full,
    PanelButtons,
    PanelTitles,
    Tabs,
}
impl RibbonDisplay {
    /// The band height this mode leaves the workspace; example: `mode.band_height(0.)`.
    pub(crate) fn band_height(self, scrollbar: f32) -> f32 {
        match self {
            Self::Full => 158. + scrollbar,
            Self::PanelButtons => 122.,
            Self::PanelTitles => 102.,
            Self::Tabs => 80.,
        }
    }
    /// The size of a group's button; only a compact mode has them.
    pub(crate) fn group_button(self) -> Option<[f32; 2]> {
        match self {
            Self::PanelButtons => Some([108., 36.]),
            Self::PanelTitles => Some([108., 18.]),
            Self::Full | Self::Tabs => None,
        }
    }
    fn menu_label(self) -> Option<&'static str> {
        match self {
            Self::Tabs => Some("Minimize to Tabs"),
            Self::PanelTitles => Some("Minimize to Panel Titles"),
            Self::PanelButtons => Some("Minimize to Panel Buttons"),
            Self::Full => None,
        }
    }
}
/// What the temporary panel below the ribbon shows.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum TemporaryPanel {
    Group(RibbonPanel),
    WholeTab,
}
#[derive(Debug)]
pub(crate) struct RibbonDisplayState {
    pub mode: RibbonDisplay,
    /// `None` cycles through every mode; a choice alternates between it and the full ribbon.
    minimized: Option<RibbonDisplay>,
    pub temporary: Option<TemporaryPanel>,
}
impl Default for RibbonDisplayState {
    fn default() -> Self {
        Self {
            mode: RibbonDisplay::Full,
            minimized: None,
            temporary: None,
        }
    }
}
impl RibbonDisplayState {
    fn advance(&mut self) {
        self.temporary = None;
        self.mode = match self.minimized {
            None => {
                let at = CYCLE
                    .iter()
                    .position(|mode| *mode == self.mode)
                    .unwrap_or(0);
                CYCLE[(at + 1) % CYCLE.len()]
            }
            Some(choice) if self.mode == choice => RibbonDisplay::Full,
            Some(choice) => choice,
        };
    }
    fn choose(&mut self, choice: RibbonDisplay) {
        self.temporary = None;
        self.minimized = Some(choice);
        self.mode = choice;
    }
    fn cycle_through_all(&mut self) {
        self.minimized = None;
    }
    /// Draw the display control and its menu at `position`; example: `state.control(ui, [900., 35.])`.
    pub(crate) fn control(&mut self, ui: &Ui, position: [f32; 2]) {
        crate::widget_abi::cursor_position(ui, position);
        if crate::widget_abi::button(ui, "Ribbon Display", [0.; 2]) {
            self.advance();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "v##ribbon-display-menu", [0.; 2]) {
            ui.open_popup("##ribbon-display-menu");
        }
        if let Some(_menu) = ui.begin_popup("##ribbon-display-menu") {
            self.menu(ui);
        }
    }
    fn menu(&mut self, ui: &Ui) {
        for choice in [
            RibbonDisplay::Tabs,
            RibbonDisplay::PanelTitles,
            RibbonDisplay::PanelButtons,
        ] {
            let label = choice.menu_label().unwrap_or_default();
            let checked = self.minimized == Some(choice);
            if ui.menu_item_config(label).selected(checked).build() {
                self.choose(choice);
            }
        }
        let cycling = self.minimized.is_none();
        if ui
            .menu_item_config("Cycle through All")
            .selected(cycling)
            .build()
        {
            self.cycle_through_all();
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/ribbon-display/modes.rs"]
mod tests;
