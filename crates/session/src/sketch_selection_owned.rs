// SPDX-License-Identifier: GPL-2.0-only
use crate::selection::{SelectionClickMode, SketchPick, SketchPickTarget};
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Read current-generation editor picks; stale handles are never published.
    /// Example: `session.selected_sketch_picks(sketch)` for rendering or constraint tools.
    pub fn selected_sketch_picks(&self, sketch: DocumentSketchId) -> Vec<SketchPick> {
        self.selected_editor_picks()
            .into_iter()
            .filter(|pick| pick.sketch() == sketch)
            .collect()
    }
    /// Read all live editor picks from this document for shared API and scene queries.
    /// Example: `session.selected_editor_picks()` without relying on a GUI cache.
    pub fn selected_editor_picks(&self) -> Vec<SketchPick> {
        self.editor_selection
            .items()
            .iter()
            .copied()
            .filter(|pick| {
                pick.validate_owner(self.document, self.generation(), pick.sketch())
                    .is_ok()
                    && self.sketch_pick(pick.sketch(), pick.target()).is_ok()
            })
            .collect()
    }

    /// Store a checked editor click in the document owner without creating history.
    /// Example: `session.set_sketch_selection(sketch, Some(target), mode, false)?;`.
    pub fn set_sketch_selection(
        &mut self,
        sketch: DocumentSketchId,
        target: Option<SketchPickTarget>,
        mode: SelectionClickMode,
        tool_active: bool,
    ) -> Result<Option<usize>, SketchError> {
        if tool_active {
            return Ok(None);
        }
        let picked = self.checked_editor_click(sketch, target, mode)?;
        let retire_browser = (picked.is_some() || mode == SelectionClickMode::Replace)
            && !self.browser_selection.items().is_empty();
        let count = self.editor_selection.apply_click(picked, mode, false);
        if retire_browser {
            self.browser_selection.clear();
            return Ok(Some(self.editor_selection.items().len()));
        }
        Ok(count)
    }
    fn checked_editor_click(
        &self,
        sketch: DocumentSketchId,
        target: Option<SketchPickTarget>,
        mode: SelectionClickMode,
    ) -> Result<Option<SketchPick>, SketchError> {
        self.snapshot().sketches.get(sketch)?;
        let picked = target
            .map(|target| self.sketch_pick(sketch, target))
            .transpose()?;
        if picked.is_some() && mode == SelectionClickMode::Toggle {
            self.editor_selection
                .validate_scope(self.document, self.generation(), sketch)?;
        }
        Ok(picked)
    }
}
