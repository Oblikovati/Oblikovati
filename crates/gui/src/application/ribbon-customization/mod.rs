// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon command edits stay in a draft until Apply or OK.
mod builtin;
mod category;
pub(crate) mod interface_export;
mod keyboard;
mod keyboard_catalog;
mod layout;
mod locations;
mod marking_environment;
mod marking_menu;
mod marking_overflow;
mod panel;
mod settings_dialog;
mod settings_xml;
mod state;
mod store;
mod target_selector;
use crate::dockable_window::DockableWindow;
use crate::ribbon::RibbonTab;
use category::Category;
use imgui::{FocusedWidget, Ui};
use state::{Command, Draft};
pub use store::GuiRibbonSettingsStore;

#[derive(Clone, Copy)]
enum ListFocus {
    Source,
    Target,
}
#[derive(Default)]
pub(crate) struct RibbonCustomization {
    open: crate::dockable_lifecycle::DockableState,
    ribbon_press: crate::popup_press::PopupPress<RibbonTab>,
    draft: Draft,
    builtins: builtin::BuiltinLayout,
    appearance: crate::ribbon_command_layout::RibbonAppearance,
    staged_appearance: crate::ribbon_command_layout::RibbonAppearance,
    escape: crate::application::document_saving::folder_browser::DialogEscape,
    pub(crate) escape_consumed: bool,
    pub(crate) navigation_consumed: bool,
    list_focus: Option<ListFocus>,
    source_focused: bool,
    target_focused: bool,
    category_open: bool,
    category_cursor: usize,
    target_selector_open: bool,
    target_cursor: usize,
    visible_source: Vec<Command>,
    visible_target_rows: usize,
    scroll_source_to_selection: bool,
    scroll_target_to_selection: bool,
    settings: settings_dialog::SettingsDialog,
    store: Option<Box<dyn GuiRibbonSettingsStore>>,
    settings_error: Option<String>,
    command_error: Option<String>,
    command_error_opening: bool,
    keyboard_search: String,
    keyboard_category: usize,
    keyboard_kind: usize,
    keyboard_draft: Option<oblikovati_model::keymap::Keymap>,
    keyboard_base: Option<oblikovati_model::keymap::KeymapPreferences>,
    keyboard_selected: Option<String>,
    keyboard_chord: String,
    keyboard_alias: String,
    keyboard_error: Option<String>,
    close_after_apply: bool,
    keyboard_conflict: bool,
    marking_draft: Option<oblikovati_model::marking_menu::MarkingMenus>,
    marking_base: Option<oblikovati_model::marking_menu::MarkingMenus>,
    marking_environment: Option<oblikovati_api::types::Environment>,
    marking_display_draft: oblikovati_model::marking_menu::MarkingMenuDisplay,
    marking_display_base: oblikovati_model::marking_menu::MarkingMenuDisplay,
    marking_selector_open: bool,
    marking_conflict: bool,
    ribbon_tab_active: bool,
    pub(crate) pending_zoom: Option<crate::application::viewport_orientation::ZoomAction>,
}
impl DockableWindow for RibbonCustomization {}
impl RibbonCustomization {
    /// Open from the ribbon button, including a click that dismisses a popup.
    pub(crate) fn ribbon_button(&mut self, ui: &Ui, tab: RibbonTab, clicked: bool) {
        if self.ribbon_press.button(ui, tab, clicked) {
            self.open(tab);
        }
    }

    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Customize", &self.open)
    }
    pub(crate) fn open(&mut self, tab: RibbonTab) {
        self.draft.open(tab);
        self.builtins.open();
        self.staged_appearance = self.appearance;
        self.keyboard_draft = None;
        self.keyboard_base = None;
        self.keyboard_selected = None;
        self.keyboard_error = None;
        self.close_after_apply = false;
        self.keyboard_conflict = false;
        self.marking_draft = None;
        self.marking_base = None;
        self.marking_environment = None;
        self.marking_conflict = false;
        self.marking_selector_open = false;
        self.escape = Default::default();
        self.list_focus = None;
        self.category_open = false;
        self.target_selector_open = false;
        self.scroll_source_to_selection = false;
        self.scroll_target_to_selection = false;
        self.settings_error = None;
        self.open.set_open(true);
    }
    pub(crate) fn has_panel(&self, tab: RibbonTab) -> bool {
        self.draft
            .applied(tab)
            .iter()
            .any(|entry| entry.command != Command::Separator)
    }
    pub(crate) fn expanded_command_count(&self, tab: RibbonTab) -> usize {
        self.draft
            .applied(tab)
            .iter()
            .filter(|entry| entry.expanded)
            .count()
    }
    pub(crate) fn has_expanded(&self, tab: RibbonTab) -> bool {
        self.draft
            .applied(tab)
            .iter()
            .any(|entry| entry.expanded && entry.command != Command::Separator)
    }
    fn command_error_popup(&mut self, ui: &Ui) {
        let Some(error) = &self.command_error else {
            return;
        };
        if std::mem::take(&mut self.command_error_opening) {
            ui.open_popup("Ribbon Settings Error");
        }
        crate::application::document_saving::folder_browser::center_dialog(ui, [600., 160.]);
        let Some(_popup) = ui.begin_modal_popup("Ribbon Settings Error") else {
            return;
        };
        ui.text_wrapped(error);
        if ui.button("Close") || ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.command_error = None;
            ui.close_current_popup();
        }
    }
    pub(crate) fn sync_before_docking(&mut self, commands: &crate::GuiCommandState) {
        if !self.open.is_open() {
            return;
        }
        self.sync_keyboard(commands);
        self.sync_marking(commands);
        self.finish_apply();
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        commands: &crate::GuiCommandState,
        actions: &mut Vec<crate::GuiShellAction>,
        export: interface_export::InterfaceExport<'_>,
    ) {
        self.escape_consumed = false;
        self.navigation_consumed = false;
        self.command_error_popup(ui);
        if !self.open.is_open() {
            return;
        }
        self.sync_before_docking(commands);
        if !self.open.is_open() {
            return;
        }
        layout::fit_window(ui);
        let mut window_open = self.open.is_open();
        let category_was_open = self.category_open;
        let target_was_open = self.target_selector_open;
        let marking_selector_was_open = self.marking_selector_open;
        Self::dockable_window(ui, "Customize", &mut window_open)
            .build(|| {
                self.tabs(ui);
                if let Some(error) = &self.settings_error {
                    ui.text_wrapped(error);
                }
                if self.keyboard_conflict {
                    ui.text_wrapped("Keyboard settings changed outside this dialog. Close and reopen Customize to edit the current settings.");
                }
                if self.marking_conflict { ui.text_wrapped("Marking menu changed outside this dialog. Close and reopen Customize to edit the current menu."); }
                self.buttons(ui, actions);
                let focused = ui.is_window_focused_with_flags(
                    imgui::WindowFocusedFlags::ROOT_AND_CHILD_WINDOWS,
                );
                if self.ribbon_tab_active {
                    self.list_keyboard(ui, focused);
                }
                let selector_was_open = category_was_open || target_was_open;
                let close_dialog = self.escape.pressed(ui)
                    || (selector_was_open && ui.is_key_pressed_no_repeat(imgui::Key::Escape));
                let dismiss_selector = marking_selector_was_open || target_was_open && !self.has_pending_changes();
                if close_dialog && dismiss_selector {
                    self.escape_consumed = true;
                } else if close_dialog && (focused || selector_was_open) {
                    self.open.set_open(false);
                    self.escape_consumed = true;
                }
            });
        if !window_open {
            self.open.set_open(false);
        }
        self.settings_file_picker(ui, || export.sections(commands));
    }
    fn source_list(&mut self, ui: &Ui) {
        let size = layout::list_size(ui, 230., 150.);
        ui.text("Available commands");
        ui.set_next_item_width(size[0]);
        let category = ui.begin_combo("##command-category", self.draft.category.label());
        if category.is_some() && !self.category_open {
            self.category_cursor = category::category_index(self.draft.category);
        }
        self.category_open = category.is_some();
        if let Some(_category) = category {
            self.category_menu(ui);
        }
        if ui.small_button("Name##available-sort") {
            self.draft.reverse_source_order();
        }
        ui.child_window("##custom-available")
            .size(size)
            .border(true)
            .build(|| {
                self.source_focused = ui.is_window_focused();
                self.visible_source.clear();
                let mut transfer = None;
                for command in self.draft.available_commands() {
                    let clicked = ui
                        .selectable_config(command.label())
                        .selected(
                            self.draft.source == Some(command)
                                || self.draft.source_extra.contains(&command),
                        )
                        .build();
                    let secondary_clicked =
                        ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Right);
                    if clicked || secondary_clicked {
                        self.draft.reset_source_selection(command);
                        self.list_focus = Some(ListFocus::Source);
                    }
                    if ui.is_item_visible() {
                        self.visible_source.push(command);
                    }
                    if self.scroll_source_to_selection && self.draft.source == Some(command) {
                        ui.set_scroll_here_y();
                        self.scroll_source_to_selection = false;
                    }
                    if ui.is_item_hovered() && ui.is_mouse_double_clicked(imgui::MouseButton::Left)
                    {
                        transfer = Some(command);
                    }
                    locations::show(ui, command);
                }
                if let Some(command) = transfer {
                    self.draft.reset_source_selection(command);
                    self.draft.add();
                }
            });
    }
    fn transfer(&mut self, ui: &Ui) {
        ui.dummy([1., 55.]);
        let disabled = ui.begin_disabled(self.draft.source.is_none());
        self.focus_add_from_source(ui);
        if ui.button(">##custom-add") {
            self.draft.add();
        }
        drop(disabled);
        let disabled = ui.begin_disabled(self.draft.selected.is_none());
        if ui.button("<##custom-remove") {
            self.draft.remove();
        }
        drop(disabled);
    }
    fn target_list(&mut self, ui: &Ui) {
        let size = layout::list_size(ui, 245., 210.);
        ui.text("Custom panel");
        let suffix = if self.draft.target_changed() {
            " *"
        } else {
            ""
        };
        let target = target_selector::target_caption(self.draft.target, suffix);
        ui.set_next_item_width(size[0]);
        let targets = ui.begin_combo_with_flags(
            "##custom-target",
            &target,
            imgui::ComboBoxFlags::HEIGHT_LARGEST,
        );
        if targets.is_some() && !self.target_selector_open {
            self.target_cursor = target_selector::target_index(self.draft.target);
        }
        self.target_selector_open = targets.is_some();
        if let Some(_targets) = targets {
            self.target_menu(ui);
        }
        layout::target_headings(ui);
        ui.child_window("##custom-selected")
            .size(size)
            .border(true)
            .build(|| {
                self.target_focused = ui.is_window_focused();
                self.visible_target_rows = 0;
                let mut selected = self.draft.selected;
                let mut remove = false;
                let mut clear_extra = false;
                let mut options = Vec::new();
                for (index, entry) in self.draft.rows().iter().enumerate() {
                    let row_x = ui.cursor_pos()[0];
                    let clicked = ui
                        .selectable_config(format!("{}##target-{index}", entry.command.label()))
                        .size([126., 0.])
                        .selected(
                            selected == Some(index) || self.draft.target_extra.contains(&index),
                        )
                        .build();
                    if ui.is_item_visible() {
                        self.visible_target_rows += 1;
                    }
                    if self.scroll_target_to_selection && selected == Some(index) {
                        ui.set_keyboard_focus_here_with_offset(FocusedWidget::Previous);
                        ui.set_scroll_here_y();
                        self.scroll_target_to_selection = false;
                    }
                    let secondary_clicked =
                        ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Right);
                    if clicked || secondary_clicked {
                        selected = Some(index);
                        clear_extra = true;
                        self.list_focus = Some(ListFocus::Target);
                    }
                    if ui.is_item_hovered() && ui.is_mouse_double_clicked(imgui::MouseButton::Left)
                    {
                        selected = Some(index);
                        clear_extra = true;
                        remove = true;
                    }
                    layout::same_row_at(ui, row_x + layout::TARGET_OPTIONS_X[0]);
                    let mut large = entry.large;
                    let mut text = entry.text;
                    let changed_large = ui.checkbox(format!("##large-{index}"), &mut large);
                    layout::same_row_at(ui, row_x + layout::TARGET_OPTIONS_X[1]);
                    let changed_text = ui.checkbox(format!("##text-{index}"), &mut text);
                    if changed_large || changed_text {
                        options.push((index, large, text));
                    }
                }
                self.draft.selected = selected;
                if clear_extra && let Some(index) = selected {
                    self.draft.reset_target_selection(index);
                }
                if remove {
                    self.draft.remove();
                } else {
                    for (index, large, text) in options {
                        self.draft.set_options(index, large, text);
                    }
                }
            });
    }
    fn reorder(&mut self, ui: &Ui) {
        ui.dummy([1., 55.]);
        let disabled = ui.begin_disabled(self.draft.selected.is_none());
        if ui.button("Up") {
            self.draft.move_selected(-1);
        }
        if ui.button("Down") {
            self.draft.move_selected(1);
        }
        drop(disabled);
    }
    fn list_keyboard(&mut self, ui: &Ui, focused: bool) {
        let active_list = match self.list_focus {
            Some(ListFocus::Source) if self.source_focused => ListFocus::Source,
            Some(ListFocus::Target) if self.target_focused => ListFocus::Target,
            _ => return,
        };
        if !focused {
            return;
        }
        if ui.io().key_ctrl && ui.is_key_pressed_no_repeat(imgui::Key::A) {
            match active_list {
                ListFocus::Source => self.draft.select_visible_source(&self.visible_source),
                ListFocus::Target => self.draft.select_all_target(),
            }
            self.navigation_consumed = true;
            return;
        }
        if !ui.io().key_shift {
            let handled = match active_list {
                ListFocus::Source => self.navigate_source(ui),
                ListFocus::Target => self.navigate_target(ui),
            };
            if handled {
                self.navigation_consumed = true;
            }
            return;
        }
        let direction = if ui.is_key_pressed_no_repeat(imgui::Key::DownArrow) {
            1
        } else if ui.is_key_pressed_no_repeat(imgui::Key::UpArrow) {
            -1
        } else {
            return;
        };
        match active_list {
            ListFocus::Source => self.draft.extend_source(direction),
            ListFocus::Target => self.draft.extend_target(direction),
        }
        self.navigation_consumed = true;
    }
    fn has_pending_changes(&self) -> bool {
        self.draft.changed()
            || self.builtins.changed()
            || self.staged_appearance != self.appearance
            || self.keyboard_changed()
            || self.marking_changed()
    }
    fn buttons(&mut self, ui: &Ui, actions: &mut Vec<crate::GuiShellAction>) {
        ui.separator();
        let changed = self.has_pending_changes();
        let conflict = self.keyboard_conflict || self.marking_conflict;
        let disabled = ui.begin_disabled(!changed || conflict);
        if ui.button("OK") {
            match self.apply() {
                Ok(()) => {
                    self.settings_error = None;
                    let keyboard = self.submit_keyboard(actions);
                    let marking = self.submit_marking(actions);
                    if keyboard || marking {
                        self.close_after_apply = true;
                    } else {
                        self.open.set_open(false);
                    }
                }
                Err(error) => self.settings_error = Some(error),
            }
        }
        drop(disabled);
        ui.same_line();
        let cancel_disabled = ui.begin_disabled(!changed);
        if ui.button("Cancel") {
            self.open.set_open(false);
        }
        drop(cancel_disabled);
        ui.same_line();
        let disabled = ui.begin_disabled(!changed || conflict);
        if ui.button("Apply") {
            self.settings_error = self.apply().err();
            if self.settings_error.is_none() {
                self.submit_keyboard(actions);
                self.submit_marking(actions);
            }
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Close") {
            self.open.set_open(false);
        }
        ui.same_line();
        if ui.button("Import") {
            self.settings.request(settings_dialog::Mode::Import);
        }
        ui.same_line();
        if ui.button("Export") {
            self.settings.request(settings_dialog::Mode::Export);
        }
    }
}

#[cfg(test)]
#[path = "store_tests.rs"]
mod store_tests;
