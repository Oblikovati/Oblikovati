// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiParameterMutation, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_api::wire::ParameterSetArgs;

#[derive(Default)]
pub(crate) struct GuiParameterEditor {
    document: Option<u64>,
    visible: bool,
    name: String,
    expression: String,
}
impl GuiParameterEditor {
    pub(crate) fn toolbar(&mut self, ui: &Ui, state: &GuiCommandState) {
        self.synchronize_document(state.document);
        let disabled = ui.begin_disabled(state.document.is_none());
        if ui.button_with_size("Parameters", [108., 58.]) {
            self.visible = !self.visible;
        }
        drop(disabled);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.synchronize_document(state.document);
        if !self.visible {
            return;
        }
        ui.window("Parameters")
            .position([320., 8.], Condition::FirstUseEver)
            .size([540., 360.], Condition::FirstUseEver)
            .build(|| self.contents(ui, state, actions));
    }
    fn synchronize_document(&mut self, document: Option<u64>) {
        if self.document == document {
            return;
        }
        *self = Self {
            document,
            ..Default::default()
        };
    }
    fn contents(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        ui.input_text("Name", &mut self.name).build();
        ui.input_text("Equation", &mut self.expression).build();
        self.apply_buttons(ui, actions);
        ui.separator();
        self.parameter_rows(ui, state);
    }
    fn apply_buttons(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let disabled =
            ui.begin_disabled(self.name.trim().is_empty() || self.expression.trim().is_empty());
        for (label, mutation) in [
            ("Add", GuiParameterMutation::Add),
            ("Apply", GuiParameterMutation::Set),
        ] {
            if ui.button(label) {
                actions.push(self.request(mutation));
            }
            ui.same_line();
        }
        drop(disabled);
        if ui.button("Cancel") {
            self.visible = false;
        }
    }
    fn request(&self, mutation: GuiParameterMutation) -> GuiShellAction {
        GuiShellAction::Parameter {
            mutation,
            request: ParameterSetArgs {
                name: self.name.clone(),
                expression: self.expression.clone(),
                ..Default::default()
            },
        }
    }
    fn parameter_rows(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.columns(4, "parameter-columns", true);
        for title in ["Name", "Equation", "Value", "Health"] {
            ui.text(title);
            ui.next_column();
        }
        ui.separator();
        for row in &state.parameters {
            if ui.selectable(&row.name) {
                self.name = row.name.clone();
                self.expression = row.expression.clone();
            }
            ui.next_column();
            ui.text(&row.expression);
            ui.next_column();
            ui.text(&row.value);
            ui.next_column();
            ui.text(&row.health);
            ui.next_column();
        }
        ui.columns(1, "parameter-columns-end", false);
    }
}
