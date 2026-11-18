// SPDX-License-Identifier: GPL-2.0-only
//! Adopt accepted part candidates through their policy owner.
use super::*;

pub(super) type NamedSketchRecompute =
    oblikovati_session::part_evaluation::CompletedPartSketchRecompute<
        NamedSolid,
        EvaluationFailure,
    >;

/// A current part prefix bound to the policy that produced it.
pub struct CertifiedPartPrefix<'a> {
    pub(crate) prefix:
        oblikovati_session::part_evaluation::PartCandidatePrefix<'a, NamedSolid, EvaluationFailure>,
    pub(crate) policy: Option<PrismResolutionPolicy>,
}
/// A candidate sealed to the policy that produced its exact bodies.
pub struct CertifiedPartCandidate {
    pub(crate) policy: PrismResolutionPolicy,
    pub(crate) candidate: oblikovati_session::part_evaluation::CapturedPartCandidate<
        oblikovati_martensite::CapturedNamedSolid,
        EvaluationFailure,
    >,
}

impl PartReplayService {
    pub(super) fn adopt_named_candidate<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedPartCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String> {
        if self.resolution != Some(candidate.policy) {
            return Err(format!(
                "candidate policy {:?}: expected {:?}",
                candidate.policy, self.resolution
            ));
        }
        let workspace = files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let completed = oblikovati_martensite::complete_named_candidate(
            candidate.candidate,
            transition,
            session,
            guid,
        )?;
        let slot = self
            .slots
            .get_mut(&document)
            .ok_or_else(|| format!("document {document}: expected its retained part slot"))?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_part_evaluation(completed, slot)
            .map_err(|error| error.to_string())
    }
}
