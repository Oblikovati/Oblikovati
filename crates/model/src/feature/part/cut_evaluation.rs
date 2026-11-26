// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{
    FeatureId,
    sketch_prism::{ResolvedSketchPrism, SketchPrismCutTarget},
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn cut_prism<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        target: &SketchPrismCutTarget,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            target.source(),
            target.key(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let body = &self.bodies[&slot];
        let result = backend.evaluate_prism_cut(feature, body, input)?;
        backend
            .verify_body_reference(&result, target.key())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature, result);
        Ok(())
    }
}
