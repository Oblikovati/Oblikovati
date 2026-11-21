// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate private feature previews under their document's identity and policy.
use super::*;
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid, PrismResolutionPolicy,
};
use oblikovati_model::feature::{FeatureId, part::PartFeatureEvaluation};

/// A private result and whether it reused the document's current checkpoints.
pub struct OwnedPartPreview {
    /// Evaluated candidate; it is never published as the current document.
    pub evaluation: PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
    /// Whether a clean checkpoint prefix was reused.
    pub used_checkpoint: bool,
}

/// A document-scoped preview failed before it could be displayed.
#[derive(Debug)]
pub struct PartPreviewError {
    document: u64,
    cause: String,
}
impl std::fmt::Display for PartPreviewError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "document {} part preview: {}", self.document, self.cause)
    }
}
impl std::error::Error for PartPreviewError {}

type PreviewEvaluation = PartFeatureEvaluation<NamedSolid, EvaluationFailure>;

fn preview_error(document: u64, cause: impl std::fmt::Display) -> PartPreviewError {
    PartPreviewError {
        document,
        cause: cause.to_string(),
    }
}

fn checkpoint_preview<O: TransactionObserver>(
    session: &DocumentSession<O>,
    candidate: &DocumentSnapshot,
    policy: Option<PrismResolutionPolicy>,
    evaluator: &mut CertifiedNamedSolidEvaluator<'_>,
    document: u64,
) -> Result<Option<PreviewEvaluation>, PartPreviewError> {
    if candidate.sketches.requires_part_dependency_recompute() {
        return Ok(None);
    }
    let Some(prefix) = session
        .owned_checkpoint_slot(policy)
        .and_then(|slot| slot.candidate_prefix(session).ok())
    else {
        return Ok(None);
    };
    prefix
        .prepare_candidate(session.history(), candidate, evaluator)
        .map_err(|error| preview_error(document, error))?
        .map(|result| result.copy_current(session.history()))
        .transpose()
        .map_err(|error| preview_error(document, error))
}

fn fresh_preview<O: TransactionObserver>(
    session: &DocumentSession<O>,
    candidate: &DocumentSnapshot,
    evaluator: &mut CertifiedNamedSolidEvaluator<'_>,
    document: u64,
) -> Result<PreviewEvaluation, PartPreviewError> {
    if candidate.sketches.requires_part_dependency_recompute() {
        return candidate
            .recompute_part_with_sketches(evaluator, session.sketch_solve_options())
            .map(|(part, _)| part)
            .map_err(|error| preview_error(document, error));
    }
    candidate
        .evaluate_part_features(evaluator)
        .map_err(|error| preview_error(document, error))
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Evaluate a private candidate without changing the document or its published results.
    /// Example: `workspace.preview_owned_part(id, Some(policy), &candidate)?`.
    pub fn preview_owned_part(
        &self,
        document: u64,
        policy: Option<PrismResolutionPolicy>,
        candidate: &DocumentSnapshot,
    ) -> Result<OwnedPartPreview, PartPreviewError> {
        self.preview_owned_candidate(document, policy, candidate)
    }

    /// Inspect an appended preview while its document prefix remains current.
    /// Example: `workspace.inspect_owned_append(id, policy, &candidate, feature, |body| body.as_prism().is_some())?`.
    pub fn inspect_owned_append<T>(
        &self,
        document: u64,
        policy: Option<PrismResolutionPolicy>,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        inspect: impl FnOnce(&NamedSolid) -> T,
    ) -> Result<(T, bool), PartPreviewError> {
        let session = self
            .session(document)
            .map_err(|error| preview_error(document, error))?;
        let guid = self
            .identity(document)
            .map_err(|error| preview_error(document, error))?
            .internal_name;
        session.inspect_owned_append(document, guid, policy, candidate, feature, inspect)
    }

    fn preview_owned_candidate(
        &self,
        document: u64,
        policy: Option<PrismResolutionPolicy>,
        candidate: &DocumentSnapshot,
    ) -> Result<OwnedPartPreview, PartPreviewError> {
        let session = self
            .session(document)
            .map_err(|error| preview_error(document, error))?;
        let guid = self
            .identity(document)
            .map_err(|error| preview_error(document, error))?
            .internal_name;
        session.preview_owned_part(document, guid, policy, candidate)
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    fn preview_owned_part(
        &self,
        document: u64,
        guid: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        candidate: &DocumentSnapshot,
    ) -> Result<OwnedPartPreview, PartPreviewError> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(self.history(), guid, policy);
        let reused = checkpoint_preview(self, candidate, policy, &mut evaluator, document)?;
        let used_checkpoint = reused.is_some();
        let evaluation = match reused {
            Some(result) => result,
            None => fresh_preview(self, candidate, &mut evaluator, document)?,
        };
        Ok(OwnedPartPreview {
            evaluation,
            used_checkpoint,
        })
    }
    fn inspect_owned_append<T>(
        &self,
        document: u64,
        guid: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        inspect: impl FnOnce(&NamedSolid) -> T,
    ) -> Result<(T, bool), PartPreviewError> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(self.history(), guid, policy);
        if !candidate.sketches.requires_part_dependency_recompute()
            && let Some(prefix) = self.owned_part_prefix(policy)
            && let Some(result) = prefix
                .prepare_tail(self.history(), candidate, feature, &mut evaluator)
                .map_err(|error| preview_error(document, error))?
        {
            let inspected = result
                .inspect_body(self.history(), feature, inspect)
                .map_err(|error| preview_error(document, error))?
                .ok_or_else(|| {
                    preview_error(
                        document,
                        format!("feature {feature:?}: expected a current body"),
                    )
                })?;
            return Ok((inspected, true));
        }
        let preview = self.preview_owned_part(document, guid, policy, candidate)?;
        let body = preview.evaluation.body(feature).ok_or_else(|| {
            preview_error(
                document,
                format!(
                    "feature {feature:?}: expected a certified body; health {:?}",
                    preview.evaluation.health()
                ),
            )
        })?;
        Ok((inspect(body), false))
    }
}
