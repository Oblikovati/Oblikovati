// SPDX-License-Identifier: GPL-2.0-only
//! Keep solved sketch consequences out of an otherwise empty undo group.
use super::{DocumentEditHistory, DocumentSnapshot, EditError, OpenTransaction, TransactionToken};
use crate::document_sketch::DocumentSketches;

impl DocumentEditHistory {
    /// Apply solved sketch state inside the active group without making it an authored edit.
    /// Example: `history.edit_sketch_consequence::<Error>(&token, solved)?`.
    pub fn edit_sketch_consequence<E>(
        &mut self,
        token: &TransactionToken,
        sketches: DocumentSketches,
    ) -> Result<bool, EditError<E>> {
        self.require_top(token).map_err(EditError::History)?;
        if self.current.sketches.same_evaluated_state(&sketches) {
            return Ok(false);
        }
        let frame = self.open.last().expect("validated frame");
        let baseline = frame.derived_baseline.as_ref().unwrap_or(&frame.before);
        let only_consequences = same_history_recipe(baseline, &self.current);
        self.edit(token, |snapshot, _| {
            snapshot.sketches = sketches;
            Ok::<(), E>(())
        })?;
        if only_consequences {
            self.open
                .last_mut()
                .expect("validated frame")
                .derived_baseline = Some(self.current.clone());
        }
        Ok(true)
    }
    pub(super) fn inherit_derived_baseline(&mut self, frame: &OpenTransaction) {
        let Some(derived) = &frame.derived_baseline else {
            return;
        };
        if !same_history_recipe(derived, &self.current) {
            return;
        }
        let Some(parent) = self.open.last_mut() else {
            return;
        };
        let baseline = parent.derived_baseline.as_ref().unwrap_or(&parent.before);
        if same_history_recipe(baseline, &frame.before) {
            parent.derived_baseline = Some(self.current.clone());
        }
    }
}
pub(super) fn same_history_recipe(first: &DocumentSnapshot, second: &DocumentSnapshot) -> bool {
    first
        .work_geometry
        .same_work_plane_recipe(&second.work_geometry)
        && first.sketches.same_recipe(&second.sketches)
        && first.parameters.same_recipe(&second.parameters)
        && first.same_metadata_recipe(second)
}
