// SPDX-License-Identifier: GPL-2.0-only
//! Resolve document identity before its session recomputes named geometry.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Recompute a document's owned named part under the supplied policy.
    /// Example: `files.recompute_owned_part(id, policy, &mut attempts)?`.
    pub fn recompute_owned_part(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        let policy = policy.map(|value| value.validate()).transpose()?;
        let guid = self
            .workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        self.commands(document)
            .map_err(|error| error.to_string())?
            .recompute_owned_part(guid, policy, attempts)
    }
}
