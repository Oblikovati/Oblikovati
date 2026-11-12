// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon command edits stay in a draft until Apply or OK.
mod panel;
mod state;
use crate::ribbon::RibbonTab;
use imgui::Ui;
use state::{COMMANDS, Command, Draft};

#[derive(Default)]
pub(crate) struct RibbonCustomization {
    open: bool,
    draft: Draft,
}
impl RibbonCustomization {
    pub(crate) fn open(&mut self, tab: RibbonTab) {
        self.draft.open(tab);
        self.open = true;
    }
    pub(crate) fn has_panel(&self, tab: RibbonTab) -> bool {
        self.draft
            .applied(tab)
            .iter()
            .any(|entry| entry.command != Command::Separator)
    }
    pub(crate) fn panel(&mut self, ui: &Ui) {
        if !self.open {
            return;
        }
        crate::widget_abi::fit_dialog(ui, "Customize", [340., 185.], [760., 440.]);
        ui.window("Customize").collapsible(false).build(|| {
            self.tabs(ui);
            self.lists(ui);
            self.buttons(ui);
        });
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
                }
            }
        }
        ui.child_window("##custom-available")
            .size([230., 200.])
            .border(true)
            .build(|| {
                for command in COMMANDS
                    .into_iter()
                    .filter(|command| self.draft.available(*command))
                    .collect::<Vec<_>>()
                {
                    if ui
                        .selectable_config(command.label())
                        .selected(self.draft.source == Some(command))
                        .build()
                    {
                        self.draft.source = Some(command);
                    }
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
        ui.child_window("##custom-selected")
            .size([245., 200.])
            .border(true)
            .build(|| {
                let mut selected = self.draft.selected;
                for (index, entry) in self.draft.rows().iter().enumerate() {
                    if ui
                        .selectable_config(format!("{}##target-{index}", entry.command.label()))
                        .selected(selected == Some(index))
                        .build()
                    {
                        selected = Some(index);
                    }
                }
                self.draft.selected = selected;
            });
        if let Some(entry) = self.draft.selected_mut() {
            ui.checkbox("Large", &mut entry.large);
            ui.same_line();
            ui.checkbox("Text", &mut entry.text);
        }
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
