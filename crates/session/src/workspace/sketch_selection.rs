// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::selection::{SelectionSet, SketchPick, SketchPickTarget};
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityDeletion, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Mint a checked pick through the borrowed owner without changing workspace focus.
    /// Example: `commands.sketch_pick(sketch, SketchPickTarget::Dimension(id))?;`.
    pub fn sketch_pick(
        &self,
        sketch: DocumentSketchId,
        target: SketchPickTarget,
    ) -> Result<SketchPick, SketchError> {
        self.session.sketch_pick(sketch, target)
    }
    /// Apply editor deletion through the same borrowed owner and transaction stream.
    /// Example: `commands.delete_selected_sketch_picks(Some(sketch), false, &mut picks)?;`.
    pub fn delete_selected_sketch_picks(
        &mut self,
        active_sketch: Option<DocumentSketchId>,
        tool_active: bool,
        picks: &mut SelectionSet<SketchPick>,
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        self.session
            .delete_selected_sketch_picks(active_sketch, tool_active, picks)
    }
}
