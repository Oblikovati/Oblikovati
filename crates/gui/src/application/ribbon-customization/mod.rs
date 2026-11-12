// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon command edits stay in a draft until Apply or OK.
mod panel;
mod state;
use crate::ribbon::RibbonTab;
use imgui::Ui;
use state::{COMMANDS, Command, Draft};

#[derive(Clone, Copy)]
enum ListFocus {
    Source,
    Target,
}
#[derive(Default)]
pub(crate) struct RibbonCustomization {
    open: bool,
    draft: Draft,
    escape: crate::application::document_saving::folder_browser::DialogEscape,
    pub(crate) escape_consumed: bool,
    pub(crate) navigation_consumed: bool,
    list_focus: Option<ListFocus>,
    source_focused: bool,
    target_focused: bool,
    visible_source: Vec<Command>,
}
impl RibbonCustomization {
    pub(crate) fn open(&mut self, tab: RibbonTab) {
        self.draft.open(tab);
        self.escape = Default::default();
        self.list_focus = None;
        self.open = true;
    }
    pub(crate) fn has_panel(&self, tab: RibbonTab) -> bool {
        self.draft
            .applied(tab)
            .iter()
            .any(|entry| entry.command != Command::Separator)
    }
    pub(crate) fn panel(&mut self, ui: &Ui) {
        self.escape_consumed = false;
        self.navigation_consumed = false;
        if !self.open {
            return;
        }
        crate::widget_abi::fit_dialog(ui, "Customize", [340., 185.], [760., 440.]);
        let mut window_open = self.open;
        ui.window("Customize")
            .opened(&mut window_open)
            .collapsible(false)
            .build(|| {
                self.tabs(ui);
                self.lists(ui);
                self.buttons(ui);
                let focused = ui.is_window_focused_with_flags(
                    imgui::WindowFocusedFlags::ROOT_AND_CHILD_WINDOWS,
                );
                self.list_keyboard(ui, focused);
                if self.escape.pressed(ui) && focused {
                    self.open = false;
                    self.escape_consumed = true;
                }
            });
        self.open &= window_open;
    }
    fn tabs(&mut self, ui: &Ui) {
        if let Some(_tabs) = ui.tab_bar("##customize-tabs") {
            if let Some(_ribbon) = ui.tab_item("Ribbon") {
                ui.text("Customize user commands");
            }
            let disabled = ui.begin_disabled(true);
            ui.tab_item("Keyboard");
            ui.tab_item("Marking Menu");
            drop(disabled);
        }
    }
    fn lists(&mut self, ui: &Ui) {
        ui.group(|| self.source_list(ui));
        ui.same_line();
        ui.group(|| self.transfer(ui));
        ui.same_line();
        ui.group(|| self.target_list(ui));
        ui.same_line();
        ui.group(|| self.reorder(ui));
    }
    fn source_list(&mut self, ui: &Ui) {
        ui.text("Available commands");
        let category = if self.draft.sketch_category {
            "Sketch Tab"
        } else {
            "All Commands"
        };
        ui.set_next_item_width(230.);
        if let Some(_category) = ui.begin_combo("##command-category", category) {
            for (label, sketch) in [("All Commands", false), ("Sketch Tab", true)] {
                if ui.selectable(label) {
                    self.draft.sketch_category = sketch;
                    self.draft.source = None;
                    self.draft.source_extra.clear();
                }
            }
        }
        ui.child_window("##custom-available")
            .size([230., 200.])
            .border(true)
            .build(|| {
                self.source_focused = ui.is_window_focused();
                self.visible_source.clear();
                let mut transfer = None;
                for command in COMMANDS
                    .into_iter()
                    .filter(|command| self.draft.available(*command))
                    .collect::<Vec<_>>()
                {
                    if ui
                        .selectable_config(command.label())
                        .selected(
                            self.draft.source == Some(command)
                                || self.draft.source_extra.contains(&command),
                        )
                        .build()
                    {
                        self.draft.reset_source_selection(command);
                        self.list_focus = Some(ListFocus::Source);
                    }
                    if ui.is_item_visible() {
                        self.visible_source.push(command);
                    }
                    if ui.is_item_hovered() && ui.is_mouse_double_clicked(imgui::MouseButton::Left)
                    {
                        transfer = Some(command);
                    }
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
        ui.text("Custom panel");
        let suffix = if self.draft.target_changed() {
            " *"
        } else {
            ""
        };
        let target = format!("Part | {}{suffix}", self.draft.target.label());
        ui.set_next_item_width(245.);
        if let Some(_targets) = ui.begin_combo("##custom-target", &target) {
            for tab in [
                RibbonTab::Model,
                RibbonTab::Sketch,
                RibbonTab::Annotate,
                RibbonTab::Tools,
                RibbonTab::Manage,
                RibbonTab::View,
            ] {
                if ui.selectable(format!("Part | {}", tab.label())) {
                    self.draft.switch(tab);
                }
            }
        }
        ui.text("Name");
        ui.same_line_with_spacing(0., 110.);
        ui.text("Large");
        ui.same_line_with_spacing(0., 12.);
        ui.text("Text");
        ui.child_window("##custom-selected")
            .size([245., 200.])
            .border(true)
            .build(|| {
                self.target_focused = ui.is_window_focused();
                let mut selected = self.draft.selected;
                let mut remove = false;
                let mut clear_extra = false;
                let mut options = Vec::new();
                for (index, entry) in self.draft.rows().iter().enumerate() {
                    if ui
                        .selectable_config(format!("{}##target-{index}", entry.command.label()))
                        .size([126., 0.])
                        .selected(
                            selected == Some(index) || self.draft.target_extra.contains(&index),
                        )
                        .build()
                    {
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
                    ui.same_line();
                    let mut large = entry.large;
                    let mut text = entry.text;
                    let changed_large = ui.checkbox(format!("##large-{index}"), &mut large);
                    ui.same_line_with_spacing(0., 28.);
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
    fn buttons(&mut self, ui: &Ui) {
        ui.separator();
        let changed = self.draft.changed();
        let disabled = ui.begin_disabled(!changed);
        if ui.button("OK") {
            self.draft.apply();
            self.open = false;
        }
        ui.same_line();
        if ui.button("Cancel") {
            self.open = false;
        }
        ui.same_line();
        if ui.button("Apply") {
            self.draft.apply();
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Close") {
            self.open = false;
        }
    }
}
