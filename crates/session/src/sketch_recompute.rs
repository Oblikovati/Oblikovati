// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::document_sketch::DocumentSketchRecompute;
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Recompute within normal history, joining an open edit group; example: `session.recompute_sketches(options, || false)?`.
    pub fn recompute_sketches(
        &mut self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<Infallible>> {
        self.run("Recompute sketches", |snapshot, _| {
            let outcome = snapshot
                .sketches
                .solve(&snapshot.parameters, options, cancelled);
            snapshot.sketches = outcome.candidate;
            Ok(outcome.report)
        })
    }
}
