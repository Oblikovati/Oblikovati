// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::{
    SketchContinuityWatch, TransactionObserver,
    workspace::{WorkspaceFocusIdentity, WorkspaceObserver, WorkspaceSession},
};

/// Borrowable desktop edit evidence that history restore cannot revive after interruption.
#[derive(Clone)]
pub struct SketchInspectionEdit {
    document: u64,
    watch: SketchContinuityWatch,
    focus: WorkspaceFocusIdentity,
}
impl SketchInspectionEdit {
    /// Retain the owner's existing edit guards without recording a model change.
    /// Example: `SketchInspectionEdit::new(document, watch.clone(), focus.clone())`.
    pub fn new(document: u64, watch: SketchContinuityWatch, focus: WorkspaceFocusIdentity) -> Self {
        Self {
            document,
            watch,
            focus,
        }
    }
    pub(crate) fn current<O: TransactionObserver, W: WorkspaceObserver>(
        &self,
        workspace: &WorkspaceSession<O, W>,
    ) -> Option<(u64, DocumentSketchId)> {
        (self.watch.is_live()
            && workspace.retains_focus(&self.focus)
            && workspace.active_document() == Some(self.document))
        .then(|| (self.document, self.watch.sketch()))
    }
}
