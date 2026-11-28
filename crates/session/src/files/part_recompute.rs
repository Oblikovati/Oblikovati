// SPDX-License-Identifier: GPL-2.0-only
//! Resolve document identity before its session recomputes named geometry.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Ensure a document has current policy-scoped feature boundaries.
    /// Example: `files.ensure_owned_checkpoints(id, policy, &mut attempts)?`.
    pub fn ensure_owned_checkpoints(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        self.sync_derived_sources(document, &[])?;
        let (guid, policy) = self.named_part_identity(document, policy)?;
        self.commands(document)
            .map_err(|error| error.to_string())?
            .ensure_owned_checkpoints(guid, policy, attempts)
    }
    /// Recompute a document's owned named part under the supplied policy.
    /// Example: `files.recompute_owned_part(id, policy, &mut attempts)?`.
    pub fn recompute_owned_part(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        self.sync_derived_sources(document, &[])?;
        let (guid, policy) = self.named_part_identity(document, policy)?;
        self.commands(document)
            .map_err(|error| error.to_string())?
            .recompute_owned_part(guid, policy, attempts)
    }
    fn named_part_identity(
        &self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    ) -> Result<
        (
            oblikovati_model::identity::DocumentGuid,
            Option<oblikovati_martensite::PrismResolutionPolicy>,
        ),
        String,
    > {
        let policy = policy.map(|value| value.validate()).transpose()?;
        let guid = self
            .workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        Ok((guid, policy))
    }
}
