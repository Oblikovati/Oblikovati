// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Modify Thicken/Offset features (solid NewSolid/Boolean and surface OffsetSrf, #3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{FeatureId, thicken_offset::ThickenOffsetDefinition};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_thicken_offset<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &ThickenOffsetDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let target_slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.target_body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let target = self.bodies.get(&target_slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "thicken/offset target slot {target_slot:?}: expected an active body"
            ))
        })?;
        let result = backend.evaluate_thicken_offset(feature, target, definition.resolve())?;
        if definition.replaces_target_body() {
            backend
                .verify_body_reference(&result, definition.target_body())
                .map_err(PartFeatureFailure::BodyReference)?;
            self.body_lineage
                .replace(target_slot, feature)
                .map_err(PartFeatureFailure::BodyReference)?;
            self.bodies.remove(&target_slot);
            self.bodies.insert(feature.into(), result);
        } else {
            self.body_lineage
                .register(feature)
                .map_err(PartFeatureFailure::BodyReference)?;
            self.bodies.insert(feature.into(), result);
        }
        Ok(())
    }
}
