// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Client, wire::SaveDocumentArgs};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};
/// Command feedback stays on the owner until the native window consumes it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DesktopCommandNotice {
    NoActiveDocument,
    SavePathRequired { document: u64 },
    Saved { document: u64, path: String },
    SaveFailed { document: u64, cause: String },
}
impl DesktopCommandNotice {
    /// Explain the result for native feedback; example: `notice.message()` in a title or status panel.
    pub fn message(&self) -> String {
        match self {
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
            if save_pressed(input) {
                self.command_notice = Some(self.save_active_document());
                changed = true;
            }
        }
        Ok(changed)
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

fn save_pressed(input: &ViewportEvent) -> bool {
    matches!(
        input,
        ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Save,
            pressed: true
        })
    )
}
