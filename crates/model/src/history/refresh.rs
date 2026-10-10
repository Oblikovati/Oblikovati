// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentEditHistory {
    /// Adopt an evaluation consequence without changing the undo cursor or reserving identities.
    /// Example: `history.refresh(|snapshot| refresh_links(snapshot))?`; authored edits use `edit`.
    pub fn refresh<E>(
        &mut self,
        refresh: impl FnOnce(&mut DocumentSnapshot) -> Result<(), E>,
    ) -> Result<bool, EditError<E>> {
        self.require_closed().map_err(EditError::History)?;
        let mut candidate = self.current.clone();
        refresh(&mut candidate).map_err(EditError::Mutation)?;
        self.validate_candidate_ids(&candidate, self.ids.high_water())
            .map_err(EditError::History)?;
        if candidate
            .parameters
            .same_evaluated_state(&self.current.parameters)
            && candidate.properties.same_recipe(&self.current.properties)
        {
            return Ok(false);
        }
        let generation = self.next_generation().map_err(EditError::History)?;
        self.current = candidate;
        self.generation = generation;
        Ok(true)
    }
}
