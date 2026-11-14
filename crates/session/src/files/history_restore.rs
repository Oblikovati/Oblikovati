// SPDX-License-Identifier: GPL-2.0-only
use super::FileWorkspaceError;
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, HistoryError, SketchReservationMarks},
    identity::{RecipeIdAllocator, RecipeIdError},
};
#[cfg(test)]
mod tests;

pub(super) fn restore_history(
    snapshot: DocumentSnapshot,
    ids: RecipeIdAllocator,
    feature_mark: u64,
    sketch_marks: SketchReservationMarks,
    derived_mark: i64,
) -> Result<DocumentEditHistory, FileWorkspaceError> {
    let history = DocumentEditHistory::new(snapshot, ids).map_err(FileWorkspaceError::History)?;
    restore_namespace_marks(history, feature_mark, sketch_marks, derived_mark)
}
fn restore_namespace_marks(
    history: DocumentEditHistory,
    feature_mark: u64,
    sketch_marks: SketchReservationMarks,
    derived_mark: i64,
) -> Result<DocumentEditHistory, FileWorkspaceError> {
    let history = history
        .with_feature_reservations(feature_mark)
        .map_err(reservation_error)?;
    let history = history
        .with_sketch_reservations(sketch_marks)
        .map_err(reservation_error)?;
    history
        .with_derived_reservations(derived_mark)
        .map_err(reservation_error)
}
fn reservation_error(error: RecipeIdError) -> FileWorkspaceError {
    FileWorkspaceError::History(HistoryError::Identity(error))
}
