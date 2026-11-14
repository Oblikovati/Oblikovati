// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::TransactionObserver;
use crate::selection::{SelectionClickMode, SelectionSet, SketchPick, SketchPickTarget};
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

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Change the owner selection shared by native tools and API readers.
    /// Example: `commands.set_sketch_selection(sketch, Some(target), mode, false)?;`.
    pub fn set_sketch_selection(
        &mut self,
        sketch: DocumentSketchId,
        target: Option<SketchPickTarget>,
        mode: SelectionClickMode,
        tool_active: bool,
    ) -> Result<Option<usize>, SketchError> {
        self.session
            .set_sketch_selection(sketch, target, mode, tool_active)
    }
    /// Clear document selection through the same owner used by API readers.
    /// Example: `commands.clear_transient_selection()` after a tool commits.
    pub fn clear_transient_selection(&mut self) -> bool {
        self.session.clear_transient_selection()
    }
}
