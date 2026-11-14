// SPDX-License-Identifier: GPL-2.0-only
//! Transient editor transitions share the document's ordered history without changing its recipe.
use super::{DocumentEditHistory, HistoryError, HistoryStep};
use crate::document_sketch::DocumentSketchId;

/// Editor identity on either side of one history step; example: a finish goes from `Some(id)` to `None`.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchEditTransition {
    pub before: Option<DocumentSketchId>,
    pub after: Option<DocumentSketchId>,
}
impl DocumentEditHistory {
    /// Associate the latest sketch creation with its editor entry; example: `history.tag_sketch_creation(id)?`.
    pub fn tag_sketch_creation(&mut self, sketch: DocumentSketchId) -> Result<(), HistoryError> {
        self.require_closed()?;
        let step = self
            .done
            .last_mut()
            .ok_or(HistoryError::Empty("sketch creation"))?;
        if step.sketch_edit.is_some()
            || step.before.sketches.get(sketch).is_ok()
            || step.after.sketches.get(sketch).is_err()
        {
            return Err(HistoryError::SketchEditTransition(format!(
                "sketch creation {sketch:?}: expected an untagged latest step that creates the sketch"
            )));
        }
        step.sketch_edit = Some(SketchEditTransition {
            before: None,
            after: Some(sketch),
        });
        Ok(())
    }
    /// Validate a mode-only finish; example: `history.validate_sketch_finish(id)?` before observer notification.
    pub fn validate_sketch_finish(&self, sketch: DocumentSketchId) -> Result<(), HistoryError> {
        self.require_closed()?;
        self.next_generation()?;
        self.current.resolve_sketch_plane(sketch).map_err(|cause| {
            HistoryError::SketchEditTransition(format!(
                "sketch finish {sketch:?}: expected a current sketch plane: {cause}"
            ))
        })?;
        Ok(())
    }
    /// Record a finish without altering the saved recipe; example: `history.record_sketch_finish(id)?`.
    pub fn record_sketch_finish(&mut self, sketch: DocumentSketchId) -> Result<(), HistoryError> {
        self.validate_sketch_finish(sketch)?;
        self.done.push(HistoryStep {
            label: "Finish Sketch".into(),
            saved: false,
            before: self.current.clone(),
            after: self.current.clone(),
            sketch_edit: Some(SketchEditTransition {
                before: Some(sketch),
                after: None,
            }),
        });
        self.undone.clear();
        Ok(())
    }
    /// Read the transition a validated undo will cross; example: `history.undo_sketch_transition()`.
    pub fn undo_sketch_transition(&self) -> Option<SketchEditTransition> {
        self.done.last().and_then(|step| step.sketch_edit)
    }
    /// Read the transition a validated redo will cross; example: `history.redo_sketch_transition()`.
    pub fn redo_sketch_transition(&self) -> Option<SketchEditTransition> {
        self.undone.last().and_then(|step| step.sketch_edit)
    }
}
