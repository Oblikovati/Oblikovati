// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Stitch features across two or more selected surface bodies (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{FeatureId, stitch::StitchDefinition};
use std::collections::BTreeSet;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_stitch<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &StitchDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let mut seen = BTreeSet::new();
        let mut slots = Vec::with_capacity(definition.surfaces().len());
        for (&source, key) in definition
            .surface_sources()
            .iter()
            .zip(definition.surfaces())
        {
            let slot = resolve_body_slot(&self.bodies, &self.body_lineage, backend, source, key)
                .map_err(PartFeatureFailure::BodyReference)?;
            if !seen.insert(slot) {
                return Err(PartFeatureFailure::BodyReference(format!(
                    "stitch surface slot {slot:?}: expected distinct surface bodies"
                )));
            }
            slots.push(slot);
        }
        let mut surfaces = Vec::with_capacity(slots.len());
        for &slot in &slots {
            let body = self.bodies.get(&slot).ok_or_else(|| {
                PartFeatureFailure::BodyReference(format!(
                    "stitch surface slot {slot:?}: expected a current surface body"
                ))
            })?;
            surfaces.push(body);
        }
        let result = backend.evaluate_stitch(feature, &surfaces, definition.resolve())?;
        self.body_lineage
            .register(feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}
