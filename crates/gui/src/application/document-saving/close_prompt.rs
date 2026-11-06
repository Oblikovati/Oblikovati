// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::{CloseDocumentArgs, DocumentInfo};
use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};

use super::folder_browser::DialogEscape;

const SAVE_CHANGES: &str = "Save changes";
const EXIT_ENVIRONMENT: &str = "Exit environment";
const FREQUENCY_LIST_WIDTH: f32 = 280.;

fn frequency_label(frequency: PromptFrequency) -> &'static str {
    match frequency {
        PromptFrequency::AlwaysShow => "Always show this message",
        PromptFrequency::OncePerOperation => "Show once per operation",
        PromptFrequency::SuppressForSession => "Suppress for this session",
        PromptFrequency::SuppressPermanently => "Suppress permanently",
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum CloseAnswer {
    Save,
    Discard,
    Keep,
}
/// What one frame's answers ask of the owner.
#[derive(Debug, Default, PartialEq)]
pub(crate) struct CloseOutcome {
    pub actions: Vec<GuiShellAction>,
    /// Documents answered Yes that have no file: each needs a Save As before it closes.
    pub first_saves: Vec<u64>,
}
/// Where one unforced close of a document goes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum CloseRoute {
    /// Nothing is at risk: the owner closes it as asked.
    Pass,
    /// Nothing is at risk, but a new part's unused identity makes the owner refuse an unforced close.
    Discard,
    AskSaveChanges,
    LeaveSketch,
}
fn close_route(state: &GuiCommandState, id: u64) -> CloseRoute {
    let edited = storage(state, id).is_some_and(|row| row.unsaved_edits);
    let at_risk = state.document == Some(id) || edited;
    if state.authoring == GuiAuthoringContext::Sketch && at_risk {
        return CloseRoute::LeaveSketch;
    }
    if edited {
        return CloseRoute::AskSaveChanges;
    }
    let refused = document(state, id).is_some_and(|row| row.dirty);
    if refused {
        CloseRoute::Discard
    } else {
        CloseRoute::Pass
    }
}
#[derive(Default)]
pub(crate) struct ClosePrompt {
    asking: Vec<u64>,
    leaving: Vec<u64>,
    /// Closes held until the sketch has finished; the next frame routes them again.
    continuing: Vec<u64>,
    opening_save: bool,
    opening_exit: bool,
    escape: DialogEscape,
}
impl ClosePrompt {
    /// A frame must follow: a popup is about to open, or a held close must be routed again.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening_save || self.opening_exit || !self.continuing.is_empty()
    }
    /// Route every unforced close in `actions`; the one place all close commands pass through.
    /// Example: `prompt.intercept(&state, &mut actions)` after the frame's controls have drawn.
    pub(crate) fn intercept(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> CloseOutcome {
        let mut outcome = CloseOutcome::default();
        let resumed = self.take_continuing(state);
        let mut kept = Vec::with_capacity(actions.len());
        for action in resumed.into_iter().chain(actions.drain(..)) {
            match unforced_close(&action) {
                Some(id) => self.route(state, id, action, &mut kept, &mut outcome),
                None => kept.push(action),
            }
        }
        *actions = kept;
        outcome
    }
    /// Held closes resume once the sketch has finished; a refused finish keeps the document open.
    fn take_continuing(&mut self, state: &GuiCommandState) -> Vec<GuiShellAction> {
        let held = std::mem::take(&mut self.continuing);
        if state.authoring == GuiAuthoringContext::Sketch {
            return Vec::new();
        }
        held.into_iter().map(|id| close(id, false)).collect()
    }
    fn route(
        &mut self,
        state: &GuiCommandState,
        id: u64,
        action: GuiShellAction,
        kept: &mut Vec<GuiShellAction>,
        outcome: &mut CloseOutcome,
    ) {
        match close_route(state, id) {
            CloseRoute::Pass => kept.push(action),
            CloseRoute::Discard => kept.push(close(id, true)),
            CloseRoute::AskSaveChanges if state.prompts.save_changes.suppressed() => {
                answer(state, id, CloseAnswer::Save, outcome);
            }
            CloseRoute::AskSaveChanges => self.ask(id),
            CloseRoute::LeaveSketch if state.prompts.exit_environment.suppressed() => {
                self.leave_sketch([id], outcome);
            }
            CloseRoute::LeaveSketch => self.ask_to_leave(id),
        }
    }
    fn ask(&mut self, id: u64) {
        if !self.asking.contains(&id) {
            self.opening_save |= self.asking.is_empty();
            self.asking.push(id);
        }
    }
    fn ask_to_leave(&mut self, id: u64) {
        if !self.leaving.contains(&id) {
            self.opening_exit |= self.leaving.is_empty();
            self.leaving.push(id);
        }
    }
    fn leave_sketch(&mut self, ids: impl IntoIterator<Item = u64>, outcome: &mut CloseOutcome) {
        let first = self.continuing.is_empty();
        self.continuing.extend(ids);
        if first {
            outcome.actions.push(GuiShellAction::FinishSketch);
        }
    }
    /// Draw the waiting prompt; example: `prompt.panel(ui, &state)` after `intercept`.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) -> CloseOutcome {
        let mut outcome = CloseOutcome::default();
        let open = |id: &u64| state.documents.iter().any(|row| row.id == *id);
        self.asking.retain(open);
        self.leaving.retain(open);
        if self.leaving.is_empty() {
            self.save_changes_prompt(ui, state, &mut outcome);
        } else {
            self.exit_environment_prompt(ui, state, &mut outcome);
        }
        outcome
    }
    fn exit_environment_prompt(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        outcome: &mut CloseOutcome,
    ) {
        if std::mem::take(&mut self.opening_exit) {
            ui.open_popup(EXIT_ENVIRONMENT);
        }
        let Some(_prompt) = ui.begin_modal_popup(EXIT_ENVIRONMENT) else {
            return;
        };
        ui.text("Exit the sketch environment and continue?");
        frequency_list(ui, state, ApplicationPrompt::ExitEnvironment, outcome);
        let accepted = crate::widget_abi::button(ui, "OK", [80., 0.]);
        ui.same_line();
        let cancelled =
            crate::widget_abi::button(ui, "Cancel", [80., 0.]) || self.escape.pressed(ui);
        if accepted {
            let held = std::mem::take(&mut self.leaving);
            self.leave_sketch(held, outcome);
        } else if cancelled {
            self.leaving.clear();
        }
        if accepted || cancelled {
            ui.close_current_popup();
        }
    }
    fn save_changes_prompt(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        outcome: &mut CloseOutcome,
    ) {
        let Some(&id) = self.asking.first() else {
            return;
        };
        if std::mem::take(&mut self.opening_save) {
            ui.open_popup(SAVE_CHANGES);
        }
        let Some(_prompt) = ui.begin_modal_popup(SAVE_CHANGES) else {
            return;
        };
        ui.text(format!(
            "Save changes to {} before closing?",
            document_name(state, id)
        ));
        frequency_list(ui, state, ApplicationPrompt::SaveChanges, outcome);
        if let Some(choice) = self.save_changes_buttons(ui) {
            self.resolve(state, choice, outcome);
            self.opening_save = !self.asking.is_empty();
            ui.close_current_popup();
        }
    }
    fn save_changes_buttons(&mut self, ui: &Ui) -> Option<CloseAnswer> {
        let mut chosen = None;
        for (label, choice) in [("Yes", CloseAnswer::Save), ("No", CloseAnswer::Discard)] {
            if crate::widget_abi::button(ui, label, [80., 0.]) {
                chosen = Some(choice);
            }
            ui.same_line();
        }
        let cancelled =
            crate::widget_abi::button(ui, "Cancel", [80., 0.]) || self.escape.pressed(ui);
        if cancelled {
            Some(CloseAnswer::Keep)
        } else {
            chosen
        }
    }
    /// Once per operation, one answer covers every waiting document; Cancel always ends the operation.
    fn resolve(
        &mut self,
        state: &GuiCommandState,
        choice: CloseAnswer,
        outcome: &mut CloseOutcome,
    ) {
        let shared = state.prompts.save_changes == PromptFrequency::OncePerOperation;
        let count = if shared || choice == CloseAnswer::Keep {
            self.asking.len()
        } else {
            1
        };
        for id in self.asking.drain(..count) {
            answer(state, id, choice, outcome);
        }
    }
}
fn frequency_list(
    ui: &Ui,
    state: &GuiCommandState,
    prompt: ApplicationPrompt,
    outcome: &mut CloseOutcome,
) {
    let current = state.prompts.frequency(prompt);
    // Wide enough for the longest choice: a clipped choice cannot be read or matched.
    ui.set_next_item_width(FREQUENCY_LIST_WIDTH);
    let Some(_list) = ui.begin_combo("##prompt-frequency", frequency_label(current)) else {
        return;
    };
    for frequency in PromptFrequency::ALL {
        if ui.selectable(frequency_label(frequency)) && frequency != current {
            outcome
                .actions
                .push(GuiShellAction::SetPromptFrequency { prompt, frequency });
        }
    }
}
fn unforced_close(action: &GuiShellAction) -> Option<u64> {
    match action {
        GuiShellAction::CloseDocument(CloseDocumentArgs { id, force: false }) => Some(*id),
        _ => None,
    }
}
fn answer(state: &GuiCommandState, id: u64, choice: CloseAnswer, outcome: &mut CloseOutcome) {
    match choice {
        CloseAnswer::Keep => {}
        CloseAnswer::Discard => outcome.actions.push(close(id, true)),
        CloseAnswer::Save if storage(state, id).is_some_and(|row| row.path.is_some()) => {
            outcome.actions.extend(activate_unless_active(state, id));
            outcome.actions.push(GuiShellAction::Save);
            // Unforced: if the save fails, the document is still modified and the owner keeps it.
            outcome.actions.push(close(id, false));
        }
        CloseAnswer::Save => outcome.first_saves.push(id),
    }
}
fn close(id: u64, force: bool) -> GuiShellAction {
    GuiShellAction::CloseDocument(CloseDocumentArgs { id, force })
}
fn activate_unless_active(state: &GuiCommandState, id: u64) -> Option<GuiShellAction> {
    (state.document != Some(id)).then_some(GuiShellAction::ActivateDocument(
        oblikovati_api::wire::ActivateDocumentArgs { id },
    ))
}
fn document(state: &GuiCommandState, id: u64) -> Option<&DocumentInfo> {
    state.documents.iter().find(|row| row.id == id)
}
fn storage(state: &GuiCommandState, id: u64) -> Option<&crate::GuiDocumentStorage> {
    state.document_storage.get(&id)
}
fn document_name(state: &GuiCommandState, id: u64) -> &str {
    document(state, id).map_or("this document", |row| row.name.as_str())
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/close_prompt.rs"]
mod tests;
