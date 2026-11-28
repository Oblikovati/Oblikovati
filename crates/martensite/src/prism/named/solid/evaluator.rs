// SPDX-License-Identifier: GPL-2.0-only
//! Bind ordered Part replay to the certified solid adapter.
use super::*;
use crate::{EvaluationEntity, EvaluationFailureKind, EvaluationPhase};
use oblikovati_model::{
    feature::{
        part::{
            PartBlendKind, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureProgramError,
        },
        sketch_prism::PrismBoolean,
        work_axis::WorkAxisLine,
    },
    history::DocumentSnapshot,
    sketch::SketchPlane,
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
    fn resolve_planar_face(&self, body: &NamedSolid, key: &[u8]) -> Result<SketchPlane, String> {
        match body {
            NamedSolid::Prism(body) => self.diagnostic.resolve_planar_face(body, key),
            NamedSolid::Derived(body) => body.face_plane(self.history, self.document, key),
        }
    }
    fn resolve_frame_origin(&self, body: &NamedSolid, key: &[u8]) -> Result<[f64; 3], String> {
        body.frame_origin(self.history, self.document, key)
    }
    fn resolve_topology_point(&self, body: &NamedSolid, key: &[u8]) -> Result<[f64; 3], String> {
        body.resolve_topology_point(self.history, self.document, key)
    }
    fn resolve_straight_edge(&self, body: &NamedSolid, key: &[u8]) -> Result<WorkAxisLine, String> {
        body.resolve_straight_edge(self.history, self.document, key)
    }
    fn resolve_torus_midplane(&self, body: &NamedSolid, key: &[u8]) -> Result<SketchPlane, String> {
        body.resolve_torus_midplane(self.history, self.document, key)
    }
    fn resolve_tangent_plane_through_point(
        &self,
        body: &NamedSolid,
        k: &[u8],
        p: [f64; 3],
    ) -> Result<SketchPlane, String> {
        body.resolve_tangent_plane_through_point(self.history, self.document, k, p)
    }
    fn resolve_tangent_plane_through_line(
        &self,
        body: &NamedSolid,
        k: &[u8],
        l: WorkAxisLine,
    ) -> Result<SketchPlane, String> {
        body.resolve_tangent_plane_through_line(self.history, self.document, k, l)
    }
    fn resolve_tangent_plane_parallel(
        &self,
        body: &NamedSolid,
        k: &[u8],
        p: SketchPlane,
    ) -> Result<SketchPlane, String> {
        body.resolve_tangent_plane_parallel(self.history, self.document, k, p)
    }
    fn evaluate_loft(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_loft::ResolvedSketchLoft,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_loft(feature, input)
    }
    fn evaluate_revolve(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_revolve(feature, input)
    }
    fn evaluate_revolve_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
        operation: PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let NamedSolid::Derived(tool) = self.evaluate_revolve(feature, input)? else {
            return Err(PartFeatureFailure::Unsupported(
                "revolve Boolean: expected a named Revolve tool".into(),
            ));
        };
        self.apply_derived_tool_boolean(feature, body, tool, operation, "revolve")
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
            failure(EvaluationPhase::Naming.refused(
                EvaluationFailureKind::InvalidReference,
                EvaluationEntity::Body,
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
        operation: PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let NamedSolid::Derived(tool) = self.evaluate_coil(feature, input)? else {
            return Err(PartFeatureFailure::Unsupported(
                "coil Boolean: expected a named Coil tool".into(),
            ));
        };
        self.apply_derived_tool_boolean(feature, body, tool, operation, "coil")
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
    fn evaluate_draft(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::draft::ResolvedDraft,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_draft(feature, body, input)
    }
    fn evaluate_thread(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::thread::ResolvedThread,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_thread(feature, body, input)
    }
    fn evaluate_delete_face(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::delete_face::ResolvedDeleteFace,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_delete_face(feature, body, input)
    }
    fn evaluate_thicken_offset(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::thicken_offset::ResolvedThickenOffset,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_thicken_offset(feature, body, input)
    }
    fn evaluate_direct_edit(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::direct_edit::ResolvedDirectEdit,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_direct_edit(feature, body, input)
    }
    fn evaluate_split(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::split::ResolvedSplit,
    ) -> Result<Vec<NamedSolid>, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_split(feature, body, input)
    }
    fn evaluate_mark(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::mark::ResolvedMark,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_mark(feature, body, input)
    }
    fn evaluate_finish(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::finish::ResolvedFinish,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_finish(feature, body, input)
    }
    fn evaluate_extend_surface(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: oblikovati_model::feature::extend_surface::ResolvedExtendSurface,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        self.evaluate_named_extend_surface(feature, body, input)
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

impl CertifiedNamedSolidEvaluator<'_> {
    fn apply_derived_tool_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        tool: Box<BlendNamedTopology>,
        operation: PrismBoolean,
        label: &str,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        use super::super::blend::extrusion_boolean::BooleanFailure;
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(kind, EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let tool = tool
            .sweep_boolean_tool(self.history, self.document)
            .map_err(|message| failure(EvaluationFailureKind::InvalidResult, message))?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                operation,
                &format!("{label} {operation}"),
            )
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    EvaluationFailureKind::UnchangedCut,
                    format!("The feature did not change the number of faces ({faces})."),
                ),
                BooleanFailure::Refused(message) => {
                    failure(EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
