// SPDX-License-Identifier: GPL-2.0-only
use crate::selection::{BrowserSketchPick, SelectionClickMode, SelectionSet};
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Resolve a browser row against its current document, independent of edit mode.
    /// Example: `let pick = session.browser_sketch_pick(sketch)?;`.
    pub fn browser_sketch_pick(
        &self,
        sketch: DocumentSketchId,
    ) -> Result<BrowserSketchPick, SketchError> {
        self.snapshot().sketches.get(sketch)?;
        Ok(BrowserSketchPick {
            document: self.document,
            sketch,
        })
    }
    /// Apply browser selection without editing the model or creating undo history.
    /// Tools retain their clicks; example: `session.apply_browser_sketch_click(Some(pick), mode, false, &mut picks)?;`.
    pub fn apply_browser_sketch_click(
        &self,
        picked: Option<BrowserSketchPick>,
        mode: SelectionClickMode,
        tool_active: bool,
        picks: &mut SelectionSet<BrowserSketchPick>,
    ) -> Result<Option<usize>, SketchError> {
        if tool_active {
            return Ok(None);
        }
        if let Some(pick) = picked {
            self.validate_browser_sketch_pick(pick)?;
            self.validate_browser_additions(mode, picks)?;
        }
        Ok(picks.apply_click(picked, mode, false))
    }
    fn validate_browser_additions(
        &self,
        mode: SelectionClickMode,
        picks: &SelectionSet<BrowserSketchPick>,
    ) -> Result<(), SketchError> {
        if mode != SelectionClickMode::Toggle {
            return Ok(());
        }
        for existing in picks.items() {
            self.validate_browser_sketch_pick(*existing)?;
        }
        Ok(())
    }
    fn validate_browser_sketch_pick(&self, pick: BrowserSketchPick) -> Result<(), SketchError> {
        if pick.document != self.document {
            return Err(SketchError::Definition(format!(
                "browser sketch pick {pick:?}: expected document {}",
                self.document
            )));
        }
        self.snapshot().sketches.get(pick.sketch)?;
        Ok(())
    }
}
