// SPDX-License-Identifier: GPL-2.0-only
use super::*;
/// The document history action is distinct from camera Previous View.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum DesktopHistoryAction {
    Undo,
    Redo,
}
impl DesktopHistoryAction {
    /// Name the action in command feedback; example: `DesktopHistoryAction::Undo.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Undo => "Undo",
            Self::Redo => "Redo",
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
    pub(super) fn restore_document_history(
        &mut self,
        action: DesktopHistoryAction,
    ) -> Option<DesktopCommandNotice> {
        let Some(document) = self.files.workspace().active_document() else {
            return Some(history_failure(
                action,
                "expected an active document".into(),
            ));
        };
        match self.files.workspace().session(document) {
            Ok(session) if session.in_transaction() => return None,
            Err(cause) => return Some(history_failure(action, cause.to_string())),
            _ => {}
        }
        Some(self.call_document_history(document, action))
    }
    fn call_document_history(
        &mut self,
        document: u64,
        action: DesktopHistoryAction,
    ) -> DesktopCommandNotice {
        let client = Client::new(self.document_host());
        let result = match action {
            DesktopHistoryAction::Undo => client.transactions().undo(),
            DesktopHistoryAction::Redo => client.transactions().redo(),
        };
        match result {
            Ok(_) => DesktopCommandNotice::HistoryCompleted { document, action },
            Err(cause) => history_failure(action, cause.to_string()),
        }
    }
}
fn history_failure(action: DesktopHistoryAction, cause: String) -> DesktopCommandNotice {
    DesktopCommandNotice::HistoryFailed { action, cause }
}
