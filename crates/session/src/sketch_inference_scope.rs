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
        self.inference_validation
            .borrow_mut()
            .check(sketch.0, self.generation(), || {
                scope.validate_geometry(geometry)
            })?;
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
        self.inference_validation.get_mut().forget(sketch.0);
        Ok(())
    }
}

/// Reuse validation only while the owning document generation is unchanged (#4107).
#[derive(Default)]
pub(crate) struct ScopeValidationCache {
    entries: std::collections::BTreeMap<u64, (u64, Result<(), SketchError>)>,
}
impl ScopeValidationCache {
    fn check(
        &mut self,
        sketch: u64,
        generation: u64,
        validate: impl FnOnce() -> Result<(), SketchError>,
    ) -> Result<(), SketchError> {
        if let Some((cached_generation, result)) = self.entries.get(&sketch)
            && *cached_generation == generation
        {
            return result.clone();
        }
        let result = validate();
        self.entries.insert(sketch, (generation, result.clone()));
        result
    }
    fn forget(&mut self, sketch: u64) {
        self.entries.remove(&sketch);
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::Cell;
    #[test]
    fn validation_cache_reuses_one_generation_and_invalidates_on_edit_or_scope_change() {
        let mut cache = ScopeValidationCache::default();
        let calls = Cell::new(0);
        for _ in 0..3 {
            cache
                .check(7, 4, || {
                    calls.set(calls.get() + 1);
                    Ok(())
                })
                .unwrap();
        }
        assert_eq!(calls.get(), 1);
        cache
            .check(7, 5, || {
                calls.set(calls.get() + 1);
                Ok(())
            })
            .unwrap();
        cache.forget(7);
        cache
            .check(7, 5, || {
                calls.set(calls.get() + 1);
                Ok(())
            })
            .unwrap();
        assert_eq!(calls.get(), 3);
    }
}
