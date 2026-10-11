// SPDX-License-Identifier: GPL-2.0-only

//! Serial document edits and transaction observations, independent of UI and I/O.
pub mod clients;
mod command;
mod error;
mod events;
pub mod files;
mod navigation;
mod refresh;
mod sketch_inference;
pub use sketch_inference::SketchInferenceProjectionInput;
mod sketch_construction;
mod sketch_create;
mod sketch_dimension_delete;
mod sketch_dimension_placement;
mod sketch_line;
mod sketch_relation_create;
mod sketch_relation_delete;
pub use sketch_relation_create::SketchGeometricCreationProjectionInput;
pub use sketch_relation_delete::SketchRelationDeleteProjectionInput;
mod sketch_primitive;
pub use sketch_line::SketchLineProjectionInput;
mod sketch_translation;
pub use sketch_construction::SketchConstructionProjectionInput;
pub use sketch_create::SketchCreationProjectionInput;
pub use sketch_primitive::SketchPrimitiveProjectionInput;
pub use sketch_translation::SketchTranslationProjectionInput;
mod sketch_delete;
mod sketch_recompute;
pub use sketch_delete::SketchDeleteProjectionInput;
pub use sketch_recompute::SketchSolveProjectionInput;
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
    sketch_options: oblikovati_solve::nonlinear::SolveOptions,
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Attach a document's history and observer; example: `DocumentSession::new(id, history, observer)`.
    pub fn new(document: u64, history: DocumentEditHistory, observer: O) -> Self {
        Self::new_with_sketch_options(document, history, observer, Default::default())
    }

    /// Inject bounded numerical policy for automatic readers; example: `DocumentSession::new_with_sketch_options(id, history, observer, options)`.
    pub fn new_with_sketch_options(
        document: u64,
        history: DocumentEditHistory,
        observer: O,
        sketch_options: oblikovati_solve::nonlinear::SolveOptions,
    ) -> Self {
        Self {
            document,
            history,
            observer,
            group: None,
            sketch_options,
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

pub use sketch_dimension_delete::SketchDimensionDeleteProjectionInput;
