// SPDX-License-Identifier: GPL-2.0-only

use super::*;

/// A validated refresh holds its owner exclusively until publication or discard.
/// Dropping this value leaves the owner unchanged.
pub struct PreparedRefresh<'a> {
    owner: &'a mut DocumentEditHistory,
    candidate: Option<(DocumentSnapshot, u64)>,
}

impl PreparedRefresh<'_> {
    /// Read the staged snapshot; example: `prepared.snapshot()` before dependent replay.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        self.candidate
            .as_ref()
            .map_or(&self.owner.current, |(snapshot, _)| snapshot)
    }
    /// Read the unchanged target history; example: `prepared.source()` before capture.
    pub fn source(&self) -> &DocumentEditHistory {
        self.owner
    }

    fn publish_scope(self) -> (bool, super::EvaluationScope) {
        let owner = self.owner;
        let changed = if let Some((snapshot, generation)) = self.candidate {
            owner.current = snapshot;
            owner.generation.store(generation, Ordering::Release);
            true
        } else {
            false
        };
        (changed, owner.evaluation_scope())
    }
    /// Publish the prepared candidate without further failure points.
    /// Example: `let changed = prepared.publish();` after another owner accepts an edit.
    pub fn publish(self) -> bool {
        self.publish_scope().0
    }
}

/// A validated dependent refresh sealed to one source candidate (#4122).
/// Example: `PreparedDependentRefresh::new(&edit, dependent)`.
pub struct PreparedDependentRefresh<'a> {
    seal: super::PreparedCandidateSeal,
    refresh: PreparedRefresh<'a>,
}
impl<'a> PreparedDependentRefresh<'a> {
    /// Read the staged dependent snapshot; example: `dependent.snapshot()` before replay.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        self.refresh.snapshot()
    }
    /// Read its unchanged target history; example: `dependent.source()` before capture.
    pub fn source(&self) -> &DocumentEditHistory {
        self.refresh.source()
    }
    pub(crate) fn seal(&self) -> super::PreparedCandidateSeal {
        self.seal.clone()
    }
    /// Associate an already validated refresh with its exact source candidate.
    /// Example: `PreparedDependentRefresh::new(&edit, dependent)`.
    pub fn new(edit: &super::PreparedPrivateEdit<'_>, refresh: PreparedRefresh<'a>) -> Self {
        Self {
            seal: edit.seal(),
            refresh,
        }
    }
}
impl super::publication::private::Sealed for PreparedDependentRefresh<'_> {}
impl super::PreparedPublication for PreparedDependentRefresh<'_> {
    type Output = bool;
    fn validate(&self, edit: &super::PreparedPrivateEdit<'_>) -> Result<(), String> {
        if !self.seal.matches(edit) {
            return Err("prepared dependent refresh: expected its exact source candidate".into());
        }
        Ok(())
    }
    fn install(self, _: &super::PublicationToken) -> bool {
        self.refresh.publish()
    }
}

/// A dependent snapshot and its validated replay, published under the target scope.
/// Example: `PreparedDependentPublication::new(refresh, replay)`.
pub struct PreparedDependentPublication<'a, P> {
    refresh: PreparedDependentRefresh<'a>,
    publication: P,
}
impl<'a, P> PreparedDependentPublication<'a, P> {
    /// Pair a refresh with its prepared replay; example: `PreparedDependentPublication::new(refresh, replay)`.
    pub fn new(refresh: PreparedDependentRefresh<'a>, publication: P) -> Self {
        Self {
            refresh,
            publication,
        }
    }
}
impl<P: super::PreparedPublication> super::publication::private::Sealed
    for PreparedDependentPublication<'_, P>
{
}
impl<P: super::PreparedPublication> super::PreparedPublication
    for PreparedDependentPublication<'_, P>
{
    type Output = (bool, super::EvaluationScope, P::Output);
    fn validate(&self, edit: &super::PreparedPrivateEdit<'_>) -> Result<(), String> {
        self.refresh.validate(edit)?;
        self.publication.validate(edit)?;
        self.publication
            .validate_target(&self.refresh.source().evaluation_scope())
    }
    fn install(self, _: &super::PublicationToken) -> Self::Output {
        let (changed, scope) = self.refresh.refresh.publish_scope();
        let token = super::PublicationToken::for_scope(scope.clone());
        (changed, scope, self.publication.install(&token))
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
        candidate
            .validate_snapshot_transition(&self.current)
            .map_err(EditError::History)?;
        self.validate_feature_ids(&candidate, self.feature_high_water())
            .map_err(EditError::History)?;
        self.validate_occurrence_ids(&candidate, self.occurrence_high_water())
            .map_err(EditError::History)?;
        self.validate_sketch_ids(&candidate, self.sketch_reservations())
            .map_err(EditError::History)?;
        self.validate_derived_ids(&candidate, self.derived_high_water)
            .map_err(EditError::History)?;
        self.validate_candidate_ids(&candidate, self.ids.high_water())
            .map_err(EditError::History)?;
        if candidate
            .parameters
            .same_evaluated_state(&self.current.parameters)
            && candidate
                .work_geometry
                .same_work_plane_recipe(&self.current.work_geometry)
            && candidate
                .sketches
                .same_evaluated_state(&self.current.sketches)
            && candidate.same_metadata_recipe(&self.current)
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
