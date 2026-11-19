// SPDX-License-Identifier: GPL-2.0-only
//! Stage all model-state table changes in one typed host request (#4119).
use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    model_state_table_cells::{parse_cell, property_text},
};
use imgui::{TableFlags, Ui};
use oblikovati_api::{
    types::{MODEL_STATE_TABLE_PROPERTY_COLUMN, ModelStatePropertyKey, VariantValue},
    wire::{
        ModelStateTableColumnRef, ModelStateTableProperty, ModelStateTablePropertyCell,
        ModelStateTableRowRef, ModelStateTableView, SetModelStateTableArgs,
    },
};
use std::collections::{BTreeMap, BTreeSet};
type CellKey = (String, String, String);

#[derive(Default)]
pub(crate) struct ModelStateTable {
    document: Option<u64>,
    dock: crate::dockable_lifecycle::DockableState,
    draft: Option<ModelStateTableView>,
    extra: crate::model_state_table_extra::ExtraSelection,
    available: Option<ModelStatePropertyKey>,
    chosen: Option<ModelStatePropertyKey>,
    buffers: BTreeMap<CellKey, String>,
    invalid: BTreeSet<CellKey>,
    dirty: bool,
    pending: bool,
    pending_request: Option<u64>,
    request_sequence: u64,
    submission_error: Option<String>,
    row_sequence: u64,
    show_verification_error: bool,
    name_prompt: Option<names::NamePrompt>,
    accept_after_verification: Option<SetModelStateTableArgs>,
}
impl DockableWindow for ModelStateTable {}
impl ModelStateTable {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Model State Table", &self.dock)
    }
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        let (Some(document), Some(view)) = (state.document, state.model_state_table.as_ref())
        else {
            return;
        };
        self.document = Some(document);
        self.draft = Some((**view).clone());
        self.extra = Default::default();
        self.available = None;
        self.chosen = None;
        self.buffers.clear();
        self.invalid.clear();
        self.dirty = false;
        self.pending = false;
        self.pending_request = None;
        self.submission_error = None;
        self.row_sequence = 0;
        self.show_verification_error = false;
        self.name_prompt = None;
        self.accept_after_verification = None;
        self.dock.set_open(true);
    }
    fn close(&mut self) {
        self.document = None;
        self.draft = None;
        self.dock.set_open(false);
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        if self.document.is_some() && self.document != state.document {
            self.close();
        }
        if !self.pending {
            return;
        }
        let Some((document, request_id, result)) = state.model_state_table_ack.as_ref() else {
            return;
        };
        if Some(*document) != self.document || Some(*request_id) != self.pending_request {
            return;
        }
        if let Err(error) = result {
            self.pending = false;
            self.pending_request = None;
            self.submission_error = Some(error.clone());
            return;
        }
        let Some((draft, current)) = self.draft.as_ref().zip(state.model_state_table.as_ref())
        else {
            return;
        };
        if current.generation < draft.generation {
            return;
        }
        if crate::model_state_table_acceptance::same_table(draft, current) {
            self.close();
        } else if current.generation > draft.generation {
            self.pending = false;
            self.pending_request = None;
        }
    }
    fn stale(&self, state: &GuiCommandState) -> bool {
        self.draft
            .as_ref()
            .zip(state.model_state_table.as_ref())
            .is_some_and(|(draft, current)| draft.generation != current.generation)
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state);
        if self.document.is_none() {
            return;
        }
        let mut open = true;
        Self::dockable_window(ui, "Model State Table", &mut open)
            .size([760., 480.], imgui::Condition::FirstUseEver)
            .build(|| self.contents(ui, state, actions));
        if !open {
            self.close();
        }
    }
    fn contents(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if let Some(_tabs) = ui.tab_bar("model-state-categories") {
            if let Some(_tab) = ui.tab_item("Properties") {
                self.properties(ui);
            }
            if let Some(_tab) = ui.tab_item("Suppression") {
                self.extra_selection(ui, crate::model_state_table_extra::ExtraKind::Suppression);
            }
            if let Some(_tab) = ui.tab_item("Parameters") {
                self.extra_selection(ui, crate::model_state_table_extra::ExtraKind::Parameters);
            }
        }
        ui.separator();
        let verification = self.current_verification(state);
        self.extra.diagnostics = verification
            .as_ref()
            .and_then(|result| result.diagnostics.clone())
            .unwrap_or_default();
        self.grid(ui);
        let verification = self.current_verification(state);
        if let Some(checked) = self.accept_after_verification.as_ref() {
            let still_current = self
                .draft
                .as_ref()
                .is_some_and(|draft| request(draft) == *checked);
            if !still_current {
                self.accept_after_verification = None;
            } else if let Some(result) = verification.as_ref() {
                self.accept_after_verification = None;
                if result.valid {
                    self.accept(actions);
                } else {
                    self.show_verification_error = true;
                }
            }
        }
        if std::mem::take(&mut self.extra.verify_requested) {
            self.verify(actions);
        }
        ui.separator();
        if verification.as_ref().is_some_and(|result| result.valid) {
            ui.text("No errors found.");
        }
        if !self.invalid.is_empty() {
            ui.text("Enter a valid value for each property.");
        }
        if self.stale(state) {
            ui.text("The document changed. Cancel and reopen the table.");
        }
        if let Some(error) = &self.submission_error {
            ui.text_wrapped(error);
        }
        let disabled = ui.begin_disabled(
            !self.dirty || self.pending || !self.invalid.is_empty() || self.stale(state),
        );
        if ui.button("OK") {
            if let Some(prompt) = self.name_problem() {
                self.name_prompt = Some(prompt);
            } else {
                match verification {
                    Some(result) if result.valid => self.accept(actions),
                    Some(_) => self.show_verification_error = true,
                    None => {
                        self.accept_after_verification = self.draft.as_ref().map(request);
                        self.verify(actions);
                    }
                }
            }
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Verify") {
            self.verify(actions);
        }
        ui.same_line();
        if ui.button("Cancel") {
            self.close();
        }
        self.verification_modal(ui);
        self.name_modal(ui, actions);
    }
    fn extra_selection(&mut self, ui: &Ui, kind: crate::model_state_table_extra::ExtraKind) {
        let Some(view) = self.draft.as_mut() else {
            return;
        };
        self.dirty |= crate::model_state_table_extra::selection(ui, view, &mut self.extra, kind);
    }
    fn properties(&mut self, ui: &Ui) {
        let Some(_panes) = ui.begin_table_with_sizing(
            "property-selection",
            3,
            TableFlags::BORDERS | TableFlags::RESIZABLE,
            [0., ui.frame_height_with_spacing() * 8.],
            0.,
        ) else {
            return;
        };
        ui.table_next_row();
        ui.table_next_column();
        self.available_properties(ui);
        ui.table_next_column();
        self.transfer_buttons(ui);
        ui.table_next_column();
        self.chosen_properties(ui);
    }
    fn available_properties(&mut self, ui: &Ui) {
        let Some(view) = self.draft.as_ref() else {
            return;
        };
        let available: Vec<_> = view
            .available_properties
            .as_deref()
            .unwrap_or_default()
            .iter()
            .map(|item| item.property.clone())
            .collect();
        let mut sets = Vec::<String>::new();
        for key in &available {
            if !sets.contains(&key.set) {
                sets.push(key.set.clone());
            }
        }
        for set in sets {
            self.available_group(ui, &available, &set);
        }
    }
    fn available_group(&mut self, ui: &Ui, available: &[ModelStatePropertyKey], set: &str) {
        let label = if set == "Summary Information" {
            "Summary"
        } else {
            set
        };
        let Some(_tree) = ui.tree_node_config(label).default_open(true).push() else {
            return;
        };
        for key in available.iter().filter(|key| key.set == set) {
            let _id = ui.push_id(format!("available-{}-{}", key.set, key.name));
            if ui
                .selectable_config(&key.name)
                .selected(self.available.as_ref() == Some(key))
                .build()
            {
                self.available = Some(key.clone());
            }
        }
    }
    fn transfer_buttons(&mut self, ui: &Ui) {
        let disabled = ui.begin_disabled(self.available.is_none());
        if ui.button("Add >") {
            self.add();
        }
        drop(disabled);
        let disabled = ui.begin_disabled(self.chosen.is_none());
        if ui.button("< Remove") {
            self.remove();
        }
        drop(disabled);
    }
    fn chosen_properties(&mut self, ui: &Ui) {
        ui.text("Chosen properties");
        let Some(view) = self.draft.as_ref() else {
            return;
        };
        for column in view.columns.as_deref().unwrap_or_default() {
            let Some(key) = column.property.as_deref() else {
                continue;
            };
            if column.kind != MODEL_STATE_TABLE_PROPERTY_COLUMN {
                continue;
            }
            let _id = ui.push_id(format!("chosen-{}-{}", key.set, key.name));
            if ui
                .selectable_config(&key.name)
                .selected(self.chosen.as_ref() == Some(key))
                .build()
            {
                self.chosen = Some(key.clone());
            }
        }
    }
    fn add(&mut self) {
        let (Some(key), Some(view)) = (self.available.clone(), self.draft.as_mut()) else {
            return;
        };
        if property_visible(view, &key) {
            return;
        }
        if !view
            .properties
            .as_deref()
            .unwrap_or_default()
            .iter()
            .any(|item| item.property == key)
        {
            let column = new_column(view, key.clone());
            view.properties.get_or_insert_with(Vec::new).push(column);
        }
        view.columns
            .get_or_insert_with(Vec::new)
            .push(ModelStateTableColumnRef {
                kind: MODEL_STATE_TABLE_PROPERTY_COLUMN,
                property: Some(Box::new(key)),
                ..Default::default()
            });
        self.dirty = true;
    }
    fn remove(&mut self) {
        let (Some(key), Some(view)) = (self.chosen.take(), self.draft.as_mut()) else {
            return;
        };
        let columns = view.columns.get_or_insert_with(Vec::new);
        let before = columns.len();
        columns.retain(|column| {
            column.kind != MODEL_STATE_TABLE_PROPERTY_COLUMN
                || column.property.as_deref() != Some(&key)
        });
        if before == columns.len() {
            return;
        }
        self.invalid
            .retain(|(_, set, name)| *set != key.set || *name != key.name);
        self.buffers
            .retain(|(_, set, name), _| *set != key.set || *name != key.name);
        self.dirty = true;
    }
    fn grid(&mut self, ui: &Ui) {
        let Some(view) = self.draft.as_mut() else {
            return;
        };
        let rows = view.rows.clone().unwrap_or_default();
        let columns = view.columns.clone().unwrap_or_default();
        let Some(_table) = ui.begin_table_with_sizing(
            "model-state-grid",
            columns.len() + 2,
            TableFlags::BORDERS | TableFlags::ROW_BG | TableFlags::RESIZABLE,
            [0., 180.],
            0.,
        ) else {
            return;
        };
        ui.table_setup_column("#");
        ui.table_setup_column("Member");
        for column in &columns {
            let name = column.property.as_deref().map_or_else(
                || crate::model_state_table_extra::name(view, column),
                |key| key.name.as_str(),
            );
            ui.table_setup_column(name);
        }
        ui.table_headers_row();
        let mut row_action = None;
        for (index, row) in rows.iter().enumerate() {
            ui.table_next_row();
            ui.table_next_column();
            let _id = ui.push_id(row_key(&row.ref_));
            ui.small_button(format!("{}##row", index + 1));
            if let Some(_menu) = ui.begin_popup_context_item() {
                if ui.menu_item("Insert Row") {
                    row_action = Some((true, index));
                }
                let deletable = crate::model_state_table_rows::can_delete(view, index);
                if ui.menu_item_config("Delete Row").enabled(deletable).build() {
                    row_action = Some((false, index));
                }
                ui.separator();
                for label in [
                    "Set As Default / Compare Base Row",
                    "Custom Parameter Column",
                    "Column Range / Increment",
                    "Custom Parameter Cell",
                    "Cell Range / Increment",
                    "Delete Column",
                ] {
                    ui.menu_item_config(label).enabled(false).build();
                }
            }
            ui.table_next_column();
            let mut name = row.name.clone();
            let invalid_name = names::invalid(&rows, index);
            let highlight = invalid_name
                .then(|| ui.push_style_color(imgui::StyleColor::FrameBg, [0.75, 0.58, 0.12, 1.0]));
            let disabled = ui.begin_disabled(index == 0);
            ui.set_next_item_width(140.);
            if ui.input_text("##member-name", &mut name).build()
                && let Some(current) = view.rows.as_mut().and_then(|items| items.get_mut(index))
            {
                current.name = name;
                self.dirty = true;
            }
            drop(disabled);
            drop(highlight);
            for column in &columns {
                ui.table_next_column();
                if let Some(key) = column.property.as_deref() {
                    if let Some(property) = view
                        .properties
                        .as_mut()
                        .and_then(|items| items.iter_mut().find(|item| item.property == *key))
                    {
                        self.dirty |= edit_cell(
                            ui,
                            &row.ref_,
                            index,
                            property,
                            &mut self.buffers,
                            &mut self.invalid,
                        );
                    }
                } else {
                    self.dirty |= crate::model_state_table_extra::cell(
                        ui,
                        view,
                        &mut self.extra,
                        index,
                        column,
                    );
                }
            }
        }
        if let Some((insert, index)) = row_action {
            self.dirty |= if insert {
                crate::model_state_table_rows::insert(view, index, &mut self.row_sequence)
            } else {
                crate::model_state_table_rows::delete(view, index)
            };
        }
    }
    fn accept(&mut self, actions: &mut Vec<GuiShellAction>) {
        let (Some(document), Some(view)) = (self.document, self.draft.as_ref()) else {
            return;
        };
        let Some(request_id) = self.request_sequence.checked_add(1) else {
            self.submission_error = Some("Table request limit reached.".into());
            return;
        };
        self.request_sequence = request_id;
        self.pending_request = Some(request_id);
        self.submission_error = None;
        actions.push(GuiShellAction::SetModelStateTable {
            document,
            request_id,
            request: request(view),
        });
        self.pending = true;
    }
}
#[path = "model_state_table_helpers.rs"]
mod helpers;
#[path = "model_state_table_names.rs"]
mod names;
#[path = "model_state_table_verification.rs"]
mod verification;
use helpers::{edit_cell, new_column, property_visible, request, row_key};

#[cfg(test)]
#[path = "model_state_table_tests.rs"]
mod tests;
