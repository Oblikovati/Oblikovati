// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Shell during ordered part replay (#4106).
use super::*;
use oblikovati_model::feature::shell::ResolvedShell;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_shell(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        openings: &[Vec<u8>],
        input: ResolvedShell,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.shell_v2_candidate(self.history, self.document, openings, &input)
            }
            NamedSolid::Derived(body) => {
                body.shell_v2_candidate(self.history, self.document, openings, &input)
            }
        }
        .map_err(|message| shell_failure(feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| shell_failure(feature, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn shell_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Shell
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
