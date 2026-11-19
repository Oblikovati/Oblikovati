// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Shell during ordered part replay (#4106).
use super::*;
use oblikovati_model::feature::shell::{ResolvedShell, ShellDirection};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_shell(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        openings: &[Vec<u8>],
        input: ResolvedShell,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        classify_shell(&input)?;
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.shell_candidate(self.history, self.document, openings, input.thickness)
            }
            NamedSolid::Derived(body) => {
                body.shell_candidate(self.history, self.document, openings, input.thickness)
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
fn classify_shell(input: &ResolvedShell) -> Result<(), PartFeatureFailure<EvaluationFailure>> {
    if input.direction != ShellDirection::Inside {
        return Err(PartFeatureFailure::Unsupported(format!(
            "shell direction {:?}: expected inside until exact directional offsets are available",
            input.direction
        )));
    }
    if !input.face_thicknesses.is_empty() {
        return Err(PartFeatureFailure::Unsupported(format!(
            "shell face thickness overrides {}: expected zero until varying thickness is available",
            input.face_thicknesses.len()
        )));
    }
    Ok(())
}
fn shell_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Shell
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unsupported_shell_inputs_are_declined_before_construction() {
        let mut input = ResolvedShell {
            thickness: 0.1,
            direction: ShellDirection::Inside,
            face_thicknesses: Vec::new(),
        };
        assert!(classify_shell(&input).is_ok());
        input.direction = ShellDirection::Outside;
        assert!(matches!(
            classify_shell(&input),
            Err(PartFeatureFailure::Unsupported(_))
        ));
        input.direction = ShellDirection::Both;
        assert!(matches!(
            classify_shell(&input),
            Err(PartFeatureFailure::Unsupported(_))
        ));
        input.direction = ShellDirection::Inside;
        input.face_thicknesses.push((vec![1], 0.2));
        assert!(matches!(
            classify_shell(&input),
            Err(PartFeatureFailure::Unsupported(_))
        ));
    }
}
