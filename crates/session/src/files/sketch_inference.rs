// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{SketchInferenceProjectionInput, workspace::SketchInferenceCommandError};
use oblikovati_api::wire::InferenceOptionsView;
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Use the retained document owner or workspace fallback; file identity stays outside the edit.
    /// Example: `files.set_sketch_inference_options_projected(&request, |candidate| encode(candidate?))?;`.
    pub fn set_sketch_inference_options_projected<R, E>(
        &mut self,
        request: &InferenceOptionsView,
        project: impl FnOnce(SketchInferenceProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, SketchInferenceCommandError<E>> {
        self.workspace
            .set_sketch_inference_options_projected(request, project)
    }
}
