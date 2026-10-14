// SPDX-License-Identifier: GPL-2.0-only
use super::request_state::ParameterEditRequest;
use super::*;
use oblikovati_api::wire::ExpressionListInfo;
#[derive(Clone, Debug, PartialEq)]
pub struct GuiParameterChoiceEdit {
    pub target: GuiParameterTextTarget,
    pub request: u64,
    pub choices: ExpressionListInfo,
}
impl GuiParameterChoiceEdit {
    /// Check an owned list before document mutation; example: `edit.validate()?` on the owner.
    pub fn validate(&self) -> Result<(), String> {
        if self.target.document == 0
            || self.target.field != GuiParameterTextField::Equation
            || self.request == 0
        {
            return Err(format!(
                "GUI value-list edit {:?}, request {}: expected an active equation target and positive request",
                self.target, self.request
            ));
        }
        validate_choice_texts(self.choices.expressions.as_deref().unwrap_or_default())
    }
}
fn validate_choice_texts(choices: &[String]) -> Result<(), String> {
    let mut bytes = 0_usize;
    for choice in choices {
        crate::parameter_drafts::require_draft_text(choice, 1024 * 1024)?;
        bytes = bytes.checked_add(choice.len()).ok_or_else(|| {
            format!("GUI value-list byte count {bytes}: expected a representable text budget")
        })?;
        if bytes > 1024 * 1024 {
            return Err(format!(
                "GUI value-list byte count {bytes}: expected at most 1048576 bytes"
            ));
        }
    }
    Ok(())
}
#[derive(Default)]
pub(super) struct ParameterValueListEditor {
    target: Option<GuiParameterTextTarget>,
    text: String,
    allow_custom: bool,
    request: ParameterEditRequest<GuiParameterChoiceEdit>,
}
impl ParameterValueListEditor {
    pub(super) fn open(&mut self, document: u64, row: &GuiParameterRow) -> Result<(), String> {
        let choices = row.detail.expression_list.as_deref();
        let expressions = choices
            .and_then(|list| list.expressions.as_deref())
            .unwrap_or_default();
        let text = expressions.join("\n");
        crate::parameter_drafts::require_draft_text(&text, 1024 * 1024)?;
        self.target = Some(GuiParameterTextTarget {
            document,
            parameter: row.parameter,
            field: GuiParameterTextField::Equation,
        });
        self.text = text;
        self.allow_custom = choices.is_some_and(|list| list.allow_custom_values);
        self.request.retire();
        Ok(())
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
        self.request.retire();
        self.text.clear();
    }
    fn submit(&mut self) -> Result<Option<GuiParameterChoiceEdit>, String> {
        if self.request.pending.is_some() {
            return Ok(None);
        }
        let target = self.target.ok_or_else(|| {
            "GUI value-list editor without target: expected an active parameter identity".to_owned()
        })?;
        crate::parameter_drafts::require_draft_text(&self.text, 1024 * 1024)?;
        let request = self.request.prepare()?;
        let edit = GuiParameterChoiceEdit {
            target,
            request,
            choices: edited_choices(&self.text, self.allow_custom),
        };
        edit.validate()?;
        self.request.publish(request, edit.clone())?;
        Ok(Some(edit))
    }
    pub(super) fn acknowledge(
        &mut self,
        edit: &GuiParameterChoiceEdit,
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
        ui.window("Value List Editor")
            .size([360., 250.], Condition::FirstUseEver)
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
        ui.text("One value per line (or comma-separated):");
        ui.input_text_multiline("##value-list", &mut self.text, [-1., 100.])
            .build();
        ui.checkbox("Allow custom values", &mut self.allow_custom);
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
            Ok(Some(edit)) => actions.push(GuiParameterAction::Choices(edit)),
            Ok(None) => {}
            Err(issue) => self.request.issue = Some(issue),
        }
    }
}
fn edited_choices(text: &str, allow_custom: bool) -> ExpressionListInfo {
    let expressions: Vec<_> = text
        .split(['\n', ','])
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(str::to_owned)
        .collect();
    ExpressionListInfo {
        custom_order: !expressions.is_empty(),
        allow_custom_values: allow_custom && !expressions.is_empty(),
        expressions: (!expressions.is_empty()).then_some(expressions),
    }
}
#[cfg(test)]
#[path = "../../tests/parameters/value_list.rs"]
mod tests;
