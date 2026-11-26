// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::{resolve_body_face, resolve_body_slot},
};
use crate::{
    feature::{
        FeatureId,
        sketch_prism::{SketchPrismDefinition, SketchPrismTarget},
        work_geometry::DatumFaceResolver,
        work_plane::WorkPlaneFaceReference,
    },
    history::DocumentSnapshot,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn replace_with_prism<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        target: &SketchPrismTarget,
        definition: &SketchPrismDefinition,
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
        let resolver = DatumFaceResolver {
            datums: &self.datums,
            resolve_face: &|reference: &WorkPlaneFaceReference| {
                resolve_body_face(&self.bodies, &self.body_lineage, backend, reference)
            },
        };
        let input = definition
            .resolve_with_datums(snapshot, &self.datums, &resolver, |plane, direction| {
                backend.prism_through_all_span(body, plane, direction)
            })
            .map_err(PartFeatureFailure::Inputs)?;
        let result = backend.evaluate_prism_boolean(feature, body, input, target.operation())?;
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
