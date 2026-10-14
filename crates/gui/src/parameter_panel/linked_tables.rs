// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::DerivedParameterTableInfo;
#[derive(Default)]
pub(super) struct LinkPicker {
    open: bool,
    source: Option<u64>,
    checked: std::collections::BTreeSet<u64>,
    request: request_state::ParameterEditRequest<GuiParameterLinkEdit>,
}
impl LinkPicker {
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        let valid = snapshot.is_some_and(|view| {
            self.source
                .is_none_or(|id| view.link_sources.iter().any(|source| source.document == id))
        });
        if !valid {
            self.close();
        }
    }
    pub(super) fn open(&mut self) {
        self.open = true;
        self.source = None;
        self.checked.clear();
        self.request.retire();
    }
    fn close(&mut self) {
        self.open = false;
        self.source = None;
        self.checked.clear();
        self.request.retire();
    }
    pub(super) fn acknowledge(&mut self, edit: &GuiParameterLinkEdit, result: Result<(), String>) {
        if self.request.acknowledge(edit, result) == GuiParameterAck::Applied {
            self.close();
        }
    }
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        snapshot: Option<&GuiParameterSnapshot>,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        if !self.open {
            return;
        }
        let Some(snapshot) = snapshot else {
            self.close();
            return;
        };
        let mut open = true;
        ui.window("Link Parameters")
            .size([360., 320.], Condition::FirstUseEver)
            .opened(&mut open)
            .build(|| self.draw_body(ui, snapshot, actions));
        if !open {
            self.close();
        }
    }
    fn draw_body(
        &mut self,
        ui: &Ui,
        snapshot: &GuiParameterSnapshot,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        let _identity = parameter_controls_identity(ui, Some(snapshot.document), self.source);
        if snapshot.link_sources.is_empty() {
            ui.text("No other open document offers parameters to link.");
        } else {
            self.draw_sources(ui, snapshot);
            self.draw_parameters(ui, snapshot);
        }
        ui.separator();
        let _disabled = ui.begin_disabled(self.request.pending.is_some() || self.source.is_none());
        if ui.button("Link") {
            self.enqueue(snapshot, actions);
        }
        ui.same_line();
        if ui.button("Cancel") {
            self.close();
        }
        if let Some(issue) = &self.request.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
    }
    fn draw_sources(&mut self, ui: &Ui, snapshot: &GuiParameterSnapshot) {
        ui.text("Derive parameters from:");
        let preview = self
            .source
            .and_then(|id| {
                snapshot
                    .link_sources
                    .iter()
                    .find(|source| source.document == id)
            })
            .map_or("(choose a document)", |source| source.display_name.as_str());
        let Some(_combo) = ui.begin_combo("##link-source", preview) else {
            return;
        };
        for source in &snapshot.link_sources {
            if ui
                .selectable_config(&source.display_name)
                .selected(self.source == Some(source.document))
                .build()
                && self.source != Some(source.document)
            {
                self.source = Some(source.document);
                self.checked.clear();
            }
        }
    }
    fn draw_parameters(&mut self, ui: &Ui, snapshot: &GuiParameterSnapshot) {
        let Some(source) = self.source.and_then(|id| {
            snapshot
                .link_sources
                .iter()
                .find(|source| source.document == id)
        }) else {
            return;
        };
        ui.separator();
        ui.text("Parameters");
        for parameter in &source.parameters {
            let mut checked = self.checked.contains(&parameter.parameter);
            if ui.checkbox(&parameter.name, &mut checked) {
                if checked {
                    self.checked.insert(parameter.parameter);
                } else {
                    self.checked.remove(&parameter.parameter);
                }
            }
        }
    }
    fn enqueue(&mut self, snapshot: &GuiParameterSnapshot, actions: &mut Vec<GuiParameterAction>) {
        let Some(source_document) = self.source else {
            return;
        };
        let request = match self.request.prepare() {
            Ok(request) => request,
            Err(issue) => return self.request.issue = Some(issue),
        };
        let edit = GuiParameterLinkEdit {
            document: snapshot.document,
            source_document,
            request,
            linked: self.checked.iter().copied().collect(),
        };
        match self.request.publish(request, edit.clone()) {
            Ok(()) => actions.push(GuiParameterAction::Link(edit)),
            Err(issue) => self.request.issue = Some(issue),
        }
    }
}
pub(super) fn draw_linked_tables(
    ui: &Ui,
    snapshot: &GuiParameterSnapshot,
    actions: &mut Vec<GuiParameterAction>,
    picker: &mut LinkPicker,
) {
    let _identity = parameter_controls_identity(ui, Some(snapshot.document), None);
    ui.separator();
    ui.text("Linked Parameters");
    if snapshot.linked_tables.is_empty() {
        ui.text("(none linked)");
    } else {
        let height =
            ui.frame_height_with_spacing() * (snapshot.linked_tables.len().min(8) + 1) as f32;
        if let Some(_table) = ui.begin_table_with_sizing(
            "##linked-tables",
            4,
            TableFlags::BORDERS | TableFlags::ROW_BG | TableFlags::SCROLL_Y,
            [0., height],
            0.,
        ) {
            for label in ["Source", "Linked", "Health", ""] {
                ui.table_setup_column(label);
            }
            ui.table_setup_scroll_freeze(0, 1);
            ui.table_headers_row();
            for table in &snapshot.linked_tables {
                draw_linked_table_row(ui, snapshot.document, table, actions);
            }
        }
    }
    if ui.button("Link...") {
        picker.open();
    }
}
fn draw_linked_table_row(
    ui: &Ui,
    document: u64,
    table: &DerivedParameterTableInfo,
    actions: &mut Vec<GuiParameterAction>,
) {
    let _identity = ui.push_id(format!("linked-table:{}", table.id));
    ui.table_next_row();
    ui.table_next_column();
    ui.text(&table.source_document);
    ui.table_next_column();
    ui.text(table.linked.as_deref().unwrap_or_default().join(", "));
    ui.table_next_column();
    ui.text(linked_health_label(&table.health));
    ui.table_next_column();
    if ui.button("Delete") {
        actions.push(GuiParameterAction::DeleteLinkedTable {
            document,
            table: table.id,
        });
    }
}
fn linked_health_label(health: &str) -> &str {
    if health.is_empty() {
        return "OK";
    }
    health
}
#[cfg(test)]
#[path = "../../tests/parameters/linked_tables.rs"]
mod tests;
