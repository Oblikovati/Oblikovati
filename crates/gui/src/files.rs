// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};
use oblikovati_api::wire::{OpenDocumentArgs, SaveDocumentAsArgs};

#[derive(Clone, Copy, Debug, PartialEq)]
enum DocumentPathPurpose {
    Open,
    SaveAs(u64),
}
struct DocumentPathDraft {
    purpose: DocumentPathPurpose,
    path: String,
}
#[derive(Default)]
pub(crate) struct GuiDocumentFiles {
    draft: Option<DocumentPathDraft>,
}
impl GuiDocumentFiles {
    pub(crate) fn request_save_path(&mut self, document: u64) -> Result<(), String> {
        if document == 0 {
            return Err(format!(
                "save dialog document {document}: expected a nonzero document identity"
            ));
        }
        self.begin(DocumentPathPurpose::SaveAs(document));
        Ok(())
    }
    fn begin(&mut self, purpose: DocumentPathPurpose) {
        self.draft = Some(DocumentPathDraft {
            purpose,
            path: String::new(),
        });
    }
    pub(crate) fn menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.menu("File", || {
            if ui.menu_item("New Part") {
                actions.push(GuiShellAction::NewPart);
            }
            self.open_menu_item(ui, state);
            file_save_item(ui, state, actions);
            self.save_as_menu_item(ui, state);
            file_close_item(ui, state, actions);
        });
    }
    fn open_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        if ui
            .menu_item_config("Open...")
            .enabled(!state.transaction_open)
            .build()
        {
            self.begin(DocumentPathPurpose::Open);
        }
    }
    fn save_as_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        let chosen = ui
            .menu_item_config("Save As...")
            .enabled(state.document.is_some() && !state.transaction_open)
            .build();
        if let Some(document) = state.document.filter(|_| chosen) {
            self.begin(DocumentPathPurpose::SaveAs(document));
        }
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        if let Some(draft) = &self.draft {
            draft.validate_path()?;
        }
        Ok(())
    }
    fn synchronize_document(&mut self, document: Option<u64>) {
        let stale = self.draft.as_ref().is_some_and(|draft| {
            matches!(draft.purpose, DocumentPathPurpose::SaveAs(owner) if document != Some(owner))
        });
        if stale {
            self.draft = None;
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.synchronize_document(state.document);
        let Some(draft) = &mut self.draft else {
            return;
        };
        let close = ui
            .window(draft.purpose.title())
            .position([320., 168.], Condition::FirstUseEver)
            .size([620., 160.], Condition::FirstUseEver)
            .collapsible(false)
            .build(|| path_controls(ui, state, draft, actions))
            .unwrap_or(false);
        if close {
            self.draft = None;
        }
    }
}
impl DocumentPathPurpose {
    fn title(self) -> &'static str {
        match self {
            Self::Open => "Open document",
            Self::SaveAs(_) => "Save document as",
        }
    }
    fn button(self) -> &'static str {
        match self {
            Self::Open => "Open",
            Self::SaveAs(_) => "Save",
        }
    }
}
impl DocumentPathDraft {
    fn validate_path(&self) -> Result<(), String> {
        if self.path.contains('\0') {
            return Err(format!(
                "document path {:?}: expected text without NUL",
                self.path
            ));
        }
        Ok(())
    }
    fn action(&self) -> Result<GuiShellAction, String> {
        self.validate_path()?;
        Ok(match self.purpose {
            DocumentPathPurpose::Open => GuiShellAction::OpenDocument(self.open_request()),
            DocumentPathPurpose::SaveAs(document) => {
                GuiShellAction::SaveDocumentAs(self.save_request(document))
            }
        })
    }
    fn open_request(&self) -> OpenDocumentArgs {
        OpenDocumentArgs {
            full_document_name: self.path.clone(),
            visible: true,
            defer_content: false,
        }
    }
    fn save_request(&self, document: u64) -> SaveDocumentAsArgs {
        SaveDocumentAsArgs {
            document,
            new_full_document_name: self.path.clone(),
        }
    }
}
fn file_save_item(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let can_save = state.document.is_some() && !state.transaction_open;
    if ui.menu_item_config("Save").enabled(can_save).build() {
        actions.push(GuiShellAction::Save);
    }
}
fn file_close_item(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let chosen = ui
        .menu_item_config("Close")
        .enabled(state.document.is_some() && !state.transaction_open)
        .build();
    if let Some(id) = state.document.filter(|_| chosen) {
        actions.push(GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs { id, force: false },
        ));
    }
}
fn path_controls(
    ui: &Ui,
    state: &GuiCommandState,
    draft: &mut DocumentPathDraft,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let entered = path_entry(ui, draft);
    let accepted = submit_path(ui, state, draft, entered, actions);
    ui.same_line();
    let cancelled = ui.button("Cancel") || ui.is_key_pressed(imgui::Key::Escape);
    accepted || cancelled
}
fn path_entry(ui: &Ui, draft: &mut DocumentPathDraft) -> bool {
    if ui.is_window_appearing() {
        ui.set_keyboard_focus_here();
    }
    ui.input_text("File name", &mut draft.path)
        .enter_returns_true(true)
        .build()
}
fn submit_path(
    ui: &Ui,
    state: &GuiCommandState,
    draft: &DocumentPathDraft,
    entered: bool,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let action = draft
        .action()
        .expect("validated draft and GUI text input exclude NUL");
    let can_submit = !draft.path.is_empty() && action.dispatchable(state);
    let disabled = ui.begin_disabled(!can_submit);
    let accepted = (ui.button(draft.purpose.button()) || entered) && can_submit;
    drop(disabled);
    if accepted {
        actions.push(action);
    }
    accepted
}

#[cfg(test)]
#[path = "../tests/files/unit.rs"]
mod tests;
