// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate multi-body Part Modify Combine features with optional tool-body retention (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{FeatureId, combine::CombineDefinition};
use std::collections::BTreeSet;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_combine<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &CombineDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let base_slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.base_source(),
            definition.base_body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let mut seen = BTreeSet::new();
        seen.insert(base_slot);
        let mut tool_slots = Vec::with_capacity(definition.tool_bodies().len());
        for (&source, key) in definition
            .tool_sources()
            .iter()
            .zip(definition.tool_bodies())
        {
            let slot = resolve_body_slot(&self.bodies, &self.body_lineage, backend, source, key)
                .map_err(PartFeatureFailure::BodyReference)?;
            if !seen.insert(slot) {
                return Err(PartFeatureFailure::BodyReference(format!(
                    "combine tool slot {slot:?}: expected distinct tool bodies separate from base"
                )));
            }
            tool_slots.push(slot);
        }
        let base = self.bodies.get(&base_slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "combine base slot {base_slot:?}: expected a current body"
            ))
        })?;
        let mut tools = Vec::with_capacity(tool_slots.len());
        for &slot in &tool_slots {
            let body = self.bodies.get(&slot).ok_or_else(|| {
                PartFeatureFailure::BodyReference(format!(
                    "combine tool slot {slot:?}: expected a current body"
                ))
            })?;
            tools.push(body);
        }
        let result = backend.evaluate_combine(feature, base, &tools, definition.resolve())?;
        backend
            .verify_body_reference(&result, definition.base_body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(base_slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&base_slot);
        self.bodies.insert(feature.into(), result);
        if !definition.keep_toolbodies() {
            for slot in tool_slots {
                self.bodies.remove(&slot);
                self.body_lineage.retire(slot);
            }
        }
        Ok(())
    }
}
