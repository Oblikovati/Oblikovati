// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Modify Split features during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::split::{ResolvedSplit, SplitKeepSide, SplitMode};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_split(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSplit,
    ) -> Result<Vec<NamedSolid>, PartFeatureFailure<EvaluationFailure>> {
        let candidates = match body {
            NamedSolid::Prism(body) => {
                body.split_candidates(self.history, self.document, feature, &input)
            }
            NamedSolid::Derived(body) => {
                body.split_candidates(self.history, self.document, feature, &input)
            }
        }
        .map_err(|message| split_failure(feature, message))?;
        let multi_body = input.mode == SplitMode::Solid && input.keep_side == SplitKeepSide::Both;
        let mut results = Vec::with_capacity(candidates.len());
        for (index, candidate) in candidates.into_iter().enumerate() {
            let named = if multi_body {
                let tag = if index == 0 { "split-a" } else { "split-b" };
                candidate
                    .name_split_body_topology(self.history, self.document, feature, tag)
                    .map_err(|message| split_failure(feature, message))?
            } else {
                candidate
                    .name_topology(self.history, self.document, feature)
                    .map_err(|message| split_failure(feature, message))?
            };
            let solid = NamedSolid::Derived(Box::new(named));
            if !multi_body {
                self.verify_result_lineage(body, &solid)
                    .map_err(PartFeatureFailure::BodyReference)?;
            }
            results.push(solid);
        }
        Ok(results)
    }
}

fn split_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Prism
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
