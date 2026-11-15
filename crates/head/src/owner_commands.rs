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
    ThemeSaved {
        name: String,
    },
    ThemeFilesDeclined {
        causes: Vec<String>,
    },
    DocumentOpened {
        document: u64,
        path: String,
    },
    DocumentClosed {
        document: u64,
    },
    WindowCaptureFailed {
        path: std::path::PathBuf,
        cause: String,
    },
    AuthoringFailed {
        cause: String,
    },
    SketchEditFailed {
        cause: String,
    },
    PreviewFailed {
        cause: String,
    },
    /// A command ended with nothing left to report: the status line returns to its idle text.
    Ready,
    PlacementPrompt {
        message: String,
    },
    SelectionChanged {
        count: usize,
    },
    SelectionFailed {
        cause: String,
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
    SavedWithWarning {
        document: u64,
        path: String,
        detail: String,
    },
    ClosedWithWarning {
        document: u64,
        detail: String,
    },
    SaveFailed {
        document: u64,
        cause: String,
    },
    CloseFailed {
        document: u64,
        cause: String,
    },
}
impl DesktopCommandNotice {
    /// Explain the result for native feedback; example: `notice.message()` in a title or status panel.
    pub fn message(&self) -> String {
        match self {
            Self::ThemeSaved { name } => format!("Saved theme {name}"),
            Self::ThemeFilesDeclined { causes } => {
                format!("Theme files declined: {}", causes.join("; "))
            }
            Self::DocumentOpened { document, path } => {
                format!("Opened document {document}: {path}")
            }
            Self::DocumentClosed { document } => format!("Closed document {document}"),
            Self::WindowCaptureFailed { path, cause } => {
                format!("Window capture {path:?} failed: {cause}")
            }
            Self::AuthoringFailed { cause } => format!("Command failed: {cause}"),
            Self::SketchEditFailed { cause } => format!("Sketch edit could not be opened: {cause}"),
            Self::PreviewFailed { cause } => format!("Preview could not be drawn: {cause}"),
            Self::Ready => "Ready".into(),
            Self::PlacementPrompt { message } => message.clone(),
            Self::SelectionChanged { count } => format!("Sketch selection count: {count}."),
            Self::SelectionFailed { cause } => format!("Selection failed: {cause}"),
            Self::PlacementFailed { cause } => format!("Sketch placement failed: {cause}"),
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
            Self::SavedWithWarning {
                document,
                path,
                detail,
            } => format!("Saved document {document}: {path}. Save warning: {detail}"),
            Self::ClosedWithWarning { document, detail } => {
                format!("Closed document {document}. Save warning: {detail}")
            }
            Self::SaveFailed { document, cause } => format!("Save document {document}: {cause}"),
            Self::CloseFailed { document, cause } => format!("Close document {document}: {cause}"),
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
        if let Ok(saved) = &result {
            self.note_recent_file(&saved.full_document_name);
        }
        self.command_notice = Some(match result {
            Ok(saved) => saved_notice(document, saved),
            Err(cause) => DesktopCommandNotice::SaveFailed { document, cause },
        });
        true
    }
    fn save_selected_document(
        &mut self,
        document: u64,
        path: &std::path::Path,
    ) -> Result<oblikovati_api::wire::SaveDocumentResult, String> {
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
            Ok(saved) => saved_notice(document, saved),
            Err(cause) => DesktopCommandNotice::SaveFailed {
                document,
                cause: cause.to_string(),
            },
        }
    }
}

pub(super) fn warning_details(
    warnings: Option<Vec<oblikovati_api::wire::SaveWarning>>,
) -> Option<String> {
    let details: Vec<_> = warnings
        .unwrap_or_default()
        .into_iter()
        .map(|warning| warning.detail)
        .collect();
    (!details.is_empty()).then(|| details.join("; "))
}
fn saved_notice(
    document: u64,
    saved: oblikovati_api::wire::SaveDocumentResult,
) -> DesktopCommandNotice {
    match warning_details(saved.warnings) {
        Some(detail) => DesktopCommandNotice::SavedWithWarning {
            document,
            path: saved.full_document_name,
            detail,
        },
        None => DesktopCommandNotice::Saved {
            document,
            path: saved.full_document_name,
        },
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

impl<
    O: super::TransactionObserver,
    W: super::WorkspaceObserver,
    F: super::DocumentFiles,
    G: super::GuidSource,
    R: super::DocumentObserverFactory<Observer = O> + super::HeadInvalidation,
    S: super::DispatchWakeup,
> super::DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn jump_history_action(
        &mut self,
        document: u64,
        position: i64,
    ) -> Result<bool, String> {
        let client = oblikovati_api::client::Client::new(self.document_host());
        let current = client
            .transactions()
            .history(&oblikovati_api::wire::TransactionHistoryArgs { document })
            .map_err(|cause| cause.to_string())?;
        client
            .transactions()
            .jump_to(&oblikovati_api::wire::TransactionJumpToArgs { document, position })
            .map_err(|cause| cause.to_string())?;
        Ok(current.position != position)
    }
    pub(super) fn update_part_action(&mut self) -> Result<bool, String> {
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .update(&oblikovati_api::wire::UpdateDocumentArgs::default())
            .map_err(|error| error.to_string())?;
        Ok(true)
    }
}
