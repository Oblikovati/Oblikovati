// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkspaceError, WorkspaceObserver, WorkspaceSession};
use crate::{CommandError, SketchInferenceProjectionInput, TransactionObserver};
use oblikovati_api::wire::InferenceOptionsView;
use oblikovati_model::sketch_settings::{SketchSettings, with_inference_options};
use std::{error::Error, fmt};
/// An owner lookup, document command, or response projection declined the edit.
#[derive(Debug)]
pub enum SketchInferenceCommandError<E> {
    Workspace(WorkspaceError),
    Command(CommandError<E>),
    Reply(E),
}
impl<E: fmt::Display> fmt::Display for SketchInferenceCommandError<E> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Workspace(error) => error.fmt(output),
            Self::Command(error) => error.fmt(output),
            Self::Reply(error) => error.fmt(output),
        }
    }
}
impl<E: Error + 'static> Error for SketchInferenceCommandError<E> {}
impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Read the active document's settings, or shared fallback when none is active.
    /// Example: `let settings = workspace.sketch_inference_settings()?;`.
    pub fn sketch_inference_settings(&self) -> Result<SketchSettings, WorkspaceError> {
        let Some(document) = self.active else {
            return Ok(self.fallback_sketch_settings.clone());
        };
        Ok(self
            .session(document)?
            .snapshot()
            .effective_sketch_settings())
    }
    /// Prepare a reply before changing the active document or shared fallback.
    /// Example: `workspace.set_sketch_inference_options_projected(&request, |candidate| encode(candidate?))?;`.
    pub fn set_sketch_inference_options_projected<R, E>(
        &mut self,
        request: &InferenceOptionsView,
        project: impl FnOnce(SketchInferenceProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, SketchInferenceCommandError<E>> {
        let Some(document) = self.active else {
            return self.set_fallback_inference(request, project);
        };
        self.commands(document)
            .map_err(SketchInferenceCommandError::Workspace)?
            .session
            .set_sketch_inference_options_projected(request, project)
            .map_err(SketchInferenceCommandError::Command)
    }
    fn set_fallback_inference<R, E>(
        &mut self,
        request: &InferenceOptionsView,
        project: impl FnOnce(SketchInferenceProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, SketchInferenceCommandError<E>> {
        let candidate = with_inference_options(&self.fallback_sketch_settings, request);
        let reply = project(candidate.as_ref().map_err(Clone::clone))
            .map_err(SketchInferenceCommandError::Reply)?;
        if let Ok(candidate) = candidate {
            self.fallback_sketch_settings = candidate;
        }
        Ok(reply)
    }
}
