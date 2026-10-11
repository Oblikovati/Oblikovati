// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::document_sketch::{DocumentSketchId, DocumentSketchRecompute};
use oblikovati_model::sketch::SketchError;
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Solve only this document's selected sketch; example: `commands.solve_sketch(id, || false)?`.
    pub fn solve_sketch(
        &mut self,
        sketch: DocumentSketchId,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<SketchError>> {
        self.session.solve_sketch(sketch, cancelled)
    }
    /// Recompute this workspace document without replacing its owner; example: `commands.recompute_sketches(options, || false)?`.
    pub fn recompute_sketches(
        &mut self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<Infallible>> {
        self.session.recompute_sketches(options, cancelled)
    }
}
