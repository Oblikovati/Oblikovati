// SPDX-License-Identifier: GPL-2.0-only
use super::{
    GuiParameterAck, GuiParameterAction, Ui, parameter_controls_identity,
    request_state::ParameterEditRequest,
};
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiParameterInitialValue {
    Numeric(String),
    Text(String),
    Boolean(bool),
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterCreation {
    pub document: u64,
    pub request: u64,
    pub name: String,
    pub value: GuiParameterInitialValue,
}
#[derive(Default)]
pub(super) struct ParameterCreationForm {
    document: Option<u64>,
    kind: usize,
    name: String,
    expression: String,
    boolean: bool,
    request: ParameterEditRequest<GuiParameterCreation>,
}
impl ParameterCreationForm {
    pub(super) fn synchronize(&mut self, document: Option<u64>) {
        if self.document == document {
            return;
        }
        self.document = document;
        self.name.clear();
        self.expression.clear();
        self.request.retire();
    }
    pub(super) fn submit(&mut self) -> Result<Option<GuiParameterCreation>, String> {
        if self.request.pending.is_some() {
            return Ok(None);
        }
        let document = self.document.ok_or_else(|| {
            "GUI parameter creation without a document: expected an active document".to_owned()
        })?;
        require_creation_text(&self.name)?;
        require_creation_text(&self.expression)?;
        let request = self.admit_request()?;
        let creation = self.owned_creation(document, request);
        self.request.publish(request, creation.clone())?;
        Ok(Some(creation))
    }
    fn admit_request(&self) -> Result<u64, String> {
        self.request.prepare()
    }
    fn owned_creation(&self, document: u64, request: u64) -> GuiParameterCreation {
        GuiParameterCreation {
            document,
            request,
            name: self.name.clone(),
            value: self.initial_value(),
        }
    }
    pub(super) fn acknowledge(
        &mut self,
        creation: &GuiParameterCreation,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        let outcome = self.request.acknowledge(creation, result);
        if outcome == GuiParameterAck::Applied {
            self.name.clear();
            self.expression.clear();
        }
        outcome
    }
    fn initial_value(&self) -> GuiParameterInitialValue {
        match self.kind {
            1 => GuiParameterInitialValue::Text(self.expression.clone()),
            2 => GuiParameterInitialValue::Boolean(self.boolean),
            _ => GuiParameterInitialValue::Numeric(self.expression.clone()),
        }
    }
    pub(super) fn draw(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        let _identity = parameter_controls_identity(ui, self.document, None);
        ui.separator();
        let _disabled =
            ui.begin_disabled(self.request.pending.is_some() || self.document.is_none());
        ui.text("Add");
        ui.same_line();
        self.draw_kind(ui);
        ui.same_line();
        ui.set_next_item_width(140.);
        ui.input_text("##add-name", &mut self.name).build();
        ui.same_line();
        self.draw_value(ui);
        ui.same_line();
        if ui.button("Add") {
            self.enqueue(actions);
        }
        if let Some(issue) = &self.request.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
    }
    fn draw_kind(&mut self, ui: &Ui) {
        let names = ["Numeric", "Text", "True/False"];
        ui.set_next_item_width(120.);
        let Some(_combo) = ui.begin_combo("##add-kind", names[self.kind]) else {
            return;
        };
        for (kind, name) in names.into_iter().enumerate() {
            if ui
                .selectable_config(name)
                .selected(self.kind == kind)
                .build()
            {
                self.kind = kind;
            }
        }
    }
    fn draw_value(&mut self, ui: &Ui) {
        if self.kind == 2 {
            ui.checkbox("value##add-bool", &mut self.boolean);
            return;
        }
        ui.set_next_item_width(140.);
        ui.input_text("##add-value", &mut self.expression).build();
    }
    fn enqueue(&mut self, actions: &mut Vec<GuiParameterAction>) {
        match self.submit() {
            Ok(Some(creation)) => actions.push(GuiParameterAction::Add(creation)),
            Ok(None) => {}
            Err(issue) => self.request.issue = Some(issue),
        }
    }
}
fn require_creation_text(text: &str) -> Result<(), String> {
    if text.len() > 1024 * 1024 || text.contains('\0') {
        return Err(format!(
            "GUI parameter creation text length {}, NUL={}: expected NUL-free text within 1048576 bytes",
            text.len(),
            text.contains('\0')
        ));
    }
    Ok(())
}
#[cfg(test)]
#[path = "../../tests/parameters/create.rs"]
mod tests;
