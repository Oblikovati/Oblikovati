// SPDX-License-Identifier: GPL-2.0-only
//! Route fillet authoring to the selected session (#4045).
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::HistoryError,
};

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create a fillet without changing document focus; example: `commands.create_fillet("Fillet", definition)?`.
    pub fn create_fillet(
        &mut self,
        name: &str,
        definition: FilletDefinition,
    ) -> Result<FeatureId, CommandError<HistoryError>> {
        self.session.create_fillet(name, definition)
    }
    /// Prepare a reply before adopting a fillet; example: `commands.create_fillet_projected("Fillet", definition, encode)?`.
    pub fn create_fillet_projected<R, E>(
        &mut self,
        name: &str,
        definition: FilletDefinition,
        project: impl FnOnce(Result<FeatureId, HistoryError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_fillet_projected(name, definition, project)
    }
}
