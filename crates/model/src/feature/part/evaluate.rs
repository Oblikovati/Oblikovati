// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PartFeatureDefinition, PartFeatureEvaluator, PartFeatureProgramError,
    body_resolution::{resolve_body_face, resolve_body_slot},
};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayRange,
        sketch_prism::SketchPrismError,
        work_geometry::{WorkGeometryError, WorkGeometryEvaluation},
    },
    history::DocumentSnapshot,
};
use std::{borrow::Cow, collections::BTreeMap};
/// Decline a named-face query when a diagnostic evaluator cannot resolve it.
/// Example: `unavailable_planar_face(&key)` reports the offending key.
pub fn unavailable_planar_face(key: &[u8]) -> String {
    format!("face key {key:?}: expected a named planar-face evaluator")
}
/// One blend operation selected before construction.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PartBlendKind {
    Fillet,
    Chamfer,
}
impl PartBlendKind {
    /// Decline an unavailable operation; example: `kind.unsupported()`.
    pub fn unsupported<Failure>(self) -> PartFeatureFailure<Failure> {
        let kind = match self {
            Self::Fillet => "fillet",
            Self::Chamfer => "chamfer",
        };
        PartFeatureFailure::Unsupported(format!(
            "{kind}: expected an available named-edge evaluator"
        ))
    }
}
/// Current ordered diagnostics; these errors are not stored in authored recipes.
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureFailure<Failure> {
    Datum(Box<WorkGeometryError>),
    Metadata(Box<WorkGeometryError>),
    Inputs(SketchPrismError),
    Kernel(Failure),
    BodyReference(String),
    FilletInput(String),
    ChamferInput(String),
    HoleInput(String),
    ShellInput(String),
    MoveBodyInput(String),
    Unsupported(String),
}
/// Derived results from a fresh replay. This does not implement the complete extrude operation or a prefix cache.
#[derive(Clone)]
pub struct PartFeatureEvaluation<Body, Failure> {
    pub(super) datums: WorkGeometryEvaluation,
    pub(super) bodies: BTreeMap<FeatureId, Body>,
    pub(super) body_lineage: super::body_lineage::BodyLineageIndex,
    pub(super) deleted: BTreeMap<FeatureId, Vec<u8>>,
    pub(super) health: Vec<FeatureHealthRecord<PartFeatureFailure<Failure>>>,
    pub(super) source: Option<std::sync::Arc<DocumentSnapshot>>,
}
/// One ordered part result and the solved sketches that produced it.
pub type RecomputedPart<Body, Failure> = (
    PartFeatureEvaluation<Body, Failure>,
    crate::document_sketch::DocumentSketches,
);
pub(super) type PartReplayCandidate<'a, Body, Failure> = (
    PartFeatureEvaluation<Body, Failure>,
    Cow<'a, DocumentSnapshot>,
);
impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    /// Copy the datum results and their ordered health from this replay.
    /// Example: `let datums = replay.datum_evaluation();` for datum queries.
    pub fn datum_evaluation(&self) -> WorkGeometryEvaluation {
        let mut datums = self.datums.clone();
        datums.health = self.health.iter().filter_map(datum_health).collect();
        if let Some(source) = &self.source
            && !datums.has_part_source_for(source)
        {
            datums.bind_part_source(std::sync::Arc::clone(source));
        }
        datums
    }
    /// Resolve a prospective sketch support against this part result.
    /// Example: `part.datum_evaluation_for_face(&backend, &reference)?` before sketch creation.
    pub fn datum_evaluation_for_face<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &self,
        backend: &B,
        reference: &crate::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<WorkGeometryEvaluation, WorkGeometryError> {
        let plane = resolve_body_face(&self.bodies, &self.body_lineage, backend, reference)?;
        let mut datums = self.datum_evaluation();
        datums.insert_sketch_face(reference, plane);
        Ok(datums)
    }
    pub(super) fn empty() -> Self {
        Self {
            datums: WorkGeometryEvaluation::default(),
            bodies: BTreeMap::new(),
            body_lineage: super::body_lineage::BodyLineageIndex::default(),
            deleted: BTreeMap::new(),
            health: Vec::new(),
            source: None,
        }
    }
    /// Inspect the exact body consumed by a successful deletion. Example: `evaluation.deleted_body(delete_id)`.
    pub fn deleted_body(&self, feature: FeatureId) -> Option<&[u8]> {
        self.deleted.get(&feature).map(Vec::as_slice)
    }
    /// Inspect successful current outputs only; example: `evaluation.body(feature_id)` returns none for a failed feature.
    pub fn body(&self, feature: FeatureId) -> Option<&Body> {
        self.bodies.get(&feature)
    }
    /// Find the current body descended from a producer; example: `evaluation.body_from_source(feature)?`.
    pub fn body_from_source(&self, source: FeatureId) -> Result<&Body, String> {
        let owners: Vec<_> = self.body_lineage.candidates(source).collect();
        let [owner] = owners.as_slice() else {
            return Err(format!(
                "body source {source:?}, owners {owners:?}: expected one current body"
            ));
        };
        self.bodies.get(owner).ok_or_else(|| {
            format!("body source {source:?}, owner {owner:?}: expected a current body")
        })
    }
    /// Borrow successful bodies in replay order; example: `evaluation.ordered_bodies().collect::<Vec<_>>()`.
    pub fn ordered_bodies(&self) -> impl Iterator<Item = &Body> {
        self.health
            .iter()
            .filter_map(|record| self.body(record.feature))
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

fn datum_health<Failure>(
    record: &FeatureHealthRecord<PartFeatureFailure<Failure>>,
) -> Option<FeatureHealthRecord<WorkGeometryError>> {
    use crate::feature::FeatureEvaluationStatus;
    let status = match &record.status {
        FeatureEvaluationStatus::Healthy => FeatureEvaluationStatus::Healthy,
        FeatureEvaluationStatus::Suppressed => FeatureEvaluationStatus::Suppressed,
        FeatureEvaluationStatus::ConditionFailed(error) => {
            FeatureEvaluationStatus::ConditionFailed(error.clone())
        }
        FeatureEvaluationStatus::Blocked { drivers } => FeatureEvaluationStatus::Blocked {
            drivers: drivers.clone(),
        },
        FeatureEvaluationStatus::Failed(
            PartFeatureFailure::Datum(error) | PartFeatureFailure::Metadata(error),
        ) => FeatureEvaluationStatus::Failed((**error).clone()),
        FeatureEvaluationStatus::Failed(_) => return None,
    };
    Some(FeatureHealthRecord {
        feature: record.feature,
        status,
    })
}
impl DocumentSnapshot {
    /// Replay every active datum and prism input with local failure quarantine.
    /// Example: `let result = snapshot.evaluate_part_features(&mut backend)?;` never reuses last-good outputs.
    pub fn evaluate_part_features<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
    ) -> Result<PartFeatureEvaluation<B::Body, B::Failure>, PartFeatureProgramError> {
        self.evaluate_part_features_cancellable(backend, || false)
    }
    /// Poll cancellation between entries and discard an incomplete report.
    /// Example: `snapshot.evaluate_part_features_cancellable(&mut backend, || scope.is_superseded())?`.
    pub fn evaluate_part_features_cancellable<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        mut cancelled: impl FnMut() -> bool,
    ) -> Result<PartFeatureEvaluation<B::Body, B::Failure>, PartFeatureProgramError> {
        self.replay_part_with_sketches(backend, None, &mut cancelled)
            .map(|(result, _)| result)
    }
    /// Recompute sketches in feature order using each current datum prefix.
    /// Example: `let (part, sketches) = snapshot.recompute_part_with_sketches(&mut backend, options)?`.
    pub fn recompute_part_with_sketches<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        options: oblikovati_solve::nonlinear::SolveOptions,
    ) -> Result<RecomputedPart<B::Body, B::Failure>, PartFeatureProgramError> {
        self.recompute_part_with_sketches_cancellable(backend, options, || false)
    }
    /// Recompute an ordered part and discard all results on cancellation.
    /// Example: `snapshot.recompute_part_with_sketches_cancellable(&mut backend, options, || false)?`.
    pub fn recompute_part_with_sketches_cancellable<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        options: oblikovati_solve::nonlinear::SolveOptions,
        mut cancelled: impl FnMut() -> bool,
    ) -> Result<RecomputedPart<B::Body, B::Failure>, PartFeatureProgramError> {
        let (part, candidate) =
            self.replay_part_with_sketches(backend, Some(options), &mut cancelled)?;
        Ok((part, candidate.into_owned().sketches))
    }
    fn replay_part_with_sketches<'a, B: PartFeatureEvaluator>(
        &'a self,
        backend: &mut B,
        options: Option<oblikovati_solve::nonlinear::SolveOptions>,
        cancelled: &mut impl FnMut() -> bool,
    ) -> Result<PartReplayCandidate<'a, B::Body, B::Failure>, PartFeatureProgramError> {
        self.replay_part_with_sketches_observed(backend, options, cancelled, &mut |_, _| {})
    }
    pub(super) fn replay_part_with_sketches_observed<'a, B: PartFeatureEvaluator>(
        &'a self,
        backend: &mut B,
        options: Option<oblikovati_solve::nonlinear::SolveOptions>,
        cancelled: &mut impl FnMut() -> bool,
        observe: &mut impl FnMut(
            &PartFeatureEvaluation<B::Body, B::Failure>,
            &[FeatureHealthRecord<PartFeatureFailure<B::Failure>>],
        ),
    ) -> Result<PartReplayCandidate<'a, B::Body, B::Failure>, PartFeatureProgramError> {
        let range = self.part_replay_range()?;
        let sequence =
            super::body_dependencies::with_replacement_reads(self.effective_work_geometry())?;
        let mut controller = FeatureReplayController::new_with_parameters(
            &sequence,
            range,
            vec![],
            &self.parameters,
        )
        .map_err(PartFeatureProgramError::Reads)?;
        let mut evaluation = PartFeatureEvaluation::empty();
        let mut candidate = Cow::Borrowed(self);
        evaluation.replay_observed(
            &mut candidate,
            backend,
            &mut controller,
            options,
            cancelled,
            observe,
        )?;
        evaluation.refresh_sketch_references(&candidate, backend);
        evaluation.health = controller
            .finish()
            .map_err(PartFeatureProgramError::Reads)?;
        if let Some(options) = options {
            let candidate = candidate.to_mut();
            let solved = candidate.solve_pending_document_sketches_with_datums(
                options,
                &mut *cancelled,
                &evaluation.datums,
            );
            require_replay_current(
                solved.report.cancelled || cancelled(),
                evaluation.health.len(),
            )?;
            candidate.sketches = solved.candidate;
        }
        let output = std::sync::Arc::new(candidate.as_ref().clone());
        let input = if self.sketches.same_recipe(&candidate.sketches) {
            std::sync::Arc::clone(&output)
        } else {
            std::sync::Arc::new(self.clone())
        };
        evaluation
            .datums
            .bind_part_sources(input, std::sync::Arc::clone(&output));
        evaluation.source = Some(output);
        Ok((evaluation, candidate))
    }
}
impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn replay<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &mut Cow<'_, DocumentSnapshot>,
        backend: &mut B,
        controller: &mut FeatureReplayController<
            '_,
            PartFeatureDefinition,
            PartFeatureFailure<Failure>,
        >,
        options: Option<oblikovati_solve::nonlinear::SolveOptions>,
        cancelled: &mut impl FnMut() -> bool,
    ) -> Result<(), PartFeatureProgramError> {
        self.replay_observed(
            snapshot,
            backend,
            controller,
            options,
            cancelled,
            &mut |_, _| {},
        )
    }
    pub(super) fn refresh_sketch_references<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &B,
    ) {
        use crate::document_sketch::SketchPlacement;
        let mut faces = BTreeMap::new();
        let mut origins = BTreeMap::new();
        for sketch in snapshot.sketches.sketches() {
            if let Some(crate::document_sketch::SketchFrameOrigin::Topology(reference)) =
                sketch.frame_edit().and_then(|edit| edit.origin.as_ref())
            {
                origins.entry(reference.key().to_vec()).or_insert_with(|| {
                    super::body_resolution::resolve_body_frame_origin(
                        &self.bodies,
                        &self.body_lineage,
                        backend,
                        reference,
                    )
                });
            }
            if let SketchPlacement::Face(reference) = sketch.placement() {
                faces.entry(reference.key().to_vec()).or_insert_with(|| {
                    resolve_body_face(&self.bodies, &self.body_lineage, backend, &reference)
                });
            }
        }
        self.datums.replace_sketch_faces(faces);
        self.datums.replace_sketch_frame_origins(origins);
    }
    pub(super) fn solve_feature_sketch(
        &self,
        snapshot: &mut Cow<'_, DocumentSnapshot>,
        id: FeatureId,
        options: oblikovati_solve::nonlinear::SolveOptions,
        completed: usize,
        cancelled: &mut impl FnMut() -> bool,
    ) -> Result<(), PartFeatureProgramError> {
        let Ok(feature) = snapshot.work_geometry.get(id) else {
            return Ok(());
        };
        let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
            return Ok(());
        };
        let sketch = definition.sketch();
        let candidate = snapshot.to_mut();
        if let Ok(solved) = candidate.solve_one_document_sketch_with_datums(
            sketch,
            options,
            &mut *cancelled,
            &self.datums,
        ) {
            require_replay_current(solved.report.cancelled || cancelled(), completed)?;
            candidate.sketches = solved.candidate;
        }
        Ok(())
    }
    pub(super) fn evaluate_entry<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
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
        if matches!(
            feature.definition(),
            PartFeatureDefinition::SketchPrism(_)
                | PartFeatureDefinition::DeleteBody(_)
                | PartFeatureDefinition::Fillet(_)
                | PartFeatureDefinition::Chamfer(_)
                | PartFeatureDefinition::Hole(_)
                | PartFeatureDefinition::Shell(_)
                | PartFeatureDefinition::MoveBody(_)
        ) {
            feature
                .validate_plane_metadata()
                .map_err(|cause| PartFeatureFailure::Metadata(Box::new(cause)))?;
        }
        match feature.definition() {
            PartFeatureDefinition::SketchPrism(definition) => {
                self.evaluate_prism(snapshot, backend, feature, definition)
            }
            PartFeatureDefinition::DeleteBody(definition) => {
                self.delete_body(backend, id, definition)
            }
            PartFeatureDefinition::Fillet(definition) => {
                self.evaluate_fillet(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Chamfer(definition) => {
                self.evaluate_chamfer(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::MoveBody(definition) => {
                self.evaluate_move_body(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Hole(definition) => {
                self.evaluate_hole(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Shell(definition) => {
                self.evaluate_shell(snapshot, backend, id, definition)
            }
            _ => {
                let bodies = &self.bodies;
                let lineage = &self.body_lineage;
                let resolve_face =
                    |reference: &crate::feature::work_plane::WorkPlaneFaceReference| {
                        resolve_body_face(bodies, lineage, backend, reference)
                    };
                self.datums
                    .evaluate_entry_with_face(
                        &snapshot.work_geometry,
                        id,
                        &snapshot.parameters,
                        &resolve_face,
                    )
                    .map_err(|cause| PartFeatureFailure::Datum(Box::new(cause)))
            }
        }
    }
    fn delete_body<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &B,
        feature: FeatureId,
        definition: &super::DeleteBodyDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let source = definition.source();
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            source,
            definition.key(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.body_lineage.retire(slot);
        self.deleted.insert(feature, definition.key().to_vec());
        Ok(())
    }

    fn evaluate_prism<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: &crate::feature::AuthoredFeature<PartFeatureDefinition>,
        definition: &crate::feature::sketch_prism::SketchPrismDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        if let Some(target) = definition.cut_target() {
            return self.cut_prism(snapshot, backend, feature.id(), target, definition);
        }
        let resolve_face =
            |reference: &_| resolve_body_face(&self.bodies, &self.body_lineage, backend, reference);
        let resolver = crate::feature::work_geometry::DatumFaceResolver {
            datums: &self.datums,
            resolve_face: &resolve_face,
        };
        let input = definition
            .resolve_with_datums(snapshot, &self.datums, &resolver, |_, _| {
                Err("through-all extent: expected Cut against a current target body".into())
            })
            .map_err(PartFeatureFailure::Inputs)?;
        let body = backend
            .evaluate_prism(feature.id(), input)
            .map_err(PartFeatureFailure::Kernel)?;
        self.body_lineage
            .register(feature.id())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.id(), body);
        Ok(())
    }
}

impl DocumentSnapshot {
    pub(super) fn part_replay_range(&self) -> Result<FeatureReplayRange, PartFeatureProgramError> {
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
pub(super) fn require_replay_current(
    cancelled: bool,
    completed: usize,
) -> Result<(), PartFeatureProgramError> {
    if cancelled {
        return Err(PartFeatureProgramError::Cancelled { completed });
    }
    Ok(())
}
