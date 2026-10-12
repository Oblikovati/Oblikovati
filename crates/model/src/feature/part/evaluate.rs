// SPDX-License-Identifier: GPL-2.0-only
use super::{PartFeatureDefinition, PartFeatureProgramError};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayRange,
        FeatureReplayStep,
        sketch_prism::{ResolvedSketchPrism, SketchPrismError},
        work_geometry::{WorkGeometryError, WorkGeometryEvaluation},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;
/// The model resolves current source inputs; the injected adapter owns kernel construction and validation.
pub trait SketchPrismEvaluator {
    type Body;
    type Failure;
    /// Construct an isolated result from checked inputs; example: `backend.evaluate_prism(input)?`.
    fn evaluate_prism(
        &mut self,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure>;
}
/// Current ordered diagnostics; these errors are not stored in authored recipes.
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureFailure<Failure> {
    Datum(Box<WorkGeometryError>),
    Metadata(Box<WorkGeometryError>),
    Inputs(SketchPrismError),
    Kernel(Failure),
}
/// Derived results from a fresh replay. This does not implement the complete extrude operation or a prefix cache.
pub struct PartFeatureEvaluation<Body, Failure> {
    datums: WorkGeometryEvaluation,
    bodies: BTreeMap<FeatureId, Body>,
    health: Vec<FeatureHealthRecord<PartFeatureFailure<Failure>>>,
}
impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    fn empty() -> Self {
        Self {
            datums: WorkGeometryEvaluation::default(),
            bodies: BTreeMap::new(),
            health: Vec::new(),
        }
    }
    /// Inspect successful current outputs only; example: `evaluation.body(feature_id)` returns none for a failed feature.
    pub fn body(&self, feature: FeatureId) -> Option<&Body> {
        self.bodies.get(&feature)
    }
    /// Inspect complete ordered health; example: `evaluation.health()` includes failures and their blocked readers.
    pub fn health(&self) -> &[FeatureHealthRecord<PartFeatureFailure<Failure>>] {
        &self.health
    }
    /// Transfer one isolated output without cloning native state; example: `evaluation.into_body(feature_id)`.
    pub fn into_body(mut self, feature: FeatureId) -> Option<Body> {
        self.bodies.remove(&feature)
    }
}
impl DocumentSnapshot {
    /// Replay every active datum and prism input with local failure quarantine.
    /// Example: `let result = snapshot.evaluate_part_features(&mut backend)?;` never reuses last-good outputs.
    pub fn evaluate_part_features<B: SketchPrismEvaluator>(
        &self,
        backend: &mut B,
    ) -> Result<PartFeatureEvaluation<B::Body, B::Failure>, PartFeatureProgramError> {
        let range = self.part_replay_range()?;
        let mut controller = FeatureReplayController::new_with_parameters(
            &self.work_geometry,
            range,
            vec![],
            &self.parameters,
        )
        .map_err(PartFeatureProgramError::Reads)?;
        let mut evaluation = PartFeatureEvaluation::empty();
        evaluation.replay(self, backend, &mut controller)?;
        evaluation.health = controller
            .finish()
            .map_err(PartFeatureProgramError::Reads)?;
        Ok(evaluation)
    }
}
impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    fn replay<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        controller: &mut FeatureReplayController<
            '_,
            PartFeatureDefinition,
            PartFeatureFailure<Failure>,
        >,
    ) -> Result<(), PartFeatureProgramError> {
        while let Some(step) = controller
            .advance()
            .map_err(PartFeatureProgramError::Reads)?
        {
            if let FeatureReplayStep::Evaluate(id) = step {
                let outcome = self.evaluate_entry(snapshot, backend, id);
                controller
                    .record_result(id, outcome)
                    .map_err(PartFeatureProgramError::Reads)?;
            }
        }
        Ok(())
    }
    fn evaluate_entry<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        id: FeatureId,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let feature = snapshot.work_geometry.get(id).map_err(|_| {
            PartFeatureFailure::Datum(Box::new(WorkGeometryError::Unavailable {
                feature: id,
                expected: "existing part feature",
            }))
        })?;
        let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
            return self
                .datums
                .evaluate_entry(&snapshot.work_geometry, id, &snapshot.parameters)
                .map_err(|cause| PartFeatureFailure::Datum(Box::new(cause)));
        };
        self.evaluate_prism(snapshot, backend, feature, definition)
    }
    fn evaluate_prism<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: &crate::feature::AuthoredFeature<PartFeatureDefinition>,
        definition: &crate::feature::sketch_prism::SketchPrismDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        feature
            .validate_plane_metadata()
            .map_err(|cause| PartFeatureFailure::Metadata(Box::new(cause)))?;
        let input = definition
            .resolve_with_datums(snapshot, &self.datums)
            .map_err(PartFeatureFailure::Inputs)?;
        let body = backend
            .evaluate_prism(input)
            .map_err(PartFeatureFailure::Kernel)?;
        self.bodies.insert(feature.id(), body);
        Ok(())
    }
}

impl DocumentSnapshot {
    fn part_replay_range(&self) -> Result<FeatureReplayRange, PartFeatureProgramError> {
        self.validate_prism_reads()?;
        self.work_geometry
            .validate_work_plane_reads()
            .map_err(PartFeatureProgramError::Reads)?;
        Ok(FeatureReplayRange {
            start: 0,
            end: self.work_geometry.effective_end(),
        })
    }
}
impl<Body, Failure> crate::feature::work_plane::WorkPlaneResolver
    for PartFeatureEvaluation<Body, Failure>
{
    fn axis(
        &self,
        feature: FeatureId,
    ) -> Result<crate::feature::work_axis::WorkAxisLine, WorkGeometryError> {
        self.datums.axis(feature)
    }
    fn plane(&self, feature: FeatureId) -> Result<crate::sketch::SketchPlane, WorkGeometryError> {
        crate::feature::work_plane::WorkPlaneResolver::plane(&self.datums, feature)
    }
    fn point(&self, feature: FeatureId) -> Result<[f64; 3], WorkGeometryError> {
        self.datums.point(feature)
    }
}
