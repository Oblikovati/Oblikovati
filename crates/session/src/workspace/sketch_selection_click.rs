// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::TransactionObserver;
use crate::selection::{SelectionClickMode, SelectionSet, SketchPick};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Apply a resolved click without changing model history or workspace focus.
    /// Example: `commands.apply_sketch_click(sketch, Some(pick), mode, false, &mut picks)?;`.
    pub fn apply_sketch_click(
        &self,
        sketch: DocumentSketchId,
        picked: Option<SketchPick>,
        mode: SelectionClickMode,
        tool_active: bool,
        picks: &mut SelectionSet<SketchPick>,
    ) -> Result<Option<usize>, SketchError> {
        self.session
            .apply_sketch_click(sketch, picked, mode, tool_active, picks)
    }
}
