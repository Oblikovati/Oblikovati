// SPDX-License-Identifier: GPL-2.0-only
//! Reversible fillet authoring through the session owner (#4045).
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::HistoryError,
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Create authored fillet inputs as one edit; example: `session.create_fillet("Fillet", definition)?`.
    pub fn create_fillet(
        &mut self,
        name: &str,
        definition: FilletDefinition,
    ) -> Result<FeatureId, CommandError<HistoryError>> {
        self.create_fillet_projected(name, definition, |result| result)
    }
    /// Prepare a reply before adopting a fillet; example: `session.create_fillet_projected("Fillet", definition, encode)?`.
    pub fn create_fillet_projected<R, E>(
        &mut self,
        name: &str,
        definition: FilletDefinition,
        project: impl FnOnce(Result<FeatureId, HistoryError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Create fillet", |snapshot, ids| {
            project(snapshot.create_fillet(name, definition, ids))
        })
    }
}
