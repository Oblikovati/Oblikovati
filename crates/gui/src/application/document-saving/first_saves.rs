// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use oblikovati_api::wire::ActivateDocumentArgs;
use std::collections::VecDeque;

/// Documents that a close answered Yes for and that have no file yet.
/// Save As works on the active document, so each one activates before its dialog opens.
#[derive(Debug, Default)]
pub(crate) struct FirstSaves {
    waiting: VecDeque<u64>,
    activation_sent: Option<u64>,
}
impl FirstSaves {
    pub(crate) fn extend(&mut self, documents: impl IntoIterator<Item = u64>) {
        self.waiting.extend(documents);
    }
    /// Cancelling one Save As leaves the rest of the operation's documents open.
    pub(crate) fn cancel(&mut self) {
        self.waiting.clear();
        self.activation_sent = None;
    }
    pub(crate) fn is_waiting(&self) -> bool {
        !self.waiting.is_empty()
    }
    /// The next document whose Save As can open now; example: `saves.next(&state, &mut actions)`.
    pub(crate) fn next(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<u64> {
        let front = *self.waiting.front()?;
        if state.document == Some(front) {
            self.activation_sent = None;
            return self.waiting.pop_front();
        }
        if self.activation_sent == Some(front) {
            // The owner refused the switch and reported why; never spin on a refused document.
            self.cancel();
            return None;
        }
        self.activation_sent = Some(front);
        actions.push(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: front,
        }));
        None
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/first_saves.rs"]
mod tests;
