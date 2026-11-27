// SPDX-License-Identifier: GPL-2.0-only
//! Resolve sketches and construct solid features in program order.
use super::{
    PartFeatureDefinition, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    PartFeatureProgramError,
    body_resolution::{resolve_body_face, resolve_body_slot},
    evaluate::require_replay_current,
};
use crate::{
    feature::{
        AuthoredFeature, FeatureId, sketch_loft::SketchLoftDefinition,
        sketch_prism::SketchPrismDefinition, sketch_sweep::SketchSweepDefinition,
    },
    history::DocumentSnapshot,
};
use std::borrow::Cow;

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
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
        let sketches = match feature.definition() {
            PartFeatureDefinition::SketchPrism(definition) => {
                [Some(definition.sketch()), None, None]
            }
            PartFeatureDefinition::SketchLoft(definition) => {
                let recipe = definition.recipe();
                let [first, last] = recipe.sections;
                [
                    Some(first.sketch),
                    Some(last.sketch),
                    recipe.center_line.map(|path| path.sketch),
                ]
            }
            PartFeatureDefinition::SketchSweep(definition) => {
                let recipe = definition.recipe();
                [
                    Some(recipe.profile_sketch),
                    (recipe.profile_sketch != recipe.path_sketch).then_some(recipe.path_sketch),
                    None,
                ]
            }
            _ => return Ok(()),
        };
        let candidate = snapshot.to_mut();
        for sketch in sketches.into_iter().flatten() {
            if let Ok(solved) = candidate.solve_one_document_sketch_with_datums(
                sketch,
                options,
                &mut *cancelled,
                &self.datums,
            ) {
                require_replay_current(solved.report.cancelled || cancelled(), completed)?;
                candidate.sketches = solved.candidate;
            }
        }
        Ok(())
    }
    pub(super) fn evaluate_prism<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: &AuthoredFeature<PartFeatureDefinition>,
        definition: &SketchPrismDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        if let Some(target) = definition.boolean_target() {
            return self.replace_with_prism(snapshot, backend, feature.id(), target, definition);
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
        self.bodies.insert(feature.id().into(), body);
        Ok(())
    }

    pub(super) fn evaluate_loft<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &SketchLoftDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let sections = definition
            .resolve(snapshot, &self.datums)
            .map_err(PartFeatureFailure::LoftInput)?;
        let body = backend.evaluate_loft(feature, sections)?;
        self.body_lineage
            .register(feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.into(), body);
        Ok(())
    }

    pub(super) fn evaluate_sweep<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &SketchSweepDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve_for_replay(snapshot, &self.datums)
            .map_err(|cause| PartFeatureFailure::SweepInput(Box::new(cause)))?;
        if let Some((operation, key, source)) = definition.boolean_target() {
            let slot = resolve_body_slot(&self.bodies, &self.body_lineage, backend, source, key)
                .map_err(PartFeatureFailure::BodyReference)?;
            let result =
                backend.evaluate_sweep_boolean(feature, &self.bodies[&slot], input, operation)?;
            return self.accept_body_replacement(backend, feature, slot, key, result);
        }
        let body = backend.evaluate_sweep(feature, input)?;
        self.body_lineage
            .register(feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.into(), body);
        Ok(())
    }
}
