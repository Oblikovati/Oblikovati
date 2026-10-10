// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentEditHistory, HistoryError};
use oblikovati_api::wire::TransactionHistoryEntry;

impl DocumentEditHistory {
    /// Read all committed steps oldest first, including the undone suffix.
    /// Example: `history.entries()[0].saved` reports whether that step was saved.
    pub fn entries(&self) -> Vec<TransactionHistoryEntry> {
        self.done
            .iter()
            .chain(self.undone.iter().rev())
            .map(|step| TransactionHistoryEntry {
                label: step.label.clone(),
                saved: step.saved,
            })
            .collect()
    }

    /// Mark the current committed step after storage has accepted the snapshot.
    /// Example: `history.mark_saved()?` after a successful file replacement.
    pub fn mark_saved(&mut self) -> Result<(), HistoryError> {
        self.require_closed()?;
        if let Some(step) = self.done.last_mut() {
            step.saved = true;
        }
        Ok(())
    }
}
