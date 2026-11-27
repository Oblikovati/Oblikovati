// SPDX-License-Identifier: GPL-2.0-only
//! Adopt a named append only through its owning document session.
use super::*;
use oblikovati_api::Error;
use oblikovati_martensite::PrismResolutionPolicy;
use oblikovati_model::history::EvaluationTransition;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Publish a captured append after its exact accepted transition.
    /// Example: `files.adopt_owned_part_candidate(id, Some(policy), candidate, &transition)?`.
    pub fn adopt_owned_part_candidate(
        &mut self,
        document: u64,
        policy: Option<PrismResolutionPolicy>,
        candidate: crate::OwnedPartCandidate,
        transition: &EvaluationTransition,
    ) -> Result<(), Error> {
        let policy = policy
            .map(|value| value.validate().map_err(Error))
            .transpose()?;
        let guid = self
            .workspace
            .identity(document)
            .map_err(|error| Error(error.to_string()))?
            .internal_name;
        let session = self
            .workspace
            .session(document)
            .map_err(|error| Error(error.to_string()))?;
        let completed = candidate.complete(document, guid, policy, session, transition)?;
        self.commands(document)
            .map_err(|error| Error(error.to_string()))?
            .adopt_owned_part(policy, completed)
            .map_err(Error)
    }
}
