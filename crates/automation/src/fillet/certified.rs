// SPDX-License-Identifier: GPL-2.0-only
//! Validate fillet candidates using an isolated certified replay (#4045).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    history::{DocumentEditHistory, SnapshotReplay},
    identity::DocumentGuid,
};
use std::sync::atomic::{AtomicU64, Ordering};

fn count_replay(counter: &AtomicU64) {
    let _ = counter.try_update(Ordering::Relaxed, Ordering::Relaxed, |value| {
        Some(value.saturating_add(1))
    });
}

/// A document-scoped validator for private, solved candidates.
pub struct CertifiedFilletValidation {
    document: DocumentGuid,
    policy: PrismResolutionPolicy,
}
/// Document-independent policy for typed Fillet operations.
pub struct CertifiedFilletAccess {
    policy: PrismResolutionPolicy,
    body_replays: AtomicU64,
    validation_replays: AtomicU64,
}
impl CertifiedFilletAccess {
    /// Bind a validated policy; example: `CertifiedFilletAccess::new(policy)?`.
    pub fn new(policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self {
            policy: policy.validate()?,
            body_replays: AtomicU64::new(0),
            validation_replays: AtomicU64::new(0),
        })
    }
    /// Count full body lookup and validation replay attempts; example: `access.replay_counts()`.
    pub fn replay_counts(&self) -> (u64, u64) {
        (
            self.body_replays.load(Ordering::Relaxed),
            self.validation_replays.load(Ordering::Relaxed),
        )
    }
}
impl FilletAccess for CertifiedFilletAccess {
    fn capture_with_prefix(
        &self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        prefix: &crate::CertifiedPartPrefix<'_>,
    ) -> Result<(usize, Option<crate::CertifiedPartCandidate>), String> {
        if prefix.policy != Some(self.policy) {
            return Err("part candidate prefix: expected the blend validation policy".into());
        }
        let expected = match kind {
            oblikovati_model::feature::part::PartBlendKind::Fillet => "fillet",
            oblikovati_model::feature::part::PartBlendKind::Chamfer => "chamfer",
        };
        require_feature_kind(candidate, feature, expected)?;
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(history, document, self.policy);
        let Some(result) =
            prefix
                .prefix
                .prepare_tail(history, candidate, feature, &mut evaluator)?
        else {
            return self
                .validate_with_prefix(kind, history, document, candidate, feature, prefix)
                .map(|count| (count, None));
        };
        let count = result.body_count();
        let candidate =
            result.capture(history, |_, body| body.capture_detached(history, document))?;
        Ok((
            count,
            Some(crate::CertifiedPartCandidate {
                policy: self.policy,
                candidate,
            }),
        ))
    }
    fn accepts_policy(&self, policy: Option<PrismResolutionPolicy>) -> bool {
        policy == Some(self.policy)
    }
    fn body(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        if edges.is_empty() {
            return Err("blend edges []: expected at least one exact edge key".into());
        }
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(history, document, self.policy);
        count_replay(&self.body_replays);
        let evaluated = history
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
        resolve_blend_body(&evaluated, history, document, edges)
    }
    fn validate(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        CertifiedFilletValidation::new(document, self.policy)?.validate_count(
            candidate,
            feature,
            "fillet",
            &self.validation_replays,
        )
    }
    fn validate_with_prefix(
        &self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        prefix: &crate::CertifiedPartPrefix<'_>,
    ) -> Result<usize, String> {
        if prefix.policy != Some(self.policy) {
            return Err("part candidate prefix: expected the blend validation policy".into());
        }
        let expected = match kind {
            oblikovati_model::feature::part::PartBlendKind::Fillet => "fillet",
            oblikovati_model::feature::part::PartBlendKind::Chamfer => "chamfer",
        };
        require_feature_kind(candidate, feature, expected)?;
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(history, document, self.policy);
        match prefix
            .prefix
            .validate_tail(history, candidate, feature, &mut evaluator)?
        {
            Some(count) => Ok(count),
            None => match kind {
                oblikovati_model::feature::part::PartBlendKind::Fillet => {
                    self.validate(document, candidate, feature)
                }
                oblikovati_model::feature::part::PartBlendKind::Chamfer => {
                    self.validate_chamfer(document, candidate, feature)
                }
            },
        }
    }
    fn validate_chamfer(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        CertifiedFilletValidation::new(document, self.policy)?.validate_count(
            candidate,
            feature,
            "chamfer",
            &self.validation_replays,
        )
    }
}

pub(crate) fn resolve_blend_body(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    edges: &[Vec<u8>],
) -> Result<Vec<u8>, String> {
    if edges.is_empty() {
        return Err("blend edges []: expected at least one exact edge key".into());
    }
    let mut owners = Vec::new();
    for body in evaluated.ordered_bodies() {
        let keys = body.subentity_reference_keys(history, document)?;
        if edges.iter().all(|edge| keys.contains(edge)) {
            owners.push(body.body_reference_key(history, document)?.to_vec());
        }
    }
    match owners.as_slice() {
        [body] => Ok(body.clone()),
        _ => Err(format!(
            "blend edges {edges:?}, matching bodies {}: expected one current body containing every edge",
            owners.len()
        )),
    }
}
impl CertifiedFilletValidation {
    /// Bind the document and explicit kernel policy; example: `CertifiedFilletValidation::new(document, policy)?`.
    pub fn new(document: DocumentGuid, policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self {
            document,
            policy: policy.validate()?,
        })
    }
}
impl FilletValidation for CertifiedFilletValidation {
    fn validate(&self, candidate: &DocumentSnapshot, feature: FeatureId) -> Result<(), String> {
        self.validate_count(candidate, feature, "fillet", &AtomicU64::new(0))
            .map(|_| ())
    }
}
impl CertifiedFilletValidation {
    fn validate_count(
        &self,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        expected: &str,
        replay_count: &AtomicU64,
    ) -> Result<usize, String> {
        require_feature_kind(candidate, feature, expected)?;
        let history = SnapshotReplay::new(candidate).map_err(|cause| cause.to_string())?;
        let mut evaluator = CertifiedNamedSolidEvaluator::with_policy(
            history.history(),
            self.document,
            self.policy,
        );
        count_replay(replay_count);
        let result = history
            .history()
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
        if result.body(feature).is_none() {
            return Err(format!(
                "{expected} {feature:?}, health {:?}: expected a valid named result",
                result.health()
            ));
        }
        Ok(result.ordered_bodies().count())
    }
}

fn require_feature_kind(
    candidate: &DocumentSnapshot,
    feature: FeatureId,
    expected: &str,
) -> Result<(), String> {
    let authored = candidate
        .work_geometry
        .get(feature)
        .map_err(|cause| cause.to_string())?;
    if authored.definition().kind() == expected {
        return Ok(());
    }
    Err(format!(
        "feature {feature:?}, kind {}: expected {expected}",
        authored.definition().kind()
    ))
}
