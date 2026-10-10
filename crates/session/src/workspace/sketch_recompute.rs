// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::document_sketch::DocumentSketchRecompute;
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Recompute this workspace document without replacing its owner; example: `commands.recompute_sketches(options, || false)?`.
    pub fn recompute_sketches(
        &mut self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<Infallible>> {
        self.session.recompute_sketches(options, cancelled)
    }
}
