// SPDX-License-Identifier: GPL-2.0-only
//! Bind ordered Part replay to the certified solid adapter.
use super::*;
use oblikovati_model::{
    feature::part::{
        PartBlendKind, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureProgramError,
    },
    history::DocumentSnapshot,
};

impl CertifiedNamedSolidEvaluator<'_> {
    /// Replay a private candidate, recomputing sketches that depend on Part datums.
    /// Example: `evaluator.evaluate_private_candidate(candidate)?`.
    pub fn evaluate_private_candidate(
        &mut self,
        candidate: &DocumentSnapshot,
    ) -> Result<PartFeatureEvaluation<NamedSolid, EvaluationFailure>, PartFeatureProgramError> {
        if candidate.sketches.requires_part_dependency_recompute() {
            candidate
                .recompute_part_with_sketches(self, Default::default())
                .map(|(part, _)| part)
        } else {
            candidate.evaluate_part_features(self)
        }
    }
}

impl PartFeatureEvaluator for CertifiedNamedSolidEvaluator<'_> {
    type Body = NamedSolid;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, body: &NamedSolid, key: &[u8]) -> Result<(), String> {
        match body {
            NamedSolid::Prism(body) => self.diagnostic.verify_body_reference(body, key),
            NamedSolid::Derived(body) => body
                .contains(self.history, self.document, key, 4)
                .map(|_| ()),
        }
    }
    fn resolve_planar_face(
        &self,
        body: &NamedSolid,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        match body {
            NamedSolid::Prism(body) => self.diagnostic.resolve_planar_face(body, key),
            NamedSolid::Derived(body) => body.face_plane(self.history, self.document, key),
        }
    }
    fn resolve_frame_origin(&self, body: &NamedSolid, key: &[u8]) -> Result<[f64; 3], String> {
        body.frame_origin(self.history, self.document, key)
    }
    fn evaluate_loft(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_loft::ResolvedSketchLoft,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "loft: expected an explicit model resolution policy".into(),
            ));
        };
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let candidate =
            crate::loft::build_loft(&input, prism.resolution_policy()).map_err(failure)?;
        let topology = BlendNamedTopology::from_loft(
            self.history,
            self.document,
            feature,
            input.sections.map(|section| section.source),
            input.center_line.map(|path| path.source),
            candidate,
        )
        .map_err(|cause| {
            failure(crate::EvaluationPhase::Naming.refused(
                crate::EvaluationFailureKind::InvalidReference,
                crate::EvaluationEntity::Body,
                format!("loft topology: {cause}"),
            ))
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
    fn evaluate_revolve(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "revolve: expected an explicit model resolution policy".into(),
            ));
        };
        let sketch_id = input.sketch.sketch.id().0;
        let policy = prism.resolution_policy();
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let topology = if input.surface {
            let candidate =
                crate::revolve::build_surface_revolve(input, policy).map_err(failure)?;
            BlendNamedTopology::from_surface_revolve(
                self.history,
                self.document,
                feature,
                sketch_id,
                candidate,
            )
        } else {
            let candidate = crate::revolve::build_revolve(input, policy).map_err(failure)?;
            BlendNamedTopology::from_revolve(
                self.history,
                self.document,
                feature,
                sketch_id,
                candidate,
            )
        }
        .map_err(|cause| {
            failure(crate::EvaluationPhase::Naming.refused(
                crate::EvaluationFailureKind::InvalidReference,
                crate::EvaluationEntity::Body,
                format!("revolve topology: {cause}"),
            ))
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
    fn evaluate_revolve_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
        operation: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        use super::super::blend::extrusion_boolean::BooleanFailure;
        let NamedSolid::Derived(tool) = self.evaluate_revolve(feature, input)? else {
            return Err(PartFeatureFailure::Unsupported(
                "revolve Boolean: expected a named Revolve tool".into(),
            ));
        };
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Prism
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let tool = tool
            .sweep_boolean_tool(self.history, self.document)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                operation,
                &format!("revolve {operation}"),
            )
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!("The feature did not change the number of faces ({faces})."),
                ),
                BooleanFailure::Refused(message) => {
                    failure(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
    fn evaluate_coil(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_coil::ResolvedSketchCoil<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "coil: expected an explicit model resolution policy".into(),
            ));
        };
        let sketch_id = input.sketch.sketch.id().0;
        let policy = prism.resolution_policy();
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let candidate = crate::coil::build_coil(input, policy).map_err(failure)?;
        let topology = BlendNamedTopology::from_coil(
            self.history,
            self.document,
            feature,
            sketch_id,
            candidate,
        )
        .map_err(|cause| {
            failure(crate::EvaluationPhase::Naming.refused(
                crate::EvaluationFailureKind::InvalidReference,
                crate::EvaluationEntity::Body,
                format!("coil topology: {cause}"),
            ))
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
    fn evaluate_coil_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_coil::ResolvedSketchCoil<'_>,
        operation: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        use super::super::blend::extrusion_boolean::BooleanFailure;
        let NamedSolid::Derived(tool) = self.evaluate_coil(feature, input)? else {
            return Err(PartFeatureFailure::Unsupported(
                "coil Boolean: expected a named Coil tool".into(),
            ));
        };
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Prism
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let tool = tool
            .sweep_boolean_tool(self.history, self.document)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                operation,
                &format!("coil {operation}"),
            )
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!("The feature did not change the number of faces ({faces})."),
                ),
                BooleanFailure::Refused(message) => {
                    failure(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
    fn evaluate_derived_part(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::derived_part::ResolvedDerivedPart,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_derived_part(feature, input)
    }
    fn evaluate_imported_model(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::imported_model::ResolvedImportedModel,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_imported_model(feature, input)
    }
    fn evaluate_unwrap(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::unwrap::ResolvedUnwrap<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_unwrap(feature, body, input)
    }
    fn evaluate_emboss(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_emboss::ResolvedSketchEmboss<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_emboss(feature, body, input)
    }
    fn evaluate_rib(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_rib::ResolvedSketchRib<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_rib(feature, body, input)
    }
    fn evaluate_decal(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_decal::ResolvedSketchDecal<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_decal(feature, body, input)
    }
    fn evaluate_sweep(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_sweep::ResolvedSketchSweep<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "sweep: expected an explicit model resolution policy".into(),
            ));
        };
        let sources = [input.profile.sketch.id().0, input.path.sketch.id().0];
        let policy = prism.resolution_policy();
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let topology = if input.surface {
            let candidate = crate::sweep::build_surface_sweep(input, policy).map_err(failure)?;
            BlendNamedTopology::from_surface_sweep(
                self.history,
                self.document,
                feature,
                sources,
                candidate,
            )
        } else {
            let candidate = crate::sweep::build_sweep(input, policy).map_err(failure)?;
            BlendNamedTopology::from_sweep(self.history, self.document, feature, sources, candidate)
        }
        .map_err(|cause| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Naming
                    .refused(
                        crate::EvaluationFailureKind::InvalidReference,
                        crate::EvaluationEntity::Body,
                        format!("sweep topology: {cause}"),
                    )
                    .for_feature(feature),
            )
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
    fn evaluate_sweep_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_sweep::ResolvedSketchSweep<'_>,
        operation: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let NamedSolid::Derived(tool) = self.evaluate_sweep(feature, input)? else {
            return Err(PartFeatureFailure::Unsupported(
                "sweep Boolean: expected a named Sweep tool".into(),
            ));
        };
        self.evaluate_named_sweep_boolean(feature, body, tool, operation)
    }
    fn evaluate_prism_outputs(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Vec<NamedSolid>, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "extrusion outputs: expected an explicit model resolution policy".into(),
            ));
        };
        prism
            .evaluate_prism_outputs(feature, input)
            .map(|bodies| {
                bodies
                    .into_iter()
                    .map(|body| NamedSolid::Prism(Box::new(body)))
                    .collect()
            })
            .map_err(PartFeatureFailure::Kernel)
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedSolid, EvaluationFailure> {
        let body = match &mut self.prism {
            Some(prism) => prism.evaluate_prism(feature, input),
            None => self.diagnostic.evaluate_prism(feature, input),
        }?;
        Ok(NamedSolid::Prism(Box::new(body)))
    }
    fn prism_through_all_span(
        &self,
        body: &NamedSolid,
        plane: oblikovati_model::sketch::SketchPlane,
        direction: oblikovati_model::feature::sketch_prism::ThroughAllDirection,
    ) -> Result<(f64, f64), String> {
        self.named_prism_through_all_span(body, plane, direction)
    }
    fn evaluate_prism_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchPrism<'_>,
        kind: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_prism_boolean(feature, body, input, kind)
    }
    fn evaluate_extrusion_pattern(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        sources: &[oblikovati_model::feature::extrusion_pattern::PatternPrismSource<'_>],
        occurrences: &[oblikovati_model::feature::extrusion_pattern::ExtrusionPatternOccurrence],
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_extrusion_pattern(feature, body, sources, occurrences)
    }
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_blend(kind, feature, body, edges, distance)
    }
    fn evaluate_fillet_v2(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        input: oblikovati_model::feature::fillet_options::ResolvedFillet,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_fillet_v2(feature, body, edges, input)
    }
    fn evaluate_chamfer_v2(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        input: oblikovati_model::feature::chamfer_options::ResolvedChamfer,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_chamfer_v2(feature, body, edges, input)
    }
    fn evaluate_move_body(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        motions: &[oblikovati_model::feature::move_body::ResolvedBodyMotion],
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_move_body(feature, body, motions)
    }
    fn evaluate_hole(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        face: &[u8],
        input: oblikovati_model::feature::hole::ResolvedHole,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_hole(feature, body, face, input)
    }
    fn evaluate_slot(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        face: &[u8],
        input: oblikovati_model::feature::slot::ResolvedSlot,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_slot(feature, body, face, input)
    }
    fn evaluate_combine(
        &mut self,
        feature: FeatureId,
        base: &NamedSolid,
        tools: &[&NamedSolid],
        input: oblikovati_model::feature::combine::ResolvedCombine,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_combine(feature, base, tools, input)
    }
    fn evaluate_thicken_offset(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::thicken_offset::ResolvedThickenOffset,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_thicken_offset(feature, body, input)
    }
    fn evaluate_shell(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        openings: &[Vec<u8>],
        input: oblikovati_model::feature::shell::ResolvedShell,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_shell(feature, body, openings, input)
    }
}
