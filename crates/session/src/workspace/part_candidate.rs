// SPDX-License-Identifier: GPL-2.0-only
//! Bind candidate replay to a registered document and its current checkpoint.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Bind the current named part to its registered document and policy.
    /// Example: `workspace.owned_part_replay(id, Some(policy))?` before an append.
    pub fn owned_part_replay(
        &self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    ) -> Result<crate::OwnedPartReplay, oblikovati_api::Error> {
        let session = self
            .session(document)
            .map_err(|error| oblikovati_api::Error(error.to_string()))?;
        let prefix = session.owned_part_prefix(policy).ok_or_else(|| {
            oblikovati_api::Error(format!(
                "document {document} policy {policy:?}: expected its current owned part"
            ))
        })?;
        let guid = self
            .identity(document)
            .map_err(|error| oblikovati_api::Error(error.to_string()))?
            .internal_name;
        Ok(crate::OwnedPartReplay::new(
            guid,
            policy,
            prefix,
            session.derived_sources().clone(),
        ))
    }
    /// Issue a replay capability for a current checkpoint of this document and policy.
    /// Example: `workspace.owned_checkpoint_replay(id, policy, prefix)?` before an edit.
    pub fn owned_checkpoint_replay(
        &self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        prefix: crate::part_evaluation::PartCheckpointPrefix<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<crate::OwnedCheckpointReplay, String> {
        let session = self.session(document).map_err(|error| error.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "document {document}: expected no open transaction for checkpoint replay"
            ));
        }
        let valid = session
            .owned_checkpoint_slot(policy)
            .is_some_and(|slot| slot.owns_current_prefix(session, &prefix));
        if !valid {
            return Err(format!(
                "document {document} policy {policy:?}: expected its current owned checkpoint"
            ));
        }
        let guid = self
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        Ok(crate::OwnedCheckpointReplay::new(
            document,
            guid,
            policy,
            prefix,
            session.derived_sources().clone(),
        ))
    }
}
