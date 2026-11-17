// SPDX-License-Identifier: GPL-2.0-only
//! Named blend execution and lineage checks (#4045, #4046).
use super::*;

impl FilletEvaluator<NamedSolid, EvaluationFailure> for CertifiedNamedSolidEvaluator<'_> {
    fn evaluate_fillet(
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
            NamedSolid::Fillet(body) | NamedSolid::Chamfer(body) => {
                body.fillet_candidate(self.history, self.document, edges, radius)
            }
        }
        .map_err(|message| failure(Phase::Fillet, feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(Phase::Naming, feature, message))?;
        Ok(NamedSolid::Fillet(Box::new(named)))
    }

    fn verify_fillet_lineage(&self, before: &NamedSolid, after: &NamedSolid) -> Result<(), String> {
        self.verify_blend_lineage(before, after)
    }
}

impl ChamferEvaluator<NamedSolid, EvaluationFailure> for CertifiedNamedSolidEvaluator<'_> {
    fn evaluate_chamfer(
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
            NamedSolid::Fillet(body) | NamedSolid::Chamfer(body) => {
                body.chamfer_candidate(self.history, self.document, edges, distance)
            }
        }
        .map_err(|message| failure(Phase::Chamfer, feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(Phase::Naming, feature, message))?;
        Ok(NamedSolid::Chamfer(Box::new(named)))
    }
    fn verify_chamfer_lineage(
        &self,
        before: &NamedSolid,
        after: &NamedSolid,
    ) -> Result<(), String> {
        self.verify_blend_lineage(before, after)
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    fn verify_blend_lineage(&self, before: &NamedSolid, after: &NamedSolid) -> Result<(), String> {
        let (prior_live, prior_deleted) = before.names(self.history, self.document)?;
        let (live, deleted) = after.names(self.history, self.document)?;
        let accounted: BTreeSet<_> = live.iter().chain(&deleted).collect();
        for key in &prior_live {
            if !accounted.contains(key) {
                return Err(format!(
                    "blend input name {key:?}: expected a live or deleted result record"
                ));
            }
        }
        for key in &prior_deleted {
            if !deleted.contains(key) {
                return Err(format!(
                    "blend deleted name {key:?}: expected a retained deletion record"
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
