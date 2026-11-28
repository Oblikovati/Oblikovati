// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PartFeatureDefinition, PartFeatureEvaluator, PartFeatureProgramError,
    body_resolution::resolve_body_face,
};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController,
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
    SweepInput(Box<crate::feature::sketch_sweep::SketchSweepError>),
    RevolveInput(Box<crate::feature::sketch_revolve::SketchRevolveError>),
    CoilInput(String),
    EmbossInput(String),
    RibInput(String),
    DecalInput(String),
    DerivedPartInput(String),
    ImportedModelInput(String),
    UnwrapInput(String),
    LoftInput(String),
    Kernel(Failure),
    BodyReference(String),
    FilletInput(String),
    ChamferInput(String),
    HoleInput(String),
    DraftInput(String),
    ThreadInput(String),
    DeleteFaceInput(String),
    ReplaceFaceInput(String),
    DirectEditInput(String),
    SplitInput(String),
    MarkInput(String),
    FinishInput(String),
    RuledSurfaceInput(String),
    ShellInput(String),
    MoveBodyInput(String),
    Unsupported(String),
}
/// Derived results from a fresh replay. This does not implement the complete extrude operation or a prefix cache.
#[derive(Clone)]
pub struct PartFeatureEvaluation<Body, Failure> {
    pub(super) datums: WorkGeometryEvaluation,
    pub(super) bodies: BTreeMap<super::body_lineage::BodySlot, Body>,
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
    /// Inspect a feature's current primary body; example: `evaluation.body(feature_id)` returns none for a failed feature.
    pub fn body(&self, feature: FeatureId) -> Option<&Body> {
        self.bodies.get(&feature.into())
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
        self.ordered_body_outputs().map(|(_, body)| body)
    }
    /// Enumerate each live body with its producer, in Part replay and output order.
    /// Example: `evaluation.ordered_body_outputs().collect::<Vec<_>>()`.
    pub fn ordered_body_outputs(&self) -> impl Iterator<Item = (FeatureId, &Body)> {
        self.health.iter().flat_map(|record| {
            self.bodies_for_feature(record.feature)
                .map(|body| (record.feature, body))
        })
    }
    /// Enumerate the current outputs produced by one feature.
    /// Example: `evaluation.bodies_for_feature(feature).count()`.
    pub fn bodies_for_feature(&self, feature: FeatureId) -> impl Iterator<Item = &Body> {
        self.bodies
            .range(super::body_lineage::BodySlot::feature_range(feature))
            .map(|(_, body)| body)
    }
    /// Inspect complete ordered health; example: `evaluation.health()` includes failures and their blocked readers.
    pub fn health(&self) -> &[FeatureHealthRecord<PartFeatureFailure<Failure>>] {
        &self.health
    }
    /// Transfer the primary output without cloning native state; example: `evaluation.into_body(feature_id)`.
    pub fn into_body(mut self, feature: FeatureId) -> Option<Body> {
        self.bodies.remove(&feature.into())
    }
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
        if feature.definition().requires_plain_plane_metadata() {
            feature
                .validate_plane_metadata()
                .map_err(|cause| PartFeatureFailure::Metadata(Box::new(cause)))?;
        }
        match feature.definition() {
            PartFeatureDefinition::SketchPrism(definition) => {
                self.evaluate_prism(snapshot, backend, feature, definition)
            }
            PartFeatureDefinition::SketchLoft(definition) => {
                self.evaluate_loft(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::SketchRevolve(definition) => {
                self.evaluate_revolve(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::SketchCoil(definition) => {
                self.evaluate_coil(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::SketchEmboss(definition) => {
                self.evaluate_emboss(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::SketchRib(definition) => {
                self.evaluate_rib(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::SketchDecal(definition) => {
                self.evaluate_decal(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::DerivedPart(definition) => {
                self.evaluate_derived_part(backend, id, definition)
            }
            PartFeatureDefinition::ImportedModel(definition) => {
                self.evaluate_imported_model(backend, id, definition)
            }
            PartFeatureDefinition::Unwrap(definition) => {
                self.evaluate_unwrap(backend, id, definition)
            }
            PartFeatureDefinition::SketchSweep(definition) => {
                self.evaluate_sweep(snapshot, backend, id, definition)
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
            PartFeatureDefinition::Slot(definition) => {
                self.evaluate_slot(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Combine(definition) => {
                self.evaluate_combine(backend, id, definition)
            }
            PartFeatureDefinition::Draft(definition) => {
                self.evaluate_draft(backend, id, definition)
            }
            PartFeatureDefinition::Thread(definition) => {
                self.evaluate_thread(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::DeleteFace(definition) => {
                self.evaluate_delete_face(backend, id, definition)
            }
            PartFeatureDefinition::ReplaceFace(definition) => {
                self.evaluate_replace_face(backend, id, definition)
            }
            PartFeatureDefinition::ThickenOffset(definition) => {
                self.evaluate_thicken_offset(backend, id, definition)
            }
            PartFeatureDefinition::DirectEdit(def) => self.evaluate_direct_edit(backend, id, def),
            PartFeatureDefinition::Split(definition) => {
                self.evaluate_split(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Mark(definition) => {
                self.evaluate_mark(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Finish(definition) => {
                self.evaluate_finish(backend, id, definition)
            }
            PartFeatureDefinition::ExtendSurface(definition) => {
                self.evaluate_extend_surface(backend, id, definition)
            }
            PartFeatureDefinition::BoundaryPatch(definition) => {
                self.evaluate_boundary_patch(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::RuledSurface(definition) => {
                self.evaluate_ruled_surface(backend, id, definition)
            }
            definition @ (PartFeatureDefinition::Mirror(_)
            | PartFeatureDefinition::CircularPattern(_)
            | PartFeatureDefinition::RectangularPattern(_)) => {
                self.evaluate_extrusion_pattern(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::Shell(definition) => {
                self.evaluate_shell(snapshot, backend, id, definition)
            }
            PartFeatureDefinition::WorkPlaneExtended(definition) => {
                self.evaluate_work_plane_extended(snapshot, backend, id, definition)
            }
            _ => {
                let bodies = &self.bodies;
                let lineage = &self.body_lineage;
                let resolve_face =
                    |reference: &crate::feature::work_plane::WorkPlaneFaceReference| {
                        resolve_body_face(bodies, lineage, backend, reference)
                    };
                let resolve_topology =
                    |reference: &crate::document_sketch::SketchFrameTopologyOrigin| {
                        super::body_resolution::resolve_body_frame_origin(
                            bodies, lineage, backend, reference,
                        )
                    };
                self.datums
                    .evaluate_entry_with_topology(
                        &snapshot.work_geometry,
                        id,
                        &snapshot.parameters,
                        &resolve_face,
                        &resolve_topology,
                    )
                    .map_err(|cause| PartFeatureFailure::Datum(Box::new(cause)))
            }
        }
    }
}
pub(super) use super::delete::require_replay_current;
#[path = "datum_health.rs"]
mod datum_health_projection;
use datum_health_projection::datum_health;
