// SPDX-License-Identifier: GPL-2.0-only
//! Bind ordered Part replay to the certified solid adapter.
use super::*;
use oblikovati_model::feature::part::{PartBlendKind, PartFeatureEvaluator};

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
    fn evaluate_blend(
        &mut self,
        kind: PartBlendKind,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let result = match kind {
            PartBlendKind::Fillet => self.evaluate_fillet(feature, body, edges, distance),
            PartBlendKind::Chamfer => self.evaluate_chamfer(feature, body, edges, distance),
        }
        .map_err(PartFeatureFailure::Kernel)?;
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
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
