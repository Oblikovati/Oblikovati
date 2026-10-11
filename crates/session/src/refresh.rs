// SPDX-License-Identifier: GPL-2.0-only

use crate::{CommandError, DocumentSession, MutationError, TransactionObserver};
use oblikovati_model::{parameter::ParameterError, quantity::Quantity};
use std::collections::BTreeMap;

impl<O: TransactionObserver> DocumentSession<O> {
    /// Refresh a derived table as a consequence of its source, preserving target undo/redo.
    /// Example: `session.refresh_derived_table(table, Some(&values))?` after source evaluation.
    pub fn refresh_derived_table(
        &mut self,
        table: i64,
        source: Option<&BTreeMap<String, Quantity>>,
    ) -> Result<bool, CommandError<ParameterError<u64>>> {
        let options = self.sketch_options;
        let changed = self
            .history
            .refresh(|snapshot| refresh_source_snapshot(snapshot, table, source, options))
            .map_err(CommandError::Edit)?;
        if changed {
            self.observer.changed(self.document, self.generation());
        }
        Ok(changed)
    }
}

fn refresh_source_snapshot(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    table: i64,
    source: Option<&BTreeMap<String, Quantity>>,
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> Result<(), MutationError<ParameterError<u64>>> {
    snapshot
        .parameters
        .sync_derived_table(table, source)
        .map_err(MutationError::Command)?;
    snapshot
        .properties
        .sync_exposed_parameters(&snapshot.parameters)
        .map_err(MutationError::Property)?;
    crate::sketch_recompute::update_pending_sketches(snapshot, options);
    Ok(())
}
