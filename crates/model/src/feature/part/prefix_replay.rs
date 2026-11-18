// SPDX-License-Identifier: GPL-2.0-only
//! Replay a candidate tail while borrowing an unchanged complete prefix.
use super::{PartBlendKind, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure};
use crate::{
    feature::{
        FeatureHealthRecord, FeatureId, FeatureReplayController, FeatureReplayRange,
        sketch_prism::ResolvedSketchPrism,
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
}

impl<Body, Failure: Clone> BorrowedPartReplay<'_, Body, Failure> {
    /// Capture every live body through the backend without changing the candidate.
    /// Example: `candidate.try_map_bodies(|id, body| copy(id, body))?`.
    pub fn try_map_bodies<Mapped, E>(
        &self,
        mut capture: impl FnMut(FeatureId, &Body) -> Result<Mapped, E>,
    ) -> Result<PartFeatureEvaluation<Mapped, Failure>, E> {
        let bodies = self
            .0
            .bodies
            .iter()
            .map(|(id, body)| capture(*id, body.get()).map(|body| (*id, body)))
            .collect::<Result<_, _>>()?;
        Ok(PartFeatureEvaluation {
            datums: self.0.datums.clone(),
            bodies,
            body_lineage: self.0.body_lineage.clone(),
            deleted: self.0.deleted.clone(),
            health: self.0.health.clone(),
            source: self.0.source.clone(),
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
            .map(|(id, body)| map(id, body).map(|body| (id, body)))
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
        result.refresh_sketch_faces(&candidate, &backend);
        result.health = controller.finish().map_err(|error| error.to_string())?;
        result.source = Some(Box::new(candidate.as_ref().clone()));
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
