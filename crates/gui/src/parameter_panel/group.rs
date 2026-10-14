// SPDX-License-Identifier: GPL-2.0-only
use super::request_state::ParameterEditRequest;
use super::*;
use oblikovati_api::types::ParameterGroupKey;
#[derive(Clone, Debug, PartialEq)]
pub struct GuiParameterGroupEdit {
    pub target: GuiParameterTextTarget,
    pub request: u64,
    pub key: ParameterGroupKey,
}
impl GuiParameterGroupEdit {
    /// Check one owned membership change; example: `edit.validate()?` before entering the document command.
    pub fn validate(&self) -> Result<(), String> {
        if self.target.document == 0
            || self.target.field != GuiParameterTextField::Name
            || self.request == 0
            || self.key.0.is_empty()
        {
            return Err(format!(
                "GUI parameter group edit {self:?}: expected an active name target, positive request, and nonempty group key"
            ));
        }
        crate::parameter_drafts::require_draft_text(&self.key.0, 1024 * 1024)
    }
}
#[derive(Default)]
pub(super) struct ParameterGroupEditor {
    target: Option<GuiParameterTextTarget>,
    name: String,
    request: ParameterEditRequest<GuiParameterGroupEdit>,
}
impl ParameterGroupEditor {
    pub(super) fn open(&mut self, document: u64, row: &GuiParameterRow) {
        self.target = Some(GuiParameterTextTarget {
            document,
            parameter: row.parameter,
            field: GuiParameterTextField::Name,
        });
        self.name.clear();
        self.request.retire();
    }
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        let valid = self.target.is_some_and(|target| {
            snapshot.is_some_and(|view| {
                view.document == target.document
                    && view
                        .rows
                        .iter()
                        .any(|row| row.parameter == target.parameter)
            })
        });
        if !valid {
            self.close();
        }
    }
    fn close(&mut self) {
        self.target = None;
        self.name.clear();
        self.request.retire();
    }
    fn submit(&mut self) -> Result<Option<GuiParameterGroupEdit>, String> {
        if self.request.pending.is_some() {
            return Ok(None);
        }
        let target = self.target.ok_or_else(|| {
            "GUI group editor without target: expected an active parameter".to_owned()
        })?;
        let request = self.request.prepare()?;
        let edit = GuiParameterGroupEdit {
            target,
            request,
            key: ParameterGroupKey(self.name.trim().to_owned().into()),
        };
        edit.validate()?;
        self.request.publish(request, edit.clone())?;
        Ok(Some(edit))
    }
    pub(super) fn acknowledge(
        &mut self,
        edit: &GuiParameterGroupEdit,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        let outcome = self.request.acknowledge(edit, result);
        if outcome == GuiParameterAck::Applied {
            self.close();
        }
        outcome
    }
    pub(super) fn draw(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        if self.target.is_none() {
            return;
        }
        let mut open = true;
        ui.window("Add to Group")
            .size([360., 180.], Condition::FirstUseEver)
            .opened(&mut open)
            .build(|| self.draw_body(ui, actions));
        if !open {
            self.close();
        }
    }
    fn draw_body(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        let Some(target) = self.target else {
            return;
        };
        let _identity =
            parameter_controls_identity(ui, Some(target.document), Some(target.parameter));
        let _disabled = ui.begin_disabled(self.request.pending.is_some());
        ui.text("Group name:");
        ui.input_text("##group-name", &mut self.name).build();
        if ui.button("OK") {
            self.enqueue(actions);
        }
        ui.same_line();
        if ui.button("Cancel") {
            self.close();
        }
        if let Some(issue) = &self.request.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
    }
    fn enqueue(&mut self, actions: &mut Vec<GuiParameterAction>) {
        match self.submit() {
            Ok(Some(edit)) => actions.push(GuiParameterAction::Group(edit)),
            Ok(None) => {}
            Err(issue) => self.request.issue = Some(issue),
        }
    }
}
#[cfg(test)]
#[path = "../../tests/parameters/group.rs"]
mod tests;
