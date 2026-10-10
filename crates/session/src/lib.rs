// SPDX-License-Identifier: GPL-2.0-only

//! Serial document edits and transaction observations, independent of UI and I/O.
mod command;
mod error;
mod events;
pub mod files;
mod navigation;
mod refresh;
#[cfg(test)]
mod tests;
mod transaction;
pub mod workspace;
pub use error::{CommandError, MutationError, SessionError};
pub use events::TransactionObserver;
use oblikovati_model::history::{DocumentEditHistory, DocumentSnapshot, TransactionToken};

struct BoundedGroup {
    token: TransactionToken,
    label: String,
    depth: u64,
}

/// One document owner; observers can inspect snapshots but cannot mutate this owner.
pub struct DocumentSession<O> {
    document: u64,
    history: DocumentEditHistory,
    observer: O,
    group: Option<BoundedGroup>,
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Attach a document's history and observer; example: `DocumentSession::new(id, history, observer)`.
    pub fn new(document: u64, history: DocumentEditHistory, observer: O) -> Self {
        Self {
            document,
            history,
            observer,
            group: None,
        }
    }

    /// Inspect the current immutable recipe; example: `session.snapshot().parameters.ids()`.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        self.history.snapshot()
    }

    /// Read the worker generation; example: `let generation = session.generation();`.
    pub fn generation(&self) -> u64 {
        self.history.generation()
    }

    /// Inspect reserved identities after rollback; example: `session.high_water()`.
    pub fn high_water(&self) -> u64 {
        self.history.high_water()
    }

    /// Inspect the injected observer; example: `session.observer().recorded_events()`.
    pub fn observer(&self) -> &O {
        &self.observer
    }

    /// Check whether a bounded command group is open; example: `session.in_transaction()`.
    pub fn in_transaction(&self) -> bool {
        self.group.is_some()
    }

    /// Read committed labels in order; example: `session.undo_labels()`.
    pub fn undo_labels(&self) -> Vec<&str> {
        self.history.undo_labels()
    }

    /// Read available labels in redo order; example: `session.redo_labels()`.
    pub fn redo_labels(&self) -> Vec<&str> {
        self.history.redo_labels()
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Read the whole committed timeline, including redo and save markers.
    /// Example: `session.history_entries()` for a history browser.
    pub fn history_entries(&self) -> Vec<oblikovati_api::wire::TransactionHistoryEntry> {
        self.history.entries()
    }
}
