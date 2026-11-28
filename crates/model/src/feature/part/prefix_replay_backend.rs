// SPDX-License-Identifier: GPL-2.0-only
//! Prefix-borrowing evaluator wrapper for candidate tail replays.
use super::{PartBlendKind, PartFeatureEvaluator, PartFeatureFailure};
use crate::feature::{
    FeatureId,
    sketch_prism::{PrismBoolean, ResolvedSketchPrism},
};

pub(super) enum PrefixBody<'a, Body> {
    Retained(&'a Body),
    Created(Body),
}

impl<Body> PrefixBody<'_, Body> {
    pub(super) fn get(&self) -> &Body {
        match self {
            Self::Retained(body) => body,
            Self::Created(body) => body,
        }
    }
}

pub(super) struct PrefixBackend<'backend, 'body, Backend>(
    pub(super) &'backend mut Backend,
    pub(super) std::marker::PhantomData<&'body ()>,
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

    fn evaluate_combine(
        &mut self,
        feature: FeatureId,
        base: &Self::Body,
        tools: &[&Self::Body],
        input: crate::feature::combine::ResolvedCombine,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        let inner: Vec<_> = tools.iter().map(|tool| tool.get()).collect();
        self.0
            .evaluate_combine(feature, base.get(), &inner, input)
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

    fn evaluate_delete_face(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::delete_face::ResolvedDeleteFace,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_delete_face(feature, body.get(), input)
            .map(PrefixBody::Created)
    }

    fn evaluate_finish(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::finish::ResolvedFinish,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_finish(feature, body.get(), input)
            .map(PrefixBody::Created)
    }

    fn evaluate_extend_surface(
        &mut self,
        feature: FeatureId,
        body: &Self::Body,
        input: crate::feature::extend_surface::ResolvedExtendSurface,
    ) -> Result<Self::Body, PartFeatureFailure<Self::Failure>> {
        self.0
            .evaluate_extend_surface(feature, body.get(), input)
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
