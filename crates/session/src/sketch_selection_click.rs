// SPDX-License-Identifier: GPL-2.0-only
use crate::selection::{SelectionClickMode, SelectionSet, SketchPick};
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Validate a resolved editor pick before selection changes; this creates no model history.
    /// Example: `session.apply_sketch_click(sketch, Some(pick), mode, false, &mut picks)?;`.
    pub fn apply_sketch_click(
        &self,
        sketch: DocumentSketchId,
        picked: Option<SketchPick>,
        mode: SelectionClickMode,
        tool_active: bool,
        picks: &mut SelectionSet<SketchPick>,
    ) -> Result<Option<usize>, SketchError> {
        if tool_active {
            return Ok(None);
        }
        self.snapshot().sketches.get(sketch)?;
        if let Some(pick) = picked {
            pick.validate_owner(self.document, self.generation(), sketch)?;
            self.validate_additive_picks(sketch, mode, picks)?;
        }
        Ok(picks.apply_click(picked, mode, false))
    }
    fn validate_additive_picks(
        &self,
        sketch: DocumentSketchId,
        mode: SelectionClickMode,
        picks: &SelectionSet<SketchPick>,
    ) -> Result<(), SketchError> {
        if mode == SelectionClickMode::Toggle {
            picks.validate_scope(self.document, self.generation(), sketch)?;
        }
        Ok(())
    }
}
