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
    application_exit: bool,
    exit_advance: bool,
    exit_waiting: Option<u64>,
    exit_progress: bool,
    shared_exit_answer: Option<(CloseAnswer, Vec<u64>)>,
}
impl ClosePrompt {
    /// Begin one guarded application exit; example: `prompt.request_application_exit()`.
    pub(crate) fn request_application_exit(&mut self) {
        if !self.application_exit {
            self.application_exit = true;
            self.exit_advance = true;
        }
    }
    /// Stop an exit when its save is cancelled; example: `prompt.cancel_application_exit()`.
    pub(crate) fn cancel_application_exit(&mut self) {
        self.application_exit = false;
        self.exit_advance = false;
        self.exit_waiting = None;
        self.exit_progress = false;
        self.shared_exit_answer = None;
    }
    /// Continue after a first-save dialog closes; example: `prompt.resume_application_exit()`.
    pub(crate) fn resume_application_exit(&mut self) {
        self.exit_advance = self.application_exit;
    }
    /// Advance one exit without closing an unconfirmed document.
    /// Example: `prompt.advance_application_exit(&state, &mut actions, false)`.
    pub(crate) fn advance_application_exit(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        saving: bool,
    ) -> Vec<u64> {
        if !self.application_exit || !std::mem::take(&mut self.exit_advance) {
            return Vec::new();
        }
        if self.waiting_for_exit_document(state, saving) {
            return Vec::new();
        }
        self.exit_progress |= self.exit_waiting.take().is_some();
        self.route_application_exit(state, actions)
    }
    fn waiting_for_exit_document(&mut self, state: &GuiCommandState, saving: bool) -> bool {
        if !self
            .exit_waiting
            .is_some_and(|id| document(state, id).is_some())
        {
            return false;
        }
        if !saving {
            self.cancel_application_exit();
        }
        true
    }
    fn route_application_exit(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Vec<u64> {
        if let Some(id) = edited_exit_document(state) {
            return self.start_edited_exit(state, id, actions);
        }
        if let Some(id) = untouched_new_document(state) {
            actions.push(close(id, true));
            self.exit_waiting = Some(id);
            self.exit_advance = true;
            return Vec::new();
        }
        let complete = self.exit_progress;
        self.cancel_application_exit();
        if complete {
            actions.push(GuiShellAction::Exit);
        }
        Vec::new()
    }
    fn start_edited_exit(
        &mut self,
        state: &GuiCommandState,
        id: u64,
        actions: &mut Vec<GuiShellAction>,
    ) -> Vec<u64> {
        if let Some(choice) = self.shared_exit_choice(id) {
            return self.answer_exit_document(state, id, choice, actions);
        }
        if state.prompts.save_changes.suppressed() {
            return self.answer_exit_document(state, id, CloseAnswer::Save, actions);
        }
        self.ask(id);
        Vec::new()
    }
    fn shared_exit_choice(&mut self, id: u64) -> Option<CloseAnswer> {
        let (choice, documents) = self.shared_exit_answer.as_mut()?;
        let index = documents.iter().position(|document| *document == id)?;
        documents.remove(index);
        Some(*choice)
    }
    fn answer_exit_document(
        &mut self,
        state: &GuiCommandState,
        id: u64,
        choice: CloseAnswer,
        actions: &mut Vec<GuiShellAction>,
    ) -> Vec<u64> {
        let mut outcome = CloseOutcome::default();
        answer(state, id, choice, &mut outcome);
        self.exit_waiting = Some(id);
        self.exit_advance = true;
        actions.extend(outcome.actions);
        outcome.first_saves
    }
    /// A frame must follow: a popup is about to open, or a held close must be routed again.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening_save || self.opening_exit || self.exit_advance || !self.continuing.is_empty()
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
                // An exit-approved save must reach the owner, which refuses a dirty close (#4056).
                Some(id) if self.exit_waiting == Some(id) => kept.push(action),
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
        self.show_shared_exit_documents(ui, state, id);
        frequency_list(ui, state, ApplicationPrompt::SaveChanges, outcome);
        if let Some(choice) = self.save_changes_buttons(ui) {
            self.resolve(state, choice, outcome);
            self.opening_save = !self.asking.is_empty();
            ui.close_current_popup();
        }
    }
    fn show_shared_exit_documents(&self, ui: &Ui, state: &GuiCommandState, id: u64) {
        if !self.application_exit || state.prompts.save_changes != PromptFrequency::OncePerOperation
        {
            return;
        }
        let others = other_edited_documents(state, Some(id));
        if others.is_empty() {
            return;
        }
        ui.text("This answer also applies to:");
        ui.child_window("##shared-exit-documents")
            .size([360., 80.])
            .border(true)
            .build(|| {
                for document in others {
                    ui.text(document_name(state, document));
                }
            });
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
        if self.application_exit {
            if choice == CloseAnswer::Keep {
                self.cancel_application_exit();
            } else {
                self.remember_shared_exit_answer(state, choice);
                self.exit_waiting = self.asking.first().copied();
                self.exit_advance = true;
            }
        }
        let shared = state.prompts.save_changes == PromptFrequency::OncePerOperation;
        let count = if (shared && !self.application_exit) || choice == CloseAnswer::Keep {
            self.asking.len()
        } else {
            1
        };
        for id in self.asking.drain(..count) {
            answer(state, id, choice, outcome);
        }
    }
    fn remember_shared_exit_answer(&mut self, state: &GuiCommandState, choice: CloseAnswer) {
        if state.prompts.save_changes != PromptFrequency::OncePerOperation {
            return;
        }
        let current = self.asking.first().copied();
        let documents = other_edited_documents(state, current);
        self.shared_exit_answer = Some((choice, documents));
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
fn other_edited_documents(state: &GuiCommandState, current: Option<u64>) -> Vec<u64> {
    state
        .documents
        .iter()
        .filter_map(|row| {
            (Some(row.id) != current
                && storage(state, row.id).is_some_and(|item| item.unsaved_edits))
            .then_some(row.id)
        })
        .collect()
}
fn edited_exit_document(state: &GuiCommandState) -> Option<u64> {
    state
        .document
        .into_iter()
        .chain(state.documents.iter().map(|row| row.id))
        .find(|id| storage(state, *id).is_some_and(|row| row.unsaved_edits))
}
fn untouched_new_document(state: &GuiCommandState) -> Option<u64> {
    state.documents.iter().find_map(|row| {
        let storage = storage(state, row.id)?;
        (row.dirty
            && storage.path.is_none()
            && !storage.unsaved_edits
            && !(state.transaction_open && state.document == Some(row.id)))
        .then_some(row.id)
    })
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
