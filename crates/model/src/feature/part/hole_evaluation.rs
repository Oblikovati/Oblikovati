// SPDX-License-Identifier: GPL-2.0-only
//! Validate named Hole inputs before replacing a current body (#4093).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, hole::HoleDefinition},
    history::DocumentSnapshot,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_hole<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &HoleDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve(&snapshot.parameters)
            .map_err(PartFeatureFailure::HoleInput)?;
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "hole body slot {slot:?}: expected a current body"
            ))
        })?;
        backend
            .resolve_planar_face(before, definition.face())
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = backend.evaluate_hole(feature, before, definition.face(), input)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        backend
            .resolve_planar_face(&result, definition.face())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }

    pub(super) fn evaluate_slot<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &crate::feature::slot::SlotDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve(&snapshot.parameters)
            .map_err(PartFeatureFailure::HoleInput)?;
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "slot body slot {slot:?}: expected a current body"
            ))
        })?;
        backend
            .resolve_planar_face(before, definition.face())
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = backend.evaluate_slot(feature, before, definition.face(), input)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        backend
            .resolve_planar_face(&result, definition.face())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }

    pub(super) fn evaluate_thread<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &crate::feature::thread::ThreadDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve(&snapshot.parameters)
            .map_err(PartFeatureFailure::ThreadInput)?;
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "thread body slot {slot:?}: expected a current body"
            ))
        })?;
        let result = backend.evaluate_thread(feature, before, input)?;
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
