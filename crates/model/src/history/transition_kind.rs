// SPDX-License-Identifier: GPL-2.0-only
//! Classify history without depending on user-visible labels (#4122).
use super::DocumentEditHistory;

/// Classifies a committed transition independently of its user-visible label.
/// Example: `HistoryTransitionKind::ModelStateTable` requires staged replay.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum HistoryTransitionKind {
    Ordinary,
    ModelStateTable,
}
impl DocumentEditHistory {
    /// Return the next undo kind; example: `history.undo_transition_kind()`.
    pub fn undo_transition_kind(&self) -> Option<HistoryTransitionKind> {
        self.done.last().map(|step| step.kind)
    }
    /// Return the next redo kind; example: `history.redo_transition_kind()`.
    pub fn redo_transition_kind(&self) -> Option<HistoryTransitionKind> {
        self.undone.last().map(|step| step.kind)
    }

    /// Check whether a history move crosses a model-state table transition.
    /// Example: `history.prepared_jump_required(0)?` checks the move to the initial state.
    pub fn prepared_jump_required(&self, position: i64) -> Result<bool, String> {
        let total = self.done.len() + self.undone.len();
        let destination = usize::try_from(position)
            .ok()
            .filter(|value| *value <= total)
            .ok_or_else(|| {
                format!("history position {position}: expected a value from 0 to {total}")
            })?;
        if destination < self.done.len() {
            return Ok(self.done[destination..]
                .iter()
                .any(|step| step.kind == HistoryTransitionKind::ModelStateTable));
        }
        Ok(self
            .undone
            .iter()
            .rev()
            .take(destination - self.done.len())
            .any(|step| step.kind == HistoryTransitionKind::ModelStateTable))
    }
}
