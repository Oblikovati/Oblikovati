// SPDX-License-Identifier: GPL-2.0-only
//! Access validated history candidates and their accepted generation transitions.
use super::*;

impl DocumentEditHistory {
    /// Borrow the next undo recipe after validating the cursor.
    /// Example: `history.undo_candidate()?` before capturing reusable output.
    pub fn undo_candidate(&self) -> Result<&DocumentSnapshot, HistoryError> {
        self.undo_label()?;
        Ok(&self.done.last().ok_or(HistoryError::Empty("undo"))?.before)
    }
    /// Borrow the next redo recipe after validating the cursor.
    /// Example: `history.redo_candidate()?` before capturing reusable output.
    pub fn redo_candidate(&self) -> Result<&DocumentSnapshot, HistoryError> {
        self.redo_label()?;
        Ok(&self.undone.last().ok_or(HistoryError::Empty("redo"))?.after)
    }
    /// Undo and retain proof of the accepted generation transition.
    /// Example: `history.undo_with_transition()?` after capturing a candidate.
    pub fn undo_with_transition(&mut self) -> Result<EvaluationTransition, HistoryError> {
        let source = self.evaluation_scope();
        self.undo()?;
        Ok(EvaluationTransition::new(source, self.evaluation_scope()))
    }
    /// Redo and retain proof of the accepted generation transition.
    /// Example: `history.redo_with_transition()?` after capturing a candidate.
    pub fn redo_with_transition(&mut self) -> Result<EvaluationTransition, HistoryError> {
        let source = self.evaluation_scope();
        self.redo()?;
        Ok(EvaluationTransition::new(source, self.evaluation_scope()))
    }
}
