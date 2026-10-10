// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentEditHistory {
    /// List committed labels oldest first; example: `history.undo_labels()`.
    pub fn undo_labels(&self) -> Vec<&str> {
        self.done.iter().map(|step| step.label.as_str()).collect()
    }

    /// List undone labels in redo order; example: `history.redo_labels()`.
    pub fn redo_labels(&self) -> Vec<&str> {
        self.undone
            .iter()
            .rev()
            .map(|step| step.label.as_str())
            .collect()
    }

    /// Restore the latest committed snapshot; example: `let label = history.undo()?;`.
    pub fn undo(&mut self) -> Result<String, HistoryError> {
        self.require_closed()?;
        let generation = self.next_generation()?;
        let step = self.done.pop().ok_or(HistoryError::Empty("undo"))?;
        self.current = step.before.clone();
        self.generation = generation;
        let label = step.label.clone();
        self.undone.push(step);
        Ok(label)
    }

    /// Reapply the most recently undone snapshot; example: `let label = history.redo()?;`.
    pub fn redo(&mut self) -> Result<String, HistoryError> {
        self.require_closed()?;
        let generation = self.next_generation()?;
        let step = self.undone.pop().ok_or(HistoryError::Empty("redo"))?;
        self.current = step.after.clone();
        self.generation = generation;
        let label = step.label.clone();
        self.done.push(step);
        Ok(label)
    }

    fn require_closed(&self) -> Result<(), HistoryError> {
        if let Some(frame) = self.open.last() {
            return Err(HistoryError::Open(frame.token.clone()));
        }
        Ok(())
    }
}
