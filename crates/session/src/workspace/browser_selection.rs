// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{TransactionObserver, selection::SelectionClickMode};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Change the same transient selection read by the API and desktop.
    /// Example: `commands.set_browser_sketch_selection(Some(sketch), mode, false)?;`.
    pub fn set_browser_sketch_selection(
        &mut self,
        sketch: Option<DocumentSketchId>,
        mode: SelectionClickMode,
        tool_active: bool,
    ) -> Result<Option<usize>, SketchError> {
        self.session
            .set_browser_sketch_selection(sketch, mode, tool_active)
    }
}
