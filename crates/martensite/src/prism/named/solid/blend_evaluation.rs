// SPDX-License-Identifier: GPL-2.0-only
//! Named blend execution and lineage checks (#4045, #4046).
use super::*;
use crate::prism::named::blend::extrusion_boolean::BooleanFailure;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let result = match kind {
            oblikovati_model::feature::part::PartBlendKind::Fillet => {
                self.evaluate_fillet(feature, body, edges, distance)
            }
            oblikovati_model::feature::part::PartBlendKind::Chamfer => {
                self.evaluate_chamfer(feature, body, edges, distance)
            }
        }
        .map_err(PartFeatureFailure::Kernel)?;
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }

    pub(super) fn evaluate_named_fillet_v2(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        input: oblikovati_model::feature::fillet_options::ResolvedFillet,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.fillet_v2_candidate(self.history, self.document, edges, &input)
            }
            NamedSolid::Derived(body) => {
                body.fillet_v2_candidate(self.history, self.document, edges, &input)
            }
        }
        .map_err(|message| PartFeatureFailure::Kernel(failure(Phase::Fillet, feature, message)))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| {
                PartFeatureFailure::Kernel(failure(Phase::Naming, feature, message))
            })?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }

    pub(super) fn evaluate_named_chamfer_v2(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        input: oblikovati_model::feature::chamfer_options::ResolvedChamfer,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.chamfer_v2_candidate(self.history, self.document, feature, edges, &input)
            }
            NamedSolid::Derived(body) => {
                body.chamfer_v2_candidate(self.history, self.document, feature, edges, &input)
            }
        }
        .map_err(|message| PartFeatureFailure::Kernel(failure(Phase::Chamfer, feature, message)))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| {
                PartFeatureFailure::Kernel(failure(Phase::Naming, feature, message))
            })?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }

    pub(super) fn evaluate_named_sweep_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        tool: Box<BlendNamedTopology>,
        operation: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let fail = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Sweep
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let tool = tool
            .sweep_boolean_tool(self.history, self.document)
            .map_err(|message| fail(crate::EvaluationFailureKind::InvalidResult, message))?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                operation,
                &format!("sweep {operation}"),
            )
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => fail(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!("The feature did not change the number of faces ({faces})."),
                ),
                BooleanFailure::Refused(message) => {
                    fail(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| fail(crate::EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }

    pub(super) fn evaluate_fillet(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<NamedSolid, EvaluationFailure> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.fillet_candidate(self.history, self.document, edges, radius)
            }
            NamedSolid::Derived(body) => {
                body.fillet_candidate(self.history, self.document, edges, radius)
            }
        }
        .map_err(|message| failure(Phase::Fillet, feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(Phase::Naming, feature, message))?;
        Ok(NamedSolid::Derived(Box::new(named)))
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_chamfer(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<NamedSolid, EvaluationFailure> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.chamfer_candidate(self.history, self.document, edges, distance)
            }
            NamedSolid::Derived(body) => {
                body.chamfer_candidate(self.history, self.document, edges, distance)
            }
        }
        .map_err(|message| failure(Phase::Chamfer, feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(Phase::Naming, feature, message))?;
        Ok(NamedSolid::Derived(Box::new(named)))
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(crate) fn verify_result_lineage(
        &self,
        before: &NamedSolid,
        after: &NamedSolid,
    ) -> Result<(), String> {
        let (prior_live, prior_deleted) = before.names(self.history, self.document)?;
        let (live, deleted) = after.names(self.history, self.document)?;
        let prior_splits = match before {
            NamedSolid::Derived(body) => Some(body.split_keys()),
            NamedSolid::Prism(_) => None,
        };
        let splits = match after {
            NamedSolid::Derived(body) => Some(body.split_keys()),
            NamedSolid::Prism(_) => None,
        };
        let accounted: BTreeSet<_> = live
            .iter()
            .chain(&deleted)
            .chain(splits.into_iter().flat_map(|names| names.keys()))
            .collect();
        for key in &prior_live {
            if !accounted.contains(key) {
                return Err(format!(
                    "input name {key:?}: expected a live, split, or deleted result record"
                ));
            }
        }
        for key in &prior_deleted {
            if !deleted.contains(key) {
                return Err(format!(
                    "deleted name {key:?}: expected a retained deletion record"
                ));
            }
        }
        for (parent, children) in prior_splits.into_iter().flatten() {
            if splits.and_then(|names| names.get(parent)) != Some(children) {
                return Err(format!(
                    "split name {parent:?}: expected retained descendant names"
                ));
            }
        }
        Ok(())
    }
}

fn failure(phase: Phase, feature: FeatureId, message: String) -> EvaluationFailure {
    phase
        .refused(Kind::InvalidResult, Entity::Body, message)
        .for_feature(feature)
}
