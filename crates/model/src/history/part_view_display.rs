// SPDX-License-Identifier: GPL-2.0-only
//! Transient part displays travel with their own history steps (#4118).
use super::{DocumentEditHistory, HistoryError, TransactionToken};
use oblikovati_api::types::PartDesignViewCamera;
use std::collections::BTreeSet;

/// Display state to restore during one view edit; example: use `camera: None` for a visibility-only edit.
#[derive(Clone, Debug, PartialEq)]
pub struct PartViewDisplayState {
    pub hidden_body_keys: BTreeSet<String>,
    pub camera: Option<PartDesignViewCamera>,
}
/// The exact live displays around one view edit; example: `PartViewDisplayTransition { before, after }`.
#[derive(Clone, Debug, PartialEq)]
pub struct PartViewDisplayTransition {
    pub before: PartViewDisplayState,
    pub after: PartViewDisplayState,
}
impl DocumentEditHistory {
    /// Attach display state to the active transaction; example: `history.tag_part_view_display(&token, transition)?`.
    pub fn tag_part_view_display(
        &mut self,
        token: &TransactionToken,
        transition: PartViewDisplayTransition,
    ) -> Result<(), HistoryError> {
        self.require_top(token)?;
        if self.open.len() != 1 {
            return Err(HistoryError::PartDesignViews(
                "nested part view display: expected an outermost transaction".into(),
            ));
        }
        let frame = self
            .open
            .last_mut()
            .ok_or(HistoryError::Empty("part view transaction"))?;
        if frame.part_view_display.is_some() {
            return Err(HistoryError::PartDesignViews(
                "part view display already tagged: expected one transition per transaction".into(),
            ));
        }
        frame.part_view_display = Some(transition);
        Ok(())
    }
    /// Read display state for the next undo; example: `history.undo_part_view_display()`.
    pub fn undo_part_view_display(&self) -> Option<&PartViewDisplayTransition> {
        self.done
            .last()
            .and_then(|step| step.part_view_display.as_ref())
    }
    /// Read display state for the next redo; example: `history.redo_part_view_display()`.
    pub fn redo_part_view_display(&self) -> Option<&PartViewDisplayTransition> {
        self.undone
            .last()
            .and_then(|step| step.part_view_display.as_ref())
    }
    /// Check whether a history jump crosses a display step; example: `history.jump_crosses_part_view(0)`.
    pub fn jump_crosses_part_view(&self, destination: usize) -> bool {
        let current = self.done.len();
        if destination < current {
            return self.done[destination..]
                .iter()
                .any(|step| step.part_view_display.is_some());
        }
        self.undone
            .iter()
            .rev()
            .take(destination - current)
            .any(|step| step.part_view_display.is_some())
    }
}
