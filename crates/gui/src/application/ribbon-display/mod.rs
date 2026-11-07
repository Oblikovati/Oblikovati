// SPDX-License-Identifier: GPL-2.0-only
//! How much of the ribbon stays open, and the temporary panel that shows what a mode hides.
use crate::ribbon::RibbonTab;
use crate::ribbon_panels::RibbonPanel;
use imgui::Ui;
use std::collections::BTreeSet;
mod panel_selector;
pub(crate) use panel_selector::PANEL_SELECTOR_WIDTH;

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
/// Where the ribbon sits around the workspace.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum RibbonPlacement {
    #[default]
    Top,
    Left,
    Right,
    Floating,
}
/// Where a side ribbon's temporary panel opens: beside the group button that asked for it.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct SideAnchor {
    pub x: f32,
    pub y: f32,
    /// The panel ends at `x` rather than starting there, on a right-docked ribbon.
    pub ends_at_x: bool,
}
const CONTEXT_MENU: &str = "##ribbon-context";
const SHOW_PANELS: &str = "##show-panels";
const DOCKING_POSITIONS: [(&str, RibbonPlacement); 3] = [
    ("Top", RibbonPlacement::Top),
    ("Left", RibbonPlacement::Left),
    ("Right", RibbonPlacement::Right),
];
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
    pub temporary_anchor: f32,
    tab_release_consumed: bool,
    pub placement: RibbonPlacement,
    pub side_anchor: Option<SideAnchor>,
    context_requested: bool,
    /// A choice in Show Panels ends the whole menu, as a choice in any other row does.
    close_context_menu: bool,
    /// Panels a user has hidden, per tab; they stay listed in Show Panels.
    pub hidden_panels: BTreeSet<(RibbonTab, RibbonPanel)>,
    pub show_panel_titles: bool,
}
impl Default for RibbonDisplayState {
    fn default() -> Self {
        Self {
            mode: RibbonDisplay::Full,
            minimized: None,
            temporary: None,
            temporary_anchor: 8.,
            tab_release_consumed: false,
            placement: RibbonPlacement::default(),
            side_anchor: None,
            context_requested: false,
            close_context_menu: false,
            // Parameters lives on Manage, and is offered here for those who want it beside Create.
            hidden_panels: BTreeSet::from([(RibbonTab::Model, RibbonPanel::Parameters)]),
            show_panel_titles: true,
        }
    }
}
impl RibbonDisplayState {
    /// What the display control will do next, for its tooltip.
    fn control_tooltip(&self) -> &'static str {
        let mut next = RibbonDisplayState {
            mode: self.mode,
            minimized: self.minimized,
            ..Default::default()
        };
        next.advance();
        next.mode.menu_label().unwrap_or("Show Full Ribbon")
    }
    /// Ask for the context menu from any of the ribbon's windows.
    pub(crate) fn request_context_menu(&mut self) {
        self.context_requested = true;
    }
    /// Move the ribbon; a temporary panel opened for the old place closes.
    pub(crate) fn place(&mut self, placement: RibbonPlacement) {
        self.placement = placement;
        self.temporary = None;
        self.side_anchor = None;
    }
    fn toggle_panel(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        if !self.hidden_panels.remove(&(tab, panel)) {
            self.hidden_panels.insert((tab, panel));
        }
    }
    /// Opened by a click, not by hovering: Escape closes this list alone, and a pointer still
    /// resting on the row must not reopen it on the next frame.
    fn show_panels_menu(&mut self, ui: &Ui, tab: RibbonTab, panels: &[RibbonPanel]) {
        if crate::widget_abi::dialog_selectable(ui, "Show Panels", false) {
            ui.open_popup(SHOW_PANELS);
        }
        let at = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        crate::widget_abi::next_window_position(ui, at);
        let Some(_list) = ui.begin_popup(SHOW_PANELS) else {
            return;
        };
        for panel in panels {
            let shown = !self.hidden_panels.contains(&(tab, *panel));
            if ui.menu_item_config(panel.label()).selected(shown).build() {
                self.toggle_panel(tab, *panel);
                self.close_context_menu = true;
            }
        }
    }
    /// Draw the context menu that any of the ribbon's windows can ask for.
    pub(crate) fn context_menu(&mut self, ui: &Ui, tab: RibbonTab, panels: &[RibbonPanel]) {
        if std::mem::take(&mut self.context_requested) {
            ui.open_popup(CONTEXT_MENU);
        }
        let Some(_menu) = ui.begin_popup(CONTEXT_MENU) else {
            return;
        };
        muted_context_item(ui, "Ribbon Appearance");
        self.show_panels_menu(ui, tab, panels);
        if ui
            .menu_item_config("Show Panel Titles")
            .selected(self.show_panel_titles)
            .build()
        {
            self.show_panel_titles = !self.show_panel_titles;
        }
        if std::mem::take(&mut self.close_context_menu) {
            ui.close_current_popup();
        }
        muted_context_item(ui, "Customize User Commands");
        ui.menu("Docking Positions", || {
            for (label, placement) in DOCKING_POSITIONS {
                let checked = self.placement == placement;
                if ui.menu_item_config(label).selected(checked).build() {
                    self.place(placement);
                }
            }
        });
        let floating = self.placement == RibbonPlacement::Floating;
        if ui
            .menu_item_config("Undock Ribbon")
            .enabled(!floating)
            .build()
        {
            self.place(RibbonPlacement::Floating);
        }
    }
    /// Dock a floating ribbon from its title-bar menu.
    pub(crate) fn floating_title_menu(&mut self, ui: &Ui) {
        let [left, top] = ui.window_pos();
        let right = left + ui.window_size()[0];
        let bottom = top + ui.frame_height();
        let [x, y] = ui.io().mouse_pos;
        let title_hovered =
            x >= left && x < right && y >= top && y < bottom && ui.is_window_hovered();
        if title_hovered && ui.is_mouse_clicked(imgui::MouseButton::Right) {
            ui.open_popup("##floating-ribbon-dock");
        }
        let Some(_menu) = ui.begin_popup("##floating-ribbon-dock") else {
            return;
        };
        for (label, position) in [
            ("Dock to Top", RibbonPlacement::Top),
            ("Dock to Left", RibbonPlacement::Left),
            ("Dock to Right", RibbonPlacement::Right),
        ] {
            if ui.menu_item(label) {
                self.place(position);
            }
        }
    }
    /// A double click on the active tab acts like the display control. Its own release must not
    /// then open the whole-tab panel that a plain tab click opens in tabs-only mode.
    pub(crate) fn advance_from_tab(&mut self) {
        self.advance();
        self.tab_release_consumed = true;
    }
    /// Whether this tab release opens the whole-tab panel; example: in tabs-only mode.
    pub(crate) fn take_tab_release(&mut self) -> bool {
        !std::mem::take(&mut self.tab_release_consumed) && self.mode == RibbonDisplay::Tabs
    }
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
        if ui.is_item_hovered() {
            ui.tooltip_text(self.control_tooltip());
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
fn muted_context_item(ui: &Ui, label: &str) {
    ui.menu_item_config(label).enabled(false).build();
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("This command is not available in this build.");
    }
}
