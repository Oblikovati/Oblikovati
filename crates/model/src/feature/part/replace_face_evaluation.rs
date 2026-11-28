// SPDX-License-Identifier: GPL-2.0-only
//! Validate named Replace Face inputs and resolve replacement face bodies before replacing a current body (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{FeatureId, replace_face::ReplaceFaceDefinition};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_replace_face<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &ReplaceFaceDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve()
            .map_err(PartFeatureFailure::ReplaceFaceInput)?;
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let mut new_slots = Vec::with_capacity(definition.new_faces().len());
        for (&source, target) in definition
            .new_face_body_sources()
            .iter()
            .zip(definition.new_faces())
        {
            let new_slot = resolve_body_slot(
                &self.bodies,
                &self.body_lineage,
                backend,
                source,
                &target.body,
            )
            .map_err(PartFeatureFailure::BodyReference)?;
            new_slots.push(new_slot);
        }
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "replace face body slot {slot:?}: expected a current body"
            ))
        })?;
        let mut new_bodies = Vec::with_capacity(new_slots.len());
        for new_slot in &new_slots {
            let body = self.bodies.get(new_slot).ok_or_else(|| {
                PartFeatureFailure::BodyReference(format!(
                    "replace face replacement body slot {new_slot:?}: expected a current body"
                ))
            })?;
            new_bodies.push(body);
        }
        let result = backend.evaluate_replace_face(feature, before, &new_bodies, input)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}
