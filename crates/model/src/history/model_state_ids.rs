// SPDX-License-Identifier: GPL-2.0-only
//! Member identities use the permanent recipe reservation stream; Primary is built in.
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations};

pub(super) fn authored_ids(snapshot: &DocumentSnapshot) -> Vec<u64> {
    snapshot
        .model_states
        .iter()
        .flat_map(|members| members.list())
        .filter(|member| member.id != 1)
        .map(|member| member.id)
        .collect()
}
impl DocumentSnapshot {
    pub(super) fn validate_model_state_transition(
        &self,
        before: &Self,
    ) -> Result<(), HistoryError> {
        if before.model_states.is_some() && self.model_states.is_none() {
            return Err(HistoryError::ModelStateMembers(
                "model-state content None: expected retained part members".into(),
            ));
        }
        self.validate_model_state_members()
    }

    pub(super) fn validate_model_state_members(&self) -> Result<(), HistoryError> {
        self.model_states
            .as_ref()
            .map_or(Ok(()), |members| members.validate())
            .map_err(HistoryError::ModelStateMembers)
    }
}
impl DocumentEditHistory {
    pub(super) fn validate_model_state_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: u64,
    ) -> Result<(), HistoryError> {
        candidate.validate_model_state_members()?;
        super::sketch_ids::require_reserved(
            authored_ids(candidate),
            authored_ids(&self.current),
            before,
            self.high_water(),
            HistoryError::UnreservedIdentity,
        )
    }
}
impl RecipeReservations<'_> {
    /// Reserve an authored member identity that undo cannot reuse. Example: `ids.allocate_model_state()?`.
    pub fn allocate_model_state(&mut self) -> Result<u64, crate::identity::RecipeIdError> {
        let reserved = self.allocate()?;
        if reserved == 1 {
            return self.allocate();
        }
        Ok(reserved)
    }
}
