// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_api::wire::InferenceOptionsView;
use oblikovati_model::sketch_settings::{
    SketchInferenceError, SketchSettings, with_inference_options,
};
/// Inspect a candidate or typed decline before committing authoring preferences.
pub type SketchInferenceProjectionInput<'a> = Result<&'a SketchSettings, SketchInferenceError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Change preferences without solving or altering existing sketch health.
    /// Example: `session.set_sketch_inference_options_projected(&request, |candidate| encode(candidate?))?;`.
    pub fn set_sketch_inference_options_projected<R, E>(
        &mut self,
        request: &InferenceOptionsView,
        project: impl FnOnce(SketchInferenceProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run_with_recompute(
            "Set sketch inference options",
            |snapshot, _| {
                let candidate =
                    with_inference_options(&snapshot.effective_sketch_settings(), request);
                let reply = project(candidate.as_ref().map_err(Clone::clone))?;
                if let Ok(candidate) = candidate {
                    snapshot.sketch_settings = Some(candidate);
                }
                Ok(reply)
            },
            SketchUpdateMode::Explicit,
        )
    }
}
