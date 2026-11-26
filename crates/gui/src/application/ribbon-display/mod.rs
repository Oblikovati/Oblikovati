// SPDX-License-Identifier: GPL-2.0-only
//! How much of the ribbon stays open, and the temporary panel that shows what a mode hides.
use crate::ribbon::RibbonTab;
use crate::ribbon_panels::RibbonPanel;
use imgui::Ui;
use std::collections::BTreeSet;
mod appearance_menu;
mod context_bounds;
mod display_control;
use context_bounds::submenu_row_pressed;
pub(crate) use display_control::DENSITY_CONTROL_WIDTH;
mod docking_menu;
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
    /// The band height this mode leaves the workspace; example: `mode.band_height()`.
    pub(crate) fn band_height(self) -> f32 {
        match self {
            Self::Full => 158.,
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
    pub appearance: crate::ribbon_command_layout::RibbonAppearance,
    /// `None` cycles through every mode; a choice alternates between it and the full ribbon.
    minimized: Option<RibbonDisplay>,
    pub temporary: Option<TemporaryPanel>,
    pub temporary_focus: bool,
    control_press: crate::popup_press::PopupPress<()>,
    menu_press: crate::popup_press::PopupPress<()>,
    selector_press: crate::popup_press::PopupPress<RibbonPlacement>,
    pending_keyboard_group: Option<RibbonPanel>,
    pub temporary_escape_consumed: bool,
    pub keep_temporary_on_click: bool,
    pub temporary_anchor: f32,
    tab_release_consumed: bool,
    pub placement: RibbonPlacement,
    pub side_anchor: Option<SideAnchor>,
    context_requested: bool,
    context_title: Option<RibbonPanel>,
    pub(crate) retain_appearance_on_escape: bool,
    context_relocated: Option<(RibbonTab, RibbonPanel)>,
    context_expansion: Option<(RibbonTab, RibbonPanel)>,
    context_command: Option<&'static str>,
    context_had_submenu: bool,
    context_bounds: Option<[[f32; 2]; 2]>,
    submenu_bounds: Option<[[f32; 2]; 2]>,
    docking_selected: bool,
    docking_open: bool,
    docking_hover_blocked: bool,
    customize_requested: bool,
    reset_requested: bool,
    /// A choice in Show Panels ends the whole menu, as a choice in any other row does.
    close_context_menu: bool,
    close_unpinned_expansion: bool,
    selector_keyboard_started: bool,
    selector_focused_panel: Option<RibbonPanel>,
    display_menu_cursor: Option<usize>,
    pub(crate) selector_open: bool,
    pub(crate) selector_pointer_inside: bool,
    pub(crate) selector_trigger: Option<[[f32; 2]; 2]>,
    pub(crate) selector_escape: bool,
    /// Panels a user has hidden, per tab; they stay listed in Show Panels.
    pub hidden_panels: BTreeSet<(RibbonTab, RibbonPanel)>,
    pub show_panel_titles: bool,
}
impl Default for RibbonDisplayState {
    fn default() -> Self {
        Self {
            mode: RibbonDisplay::Full,
            appearance: Default::default(),
            minimized: Some(RibbonDisplay::Tabs),
            temporary: None,
            temporary_focus: false,
            control_press: Default::default(),
            menu_press: Default::default(),
            selector_press: Default::default(),
            pending_keyboard_group: None,
            temporary_escape_consumed: false,
            keep_temporary_on_click: false,
            temporary_anchor: 8.,
            tab_release_consumed: false,
            placement: RibbonPlacement::default(),
            side_anchor: None,
            context_requested: false,
            context_title: None,
            retain_appearance_on_escape: false,
            context_relocated: None,
            context_expansion: None,
            context_command: None,
            context_had_submenu: false,
            context_bounds: None,
            submenu_bounds: None,
            docking_selected: false,
            docking_open: false,
            docking_hover_blocked: false,
            customize_requested: false,
            reset_requested: false,
            close_context_menu: false,
            close_unpinned_expansion: false,
            selector_keyboard_started: false,
            selector_focused_panel: None,
            display_menu_cursor: None,
            selector_open: false,
            selector_pointer_inside: false,
            selector_trigger: None,
            selector_escape: false,
            // Parameters lives on Manage, and is offered here for those who want it beside Create.
            hidden_panels: BTreeSet::from([
                (RibbonTab::Model, RibbonPanel::PartPrimitives),
                (RibbonTab::Model, RibbonPanel::Parameters),
                (RibbonTab::Sketch, RibbonPanel::Layout),
                (RibbonTab::Sketch, RibbonPanel::Parameters),
                (RibbonTab::Tools, RibbonPanel::ToolTutorials),
            ]),
            show_panel_titles: true,
        }
    }
}
impl RibbonDisplayState {
    /// Expand a new tab after an open compact panel; for example, Create to Annotate (#3856).
    pub(crate) fn switch_from_open_compact_panel(&mut self) {
        if self.mode == RibbonDisplay::PanelButtons
            && matches!(self.temporary, Some(TemporaryPanel::Group(_)))
        {
            self.temporary = None;
            self.mode = RibbonDisplay::Full;
        }
    }
    pub(crate) fn open_temporary(&mut self, panel: TemporaryPanel) {
        if self.temporary != Some(panel) {
            self.temporary_focus = true;
        }
        self.temporary = Some(panel);
    }
    pub(crate) fn request_keyboard_group(&mut self, panel: RibbonPanel) {
        self.pending_keyboard_group = Some(panel);
    }
    pub(crate) fn take_keyboard_group(&mut self, panel: RibbonPanel) -> bool {
        if self.pending_keyboard_group != Some(panel) {
            return false;
        }
        self.pending_keyboard_group = None;
        true
    }
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
    /// Move the ribbon; a temporary panel opened for the old place closes.
    pub(crate) fn place(&mut self, placement: RibbonPlacement) {
        self.placement = placement;
        self.docking_selected = true;
        self.temporary = None;
        self.side_anchor = None;
    }
    fn toggle_panel(&mut self, tab: RibbonTab, panel: RibbonPanel) {
        if !self.hidden_panels.remove(&(tab, panel)) {
            self.hidden_panels.insert((tab, panel));
            if self.temporary == Some(TemporaryPanel::Group(panel)) {
                self.temporary = None;
                self.side_anchor = None;
            }
        }
    }
    /// Opened by a click, not by hovering: Escape closes this list alone, and a pointer still
    /// resting on the row must not reopen it on the next frame.
    fn show_panels_menu(
        &mut self,
        ui: &Ui,
        tab: RibbonTab,
        panels: &[RibbonPanel],
        submenu_open: bool,
    ) {
        if submenu_row_pressed(ui, "Show Panels", submenu_open) {
            ui.open_popup(SHOW_PANELS);
        }
        let at = [ui.item_rect_max()[0], ui.item_rect_min()[1]];
        crate::widget_abi::next_window_position(ui, at);
        let Some(_list) = ui.begin_popup(SHOW_PANELS) else {
            return;
        };
        self.capture_submenu_bounds(ui);
        for panel in panels {
            let shown = !self.hidden_panels.contains(&(tab, *panel));
            if ui.menu_item_config(panel.label()).selected(shown).build() {
                self.toggle_panel(tab, *panel);
                self.finish_title_context_selection();
            }
        }
    }
    /// Draw the context menu that any of the ribbon's windows can ask for.
    pub(crate) fn context_menu(
        &mut self,
        ui: &Ui,
        tab: RibbonTab,
        panels: &[RibbonPanel],
        customizable: bool,
        customization: &mut crate::application::ribbon_customization::RibbonCustomization,
        toolbar: (
            &crate::GuiCommandState,
            &mut crate::application::quick_access_toolbar::QuickAccessToolbar,
        ),
    ) {
        self.retain_appearance_on_escape = false;
        let context_opened = std::mem::take(&mut self.context_requested);
        if context_opened {
            ui.open_popup(CONTEXT_MENU);
        }
        let Some(_menu) = ui.begin_popup(CONTEXT_MENU) else {
            self.context_title = None;
            self.context_expansion = None;
            self.context_command = None;
            self.context_had_submenu = false;
            self.context_bounds = None;
            self.submenu_bounds = None;
            self.docking_open = false;
            self.docking_hover_blocked = false;
            return;
        };
        let submenu_open = self.submenu_bounds.is_some();
        self.capture_context_bounds(ui);
        if context_opened {
            ui.set_keyboard_focus_here();
        }
        self.appearance_menu(ui, submenu_open, customization);
        if self.context_command.is_some() {
            self.expanded_command_menu(ui, customization, toolbar);
        }
        self.show_panels_menu(ui, tab, panels, submenu_open);
        if ui
            .menu_item_config("Show Panel Titles")
            .selected(self.show_panel_titles)
            .build()
        {
            self.show_panel_titles = !self.show_panel_titles;
        }
        if ui
            .menu_item_config("Customize User Commands")
            .enabled(customizable)
            .build()
        {
            self.customize_requested = true;
        }
        self.docking_menu(ui);
        if std::mem::take(&mut self.close_context_menu) {
            ui.close_current_popup();
        }
        let floating = self.placement == RibbonPlacement::Floating;
        if ui
            .menu_item_config("Undock Ribbon")
            .enabled(!floating)
            .build()
        {
            self.place(RibbonPlacement::Floating);
        }
    }
    pub(crate) fn take_customization_request(&mut self) -> bool {
        std::mem::take(&mut self.customize_requested)
    }
    /// Open ribbon reset confirmation; example: `display.request_reset()`.
    pub(crate) fn request_reset(&mut self) {
        self.reset_requested = true;
    }
    /// Consume a reset request; example: `display.take_reset_request()`.
    pub(crate) fn take_reset_request(&mut self) -> bool {
        std::mem::take(&mut self.reset_requested)
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
    /// The selector dismisses on press, so the item beneath it must receive that press too.
    pub(crate) fn selector_pointer_press(&self, ui: &Ui) -> bool {
        Self::selector_pointer_press_when(ui, self.selector_open)
    }
    /// Check the prior selector state before ImGui closes it on this press.
    /// Example: `RibbonDisplayState::selector_pointer_press_when(ui, was_open)`.
    pub(crate) fn selector_pointer_press_when(ui: &Ui, selector_open: bool) -> bool {
        if !selector_open || !ui.is_mouse_clicked(imgui::MouseButton::Left) {
            return false;
        }
        let [left, top] = ui.item_rect_min();
        let [right, bottom] = ui.item_rect_max();
        let [x, y] = ui.io().mouse_pos;
        (left..=right).contains(&x) && (top..=bottom).contains(&y)
    }
    fn menu(&mut self, ui: &Ui) {
        let selected = self.display_menu_keyboard(ui);
        for (index, choice) in [
            RibbonDisplay::Tabs,
            RibbonDisplay::PanelTitles,
            RibbonDisplay::PanelButtons,
        ]
        .into_iter()
        .enumerate()
        {
            if self.display_menu_cursor == Some(index) {
                ui.set_keyboard_focus_here();
            }
            let label = choice.menu_label().unwrap_or_default();
            let checked = self.minimized == Some(choice);
            if ui.menu_item_config(label).selected(checked).build() || selected == Some(index) {
                self.choose(choice);
                ui.close_current_popup();
            }
        }
        let cycling = self.minimized.is_none();
        if self.display_menu_cursor == Some(3) {
            ui.set_keyboard_focus_here();
        }
        if ui
            .menu_item_config("Cycle through All")
            .selected(cycling)
            .build()
            || selected == Some(3)
        {
            self.cycle_through_all();
            ui.close_current_popup();
        }
    }
    fn display_menu_keyboard(&mut self, ui: &Ui) -> Option<usize> {
        if ui.is_mouse_clicked(imgui::MouseButton::Left) {
            self.display_menu_cursor = None;
        }
        if ui.is_key_pressed_no_repeat(imgui::Key::DownArrow) {
            self.display_menu_cursor = Some(
                self.display_menu_cursor
                    .map_or(0, |index| (index + 1).min(3)),
            );
        } else if ui.is_key_pressed_no_repeat(imgui::Key::UpArrow) {
            self.display_menu_cursor = Some(
                self.display_menu_cursor
                    .map_or(3, |index| index.saturating_sub(1)),
            );
        }
        (ui.is_key_pressed_no_repeat(imgui::Key::Enter)
            || ui.is_key_pressed_no_repeat(imgui::Key::KeypadEnter))
        .then_some(self.display_menu_cursor)
        .flatten()
    }
}
#[cfg(test)]
#[path = "../../../tests/application/ribbon-display/modes.rs"]
mod tests;
