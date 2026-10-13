// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{SaveDocumentArgs, SaveDocumentAsArgs},
};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};
#[path = "owner_history.rs"]
mod history;
pub use history::DesktopHistoryAction;
#[derive(Clone, Copy)]
enum NativeDocumentCommand {
    Save,
    History(DesktopHistoryAction),
}
/// Command feedback stays on the owner until the native window consumes it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DesktopCommandNotice {
    PlacementPrompt {
        message: String,
    },
    PlacementFailed {
        cause: String,
    },
    AddInStartupFailed {
        library: std::path::PathBuf,
        cause: String,
    },
    NoActiveDocument,
    HistoryCompleted {
        document: u64,
        action: DesktopHistoryAction,
    },
    HistoryFailed {
        action: DesktopHistoryAction,
        cause: String,
    },
    SavePathRequired {
        document: u64,
    },
    Saved {
        document: u64,
        path: String,
    },
    SaveFailed {
        document: u64,
        cause: String,
    },
}
impl DesktopCommandNotice {
    /// Explain the result for native feedback; example: `notice.message()` in a title or status panel.
    pub fn message(&self) -> String {
        match self {
            Self::PlacementPrompt { message } => message.clone(),
            Self::PlacementFailed { cause } => format!("Slot could not be created: {cause}"),
            Self::AddInStartupFailed { library, cause } => format!("Add-in {library:?}: {cause}"),
            Self::HistoryCompleted { document, action } => {
                format!("{} document {document}: complete", action.label())
            }
            Self::HistoryFailed { action, cause } => format!("{}: {cause}", action.label()),
            Self::NoActiveDocument => "Save: no active document".into(),
            Self::SavePathRequired { document } => {
                format!("Save document {document}: a save path is required")
            }
            Self::Saved { document, path } => format!("Saved document {document}: {path}"),
            Self::SaveFailed { document, cause } => format!("Save document {document}: {cause}"),
        }
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn process_document_inputs(
        &mut self,
        inputs: &[ViewportEvent],
    ) -> Result<bool, String> {
        let mut changed = false;
        for input in inputs {
            let Some(command) = document_command(input) else {
                continue;
            };
            let notice = match command {
                NativeDocumentCommand::Save => Some(self.save_active_document()),
                NativeDocumentCommand::History(action) => self.restore_document_history(action),
            };
            if notice.is_some() {
                self.command_notice = notice;
                changed = true;
            }
        }
        Ok(changed)
    }
    pub(super) fn complete_document_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> bool {
        let Some(path) = selected else {
            return false;
        };
        let result = self.save_selected_document(document, path);
        self.command_notice = Some(match result {
            Ok(path) => DesktopCommandNotice::Saved { document, path },
            Err(cause) => DesktopCommandNotice::SaveFailed { document, cause },
        });
        true
    }
    fn save_selected_document(
        &mut self,
        document: u64,
        path: &std::path::Path,
    ) -> Result<String, String> {
        self.files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let name = path
            .to_str()
            .ok_or_else(|| format!("save path {path:?}: expected a UTF-8 document name"))?;
        Client::new(self.document_host())
            .documents()
            .save_as(&SaveDocumentAsArgs {
                document,
                new_full_document_name: name.into(),
            })
            .map(|saved| saved.full_document_name)
            .map_err(|cause| cause.to_string())
    }
    fn save_active_document(&mut self) -> DesktopCommandNotice {
        let Some(document) = self.files.workspace().active_document() else {
            return DesktopCommandNotice::NoActiveDocument;
        };
        if self.files.path(document).is_err() {
            return DesktopCommandNotice::SavePathRequired { document };
        }
        self.save_bound_document(document)
    }
    fn save_bound_document(&mut self, document: u64) -> DesktopCommandNotice {
        match Client::new(self.document_host())
            .documents()
            .save(&SaveDocumentArgs { document })
        {
            Ok(saved) => DesktopCommandNotice::Saved {
                document,
                path: saved.full_document_name,
            },
            Err(cause) => DesktopCommandNotice::SaveFailed {
                document,
                cause: cause.to_string(),
            },
        }
    }
}

fn document_command(input: &ViewportEvent) -> Option<NativeDocumentCommand> {
    let ViewportEvent::Input(ViewportInput::Key { key, pressed: true }) = input else {
        return None;
    };
    match key {
        ViewportKey::Save => Some(NativeDocumentCommand::Save),
        ViewportKey::Undo => Some(NativeDocumentCommand::History(DesktopHistoryAction::Undo)),
        ViewportKey::Redo => Some(NativeDocumentCommand::History(DesktopHistoryAction::Redo)),
        _ => None,
    }
}
