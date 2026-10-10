// SPDX-License-Identifier: GPL-2.0-only

use super::*;

/// A validated refresh holds its owner exclusively until publication or discard.
/// Dropping this value leaves the owner unchanged.
pub struct PreparedRefresh<'a> {
    owner: &'a mut DocumentEditHistory,
    candidate: Option<(DocumentSnapshot, u64)>,
}

impl PreparedRefresh<'_> {
    /// Publish the prepared candidate without further failure points.
    /// Example: `let changed = prepared.publish();` after another owner accepts an edit.
    pub fn publish(self) -> bool {
        let Some((snapshot, generation)) = self.candidate else {
            return false;
        };
        self.owner.current = snapshot;
        self.owner.generation = generation;
        true
    }
}

impl DocumentEditHistory {
    /// Adopt an evaluation consequence without changing the undo cursor or reserving identities.
    /// Example: `history.refresh(|snapshot| refresh_links(snapshot))?`; authored edits use `edit`.
    pub fn refresh<E>(
        &mut self,
        refresh: impl FnOnce(&mut DocumentSnapshot) -> Result<(), E>,
    ) -> Result<bool, EditError<E>> {
        Ok(self.prepare_refresh(refresh)?.publish())
    }

    /// Validate a consequence without publishing it; the exclusive borrow prevents stale adoption.
    /// Example: `let prepared = history.prepare_refresh(|snapshot| update(snapshot))?;`.
    pub fn prepare_refresh<E>(
        &mut self,
        refresh: impl FnOnce(&mut DocumentSnapshot) -> Result<(), E>,
    ) -> Result<PreparedRefresh<'_>, EditError<E>> {
        self.require_closed().map_err(EditError::History)?;
        let mut candidate = self.current.clone();
        refresh(&mut candidate).map_err(EditError::Mutation)?;
        self.validate_sketch_ids(&candidate, self.sketch_reservations())
            .map_err(EditError::History)?;
        self.validate_derived_ids(&candidate, self.derived_high_water)
            .map_err(EditError::History)?;
        self.validate_candidate_ids(&candidate, self.ids.high_water())
            .map_err(EditError::History)?;
        if candidate
            .parameters
            .same_evaluated_state(&self.current.parameters)
            && candidate.properties.same_recipe(&self.current.properties)
            && candidate.sketches.same_recipe(&self.current.sketches)
            && candidate.attachments == self.current.attachments
            && candidate.resources == self.current.resources
            && candidate.interests == self.current.interests
            && candidate.attributes.same_recipe(&self.current.attributes)
        {
            return Ok(PreparedRefresh {
                owner: self,
                candidate: None,
            });
        }
        let generation = self.next_generation().map_err(EditError::History)?;
        Ok(PreparedRefresh {
            owner: self,
            candidate: Some((candidate, generation)),
        })
    }
}
