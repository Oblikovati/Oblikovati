// SPDX-License-Identifier: GPL-2.0-only
//! Replay a candidate tail while borrowing an unchanged complete prefix.
use super::{PartBlendKind, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayRange,
        sketch_prism::{PrismBoolean, ResolvedSketchPrism},
    },
    history::DocumentSnapshot,
};
use std::borrow::Cow;

enum PrefixBody<'a, Body> {
    Retained(&'a Body),
    Created(Body),
}
impl<Body> PrefixBody<'_, Body> {
    fn get(&self) -> &Body {
        match self {
            Self::Retained(body) => body,
            Self::Created(body) => body,
        }
    }
}

struct PrefixBackend<'backend, 'body, Backend>(
    &'backend mut Backend,
    std::marker::PhantomData<&'body ()>,
);
impl<'a, Backend: PartFeatureEvaluator> PartFeatureEvaluator for PrefixBackend<'_, 'a, Backend>
where
    Backend::Body: 'a,
{
    type Body = PrefixBody<'a, Backend::Body>;
    type Failure = Backend::Failure;
    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String> {
        self.0.verify_body_reference(body.get(), key)
    }
    fn resolve_planar_face(
        &self,
        body: &Self::Body,
        key: &[u8],
    ) -> Result<crate::sketch::SketchPlane, String> {
        self.0.resolve_planar_face(body.get(), key)
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_blend(kind, feature, body.get(), edges, distance)
            .map(PrefixBody::Created)
    }
    fn evaluate_fillet_v2(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        input: crate::feature::fillet_options::ResolvedFillet,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_fillet_v2(feature, body.get(), edges, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_chamfer_v2(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        edges: &[Vec<u8>],
        input: crate::feature::chamfer_options::ResolvedChamfer,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_chamfer_v2(feature, body.get(), edges, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_move_body(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        motions: &[crate::feature::move_body::ResolvedBodyMotion],
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_move_body(feature, body.get(), motions)
            .map(PrefixBody::Created)
    }
    fn evaluate_hole(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        face: &[u8],
        input: crate::feature::hole::ResolvedHole,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_hole(feature, body.get(), face, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_slot(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        face: &[u8],
        input: crate::feature::slot::ResolvedSlot,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_slot(feature, body.get(), face, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_draft(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::draft::ResolvedDraft,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_draft(feature, body.get(), input)
            .map(PrefixBody::Created)
    }
    fn evaluate_shell(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        openings: &[Vec<u8>],
        input: crate::feature::shell::ResolvedShell,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_shell(feature, body.get(), openings, input)
            .map(PrefixBody::Created)
    }
    fn prism_through_all_span(
        &self,
        body: &Self::Body,
        plane: crate::sketch::SketchPlane,
        direction: crate::feature::sketch_prism::ThroughAllDirection,
    ) -> Result<(f64, f64), String> {
        self.0.prism_through_all_span(body.get(), plane, direction)
    }
    fn evaluate_prism_boolean(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: ResolvedSketchPrism<'_>,
        operation: crate::feature::sketch_prism::PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_prism_boolean(feature, body.get(), input, operation)
            .map(PrefixBody::Created)
    }
    fn evaluate_extrusion_pattern(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        sources: &[crate::feature::extrusion_pattern::PatternPrismSource<'_>],
        occurrences: &[crate::feature::extrusion_pattern::ExtrusionPatternOccurrence],
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_extrusion_pattern(feature, body.get(), sources, occurrences)
            .map(PrefixBody::Created)
    }
    fn evaluate_loft(
        &mut self,
        feature: FeatureId,
        input: crate::feature::sketch_loft::ResolvedSketchLoft,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_loft(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_revolve(
        &mut self,
        feature: FeatureId,
        input: crate::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_revolve(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_revolve_boolean(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
        operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_revolve_boolean(feature, body.get(), input, operation)
            .map(PrefixBody::Created)
    }
    fn evaluate_coil(
        &mut self,
        feature: FeatureId,
        input: crate::feature::sketch_coil::ResolvedSketchCoil<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_coil(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_coil_boolean(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_coil::ResolvedSketchCoil<'_>,
        operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_coil_boolean(feature, body.get(), input, operation)
            .map(PrefixBody::Created)
    }
    fn evaluate_derived_part(
        &mut self,
        feature: FeatureId,
        input: crate::feature::derived_part::ResolvedDerivedPart,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_derived_part(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_imported_model(
        &mut self,
        feature: FeatureId,
        input: crate::feature::imported_model::ResolvedImportedModel,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_imported_model(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_unwrap(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::unwrap::ResolvedUnwrap<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_unwrap(feature, body.get(), input)
            .map(PrefixBody::Created)
    }
    fn evaluate_emboss(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_emboss::ResolvedSketchEmboss<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_emboss(feature, body.get(), input)
            .map(PrefixBody::Created)
    }
    fn evaluate_rib(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_rib::ResolvedSketchRib<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_rib(feature, body.get(), input)
            .map(PrefixBody::Created)
    }
    fn evaluate_decal(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_decal::ResolvedSketchDecal<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_decal(feature, body.get(), input)
            .map(PrefixBody::Created)
    }
    fn evaluate_sweep(
        &mut self,
        feature: FeatureId,
        input: crate::feature::sketch_sweep::ResolvedSketchSweep<'_>,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_sweep(feature, input)
            .map(PrefixBody::Created)
    }
    fn evaluate_sweep_boolean(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::sketch_sweep::ResolvedSketchSweep<'_>,
        operation: PrismBoolean,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_sweep_boolean(feature, body.get(), input, operation)
            .map(PrefixBody::Created)
    }
    fn evaluate_prism_outputs(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Vec<Self::Body>, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_prism_outputs(feature, input)
            .map(|bodies| bodies.into_iter().map(PrefixBody::Created).collect())
    }

    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        self.0
            .evaluate_prism(feature, input)
            .map(PrefixBody::Created)
    }
}

/// A private candidate result that borrows its unchanged source bodies.
pub struct BorrowedPartReplay<'a, Body, Failure>(
    PartFeatureEvaluation<PrefixBody<'a, Body>, Failure>,
);
impl<Body, Failure> BorrowedPartReplay<'_, Body, Failure> {
    /// Inspect ordered candidate health; example: `candidate.health()` after preview.
    pub fn health(&self) -> &[FeatureHealthRecord<PartFeatureFailure<Failure>>] {
        self.0.health()
    }
    /// Count live candidate bodies; example: `candidate.body_count()` for a command reply.
    pub fn body_count(&self) -> usize {
        self.0.ordered_bodies().count()
    }
    /// Check a current candidate output; example: `candidate.has_body(feature)`.
    pub fn has_body(&self, feature: FeatureId) -> bool {
        self.0.body(feature).is_some()
    }
    /// Borrow a candidate body for exact validation; example: `candidate.body(feature)`.
    pub fn body(&self, feature: FeatureId) -> Option<&Body> {
        self.0.body(feature).map(PrefixBody::get)
    }
    /// Resolve a producer through later body replacements; example: `candidate.body_from_source(feature)?`.
    pub fn body_from_source(&self, feature: FeatureId) -> Result<&Body, String> {
        self.0.body_from_source(feature).map(PrefixBody::get)
    }
}

impl<Body, Failure: Clone> BorrowedPartReplay<'_, Body, Failure> {
    /// Capture every live body through the backend without changing the candidate.
    /// Example: `candidate.try_map_bodies(|id, body| copy(id, body))?`.
    pub fn try_map_bodies<Mapped, E>(
        &self,
        mut capture: impl FnMut(FeatureId, &Body) -> Result<Mapped, E>,
    ) -> Result<PartFeatureEvaluation<Mapped, Failure>, E> {
        self.0.try_map_bodies(|id, body| capture(id, body.get()))
    }
}

impl<Body, Failure: Clone> PartFeatureEvaluation<Body, Failure> {
    /// Capture each body while retaining the exact replay metadata.
    /// Example: `result.try_map_bodies(|feature, body| capture(feature, body))?`.
    pub fn try_map_bodies<Mapped, E>(
        &self,
        mut capture: impl FnMut(FeatureId, &Body) -> Result<Mapped, E>,
    ) -> Result<PartFeatureEvaluation<Mapped, Failure>, E> {
        let bodies = self
            .bodies
            .iter()
            .map(|(id, body)| capture(id.feature, body).map(|body| (*id, body)))
            .collect::<Result<_, _>>()?;
        Ok(PartFeatureEvaluation {
            datums: self.datums.clone(),
            bodies,
            body_lineage: self.body_lineage.clone(),
            deleted: self.deleted.clone(),
            health: self.health.clone(),
            source: self.source.clone(),
        })
    }
}

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    /// Convert captured bodies while retaining the evaluation recipe and lineage.
    /// Example: `evaluation.try_map_owned_bodies(|id, body| restore(id, body))?`.
    pub fn try_map_owned_bodies<Mapped, E>(
        self,
        mut map: impl FnMut(FeatureId, Body) -> Result<Mapped, E>,
    ) -> Result<PartFeatureEvaluation<Mapped, Failure>, E> {
        let bodies = self
            .bodies
            .into_iter()
            .map(|(id, body)| map(id.feature, body).map(|body| (id, body)))
            .collect::<Result<_, _>>()?;
        Ok(PartFeatureEvaluation {
            datums: self.datums,
            bodies,
            body_lineage: self.body_lineage,
            deleted: self.deleted,
            health: self.health,
            source: self.source,
        })
    }
    /// Borrow the evaluated recipe for independent presentation replay; example: `evaluation.replay_snapshot()`.
    pub fn replay_snapshot(&self) -> Option<&std::sync::Arc<DocumentSnapshot>> {
        self.source.as_ref()
    }
    /// Find a possible dirty tail since this result's source recipe.
    /// Example: `evaluation.replay_change(&candidate)` before planning a replay.
    pub fn replay_change(&self, candidate: &DocumentSnapshot) -> Option<FeatureReplayRange> {
        self.source
            .as_deref()
            .and_then(|source| source.part_replay_change(candidate))
    }
}

impl<Body, Failure: Clone> PartFeatureEvaluation<Body, Failure> {
    /// Replay a candidate after an unchanged complete prefix, without publishing it.
    /// Example: `current.replay_candidate_tail(&candidate, &mut backend)?`.
    /// The caller must keep the original result's document and policy current.
    pub fn replay_candidate_tail<'a, B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &'a self,
        candidate: &DocumentSnapshot,
        backend: &mut B,
    ) -> Result<BorrowedPartReplay<'a, Body, Failure>, String> {
        let source = self
            .source
            .as_deref()
            .ok_or("part prefix: expected its source recipe")?;
        let expected = self.health.len();
        let range = candidate
            .part_replay_range()
            .map_err(|error| error.to_string())?;
        let changed = source
            .part_replay_change(candidate)
            .ok_or("part prefix: expected a changed tail")?;
        if changed.start != expected || expected > range.end {
            return Err(format!(
                "part prefix {expected}, change {changed:?}: expected an unchanged complete prefix"
            ));
        }
        self.replay_tail(
            candidate,
            backend,
            FeatureReplayRange {
                start: expected,
                end: range.end,
            },
        )
    }
    fn replay_tail<'a, B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &'a self,
        candidate: &DocumentSnapshot,
        backend: &mut B,
        range: FeatureReplayRange,
    ) -> Result<BorrowedPartReplay<'a, Body, Failure>, String> {
        let sequence =
            super::body_dependencies::with_replacement_reads(candidate.effective_work_geometry())
                .map_err(|error| error.to_string())?;
        let mut controller = FeatureReplayController::new_with_parameters(
            &sequence,
            range,
            self.health.clone(),
            &candidate.parameters,
        )
        .map_err(|error| error.to_string())?;
        let mut result = self.borrow_prefix();
        let mut candidate = Cow::Borrowed(candidate);
        let mut backend = PrefixBackend(backend, std::marker::PhantomData);
        result
            .replay(
                &mut candidate,
                &mut backend,
                &mut controller,
                None,
                &mut || false,
            )
            .map_err(|error| error.to_string())?;
        result.refresh_sketch_references(&candidate, &backend);
        result.health = controller.finish().map_err(|error| error.to_string())?;
        result.source = Some(std::sync::Arc::new(candidate.as_ref().clone()));
        Ok(BorrowedPartReplay(result))
    }
    fn borrow_prefix(&self) -> PartFeatureEvaluation<PrefixBody<'_, Body>, Failure> {
        PartFeatureEvaluation {
            datums: self.datums.clone(),
            bodies: self
                .bodies
                .iter()
                .map(|(id, body)| (*id, PrefixBody::Retained(body)))
                .collect(),
            body_lineage: self.body_lineage.clone(),
            deleted: self.deleted.clone(),
            health: self.health.clone(),
            source: None,
        }
    }
}
