// SPDX-License-Identifier: GPL-2.0-only
//! Session-only candidate geometry for each 2D sketch (#4107).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_api::types::{INFERENCE_SCOPE_CURRENT_COMMAND, SketchInferenceScope};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchError, SketchInferenceCandidates},
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Read a sketch's current valid scope; example: `session.sketch_inference_scope(id)?`.
    pub fn sketch_inference_scope(
        &self,
        sketch: DocumentSketchId,
    ) -> Result<SketchInferenceCandidates, SketchError> {
        let geometry = self.snapshot().sketches.get(sketch)?.geometry();
        let scope = match self.inference_scopes.get(&sketch.0) {
            Some(scope) => scope.clone(),
            None => SketchInferenceCandidates::new(INFERENCE_SCOPE_CURRENT_COMMAND, &[])?,
        };
        scope.validate_geometry(geometry)?;
        Ok(scope)
    }

    /// Replace the scope only after all identities validate; example: `session.set_sketch_inference_scope(id, mode, &entities)?`.
    pub fn set_sketch_inference_scope(
        &mut self,
        sketch: DocumentSketchId,
        mode: SketchInferenceScope,
        entities: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        let candidate = SketchInferenceCandidates::new(mode, entities)?;
        candidate.validate_geometry(self.snapshot().sketches.get(sketch)?.geometry())?;
        self.inference_scopes.insert(sketch.0, candidate);
        Ok(())
    }
}
