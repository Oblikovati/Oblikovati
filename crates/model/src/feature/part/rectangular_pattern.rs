// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PartFeatureDefinition, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::{resolve_body_face, resolve_body_slot},
};
use crate::{
    feature::{
        FeatureId,
        rectangular_pattern::{PatternPrismSource, RectangularPatternDefinition},
        work_geometry::DatumFaceResolver,
        work_plane::WorkPlaneFaceReference,
    },
    history::DocumentSnapshot,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_rectangular_pattern<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &RectangularPatternDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let mut slot = None;
        let mut sources = Vec::new();
        let mut target_keys = Vec::new();
        let resolver = DatumFaceResolver {
            datums: &self.datums,
            resolve_face: &|reference: &WorkPlaneFaceReference| {
                resolve_body_face(&self.bodies, &self.body_lineage, backend, reference)
            },
        };
        for source in &definition.recipe().sources {
            let authored = snapshot
                .work_geometry
                .get(*source)
                .map_err(|cause| PartFeatureFailure::Unsupported(cause.to_string()))?;
            let PartFeatureDefinition::SketchPrism(prism) = authored.definition() else {
                return Err(PartFeatureFailure::Unsupported(format!(
                    "rectangular pattern source {source:?}: expected an extrusion"
                )));
            };
            let target = prism.boolean_target().ok_or_else(|| {
                PartFeatureFailure::Unsupported(format!(
                    "rectangular pattern source {source:?}: expected a Join or Cut extrusion"
                ))
            })?;
            let current = resolve_body_slot(
                &self.bodies,
                &self.body_lineage,
                backend,
                target.source(),
                target.key(),
            )
            .map_err(PartFeatureFailure::BodyReference)?;
            if slot.is_some_and(|other| current != other)
                || !self
                    .body_lineage
                    .candidates(*source)
                    .any(|id| id == current)
            {
                return Err(PartFeatureFailure::BodyReference(format!(
                    "rectangular pattern source {source:?}: expected a healthy source on the common current body"
                )));
            }
            let body = &self.bodies[&current];
            let input = prism
                .resolve_with_datums(snapshot, &self.datums, &resolver, |plane, direction| {
                    backend.prism_through_all_span(body, plane, direction)
                })
                .map_err(PartFeatureFailure::Inputs)?;
            sources.push(PatternPrismSource {
                source: *source,
                prism: input,
                operation: target.operation(),
            });
            target_keys.push(target.key());
            slot = Some(current);
        }
        let slot = slot.ok_or_else(|| {
            PartFeatureFailure::BodyReference("rectangular pattern: expected a source body".into())
        })?;
        let occurrences = definition
            .occurrences()
            .map_err(PartFeatureFailure::Unsupported)?;
        let result = backend.evaluate_rectangular_pattern(
            feature,
            &self.bodies[&slot],
            &sources,
            &occurrences,
        )?;
        for key in target_keys {
            backend
                .verify_body_reference(&result, key)
                .map_err(PartFeatureFailure::BodyReference)?;
        }
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature, result);
        Ok(())
    }
}
