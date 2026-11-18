// SPDX-License-Identifier: GPL-2.0-only
//! Derived named part results belong to their document session.
use crate::{
    DocumentSession, TransactionObserver,
    part_evaluation::{
        CompletedPartCheckpoints, CompletedPartEvaluation, CompletedPartSketchRecompute,
        PartCheckpointSlot, PartEvaluationSlot,
    },
};
use oblikovati_martensite::{EvaluationFailure, NamedSolid, PrismResolutionPolicy};

type NamedPartSlot = PartEvaluationSlot<NamedSolid, EvaluationFailure>;
type NamedCheckpointSlot = PartCheckpointSlot<NamedSolid, EvaluationFailure>;
type NamedPartResult = CompletedPartEvaluation<NamedSolid, EvaluationFailure>;
type NamedCheckpoints = CompletedPartCheckpoints<NamedSolid, EvaluationFailure>;
type NamedSketchResult = CompletedPartSketchRecompute<NamedSolid, EvaluationFailure>;

pub(crate) struct DocumentPartReplay {
    policy: Option<PrismResolutionPolicy>,
    part: NamedPartSlot,
    checkpoints: NamedCheckpointSlot,
}
impl DocumentPartReplay {
    fn new(policy: Option<PrismResolutionPolicy>) -> Self {
        Self {
            policy,
            part: Default::default(),
            checkpoints: Default::default(),
        }
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Borrow a part only under the policy that produced it.
    /// Example: `session.owned_part_slot(policy)` before a named query.
    pub fn owned_part_slot(&self, policy: Option<PrismResolutionPolicy>) -> Option<&NamedPartSlot> {
        self.part_replay
            .as_ref()
            .filter(|owner| owner.policy == policy)
            .map(|owner| &owner.part)
    }
    /// Borrow checkpoints only under their recorded policy.
    /// Example: `session.owned_checkpoint_slot(policy)` before an edit.
    pub fn owned_checkpoint_slot(
        &self,
        policy: Option<PrismResolutionPolicy>,
    ) -> Option<&NamedCheckpointSlot> {
        self.part_replay
            .as_ref()
            .filter(|owner| owner.policy == policy)
            .map(|owner| &owner.checkpoints)
    }
    fn with_part_replay<T, E>(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        operation: impl FnOnce(&mut Self, &mut DocumentPartReplay) -> Result<T, E>,
    ) -> Result<T, E> {
        let previous = self.part_replay.take();
        let (previous, mut current, reused) = match previous {
            Some(owner) if owner.policy == policy => (None, owner, true),
            other => (other, DocumentPartReplay::new(policy), false),
        };
        let result = operation(self, &mut current);
        self.part_replay = if result.is_ok() || reused {
            Some(current)
        } else {
            previous
        };
        result
    }
    /// Publish a named part into its document-owned slot.
    /// Example: `session.adopt_owned_part(policy, completed)?`.
    pub fn adopt_owned_part(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedPartResult,
    ) -> Result<(), String> {
        self.with_part_replay(policy, |session, owner| {
            session
                .adopt_part_evaluation(completed, &mut owner.part)
                .map_err(|error| error.to_string())?;
            owner.checkpoints = Default::default();
            Ok(())
        })
    }
    /// Publish an unchanged sketch replay or update its sketches.
    /// Example: `session.adopt_owned_sketches(policy, completed)?`.
    pub fn adopt_owned_sketches(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedSketchResult,
    ) -> Result<bool, String> {
        self.with_part_replay(policy, |session, owner| {
            let published = session
                .adopt_sketch_replay(completed, &mut owner.part)
                .map_err(|error| error.to_string())?;
            if published {
                owner.checkpoints = Default::default();
            }
            Ok(published)
        })
    }
    /// Store current feature boundaries for later edits.
    /// Example: `session.adopt_owned_checkpoints(policy, completed)?`.
    pub fn adopt_owned_checkpoints(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedCheckpoints,
    ) -> Result<(), String> {
        self.with_part_replay(policy, |session, owner| {
            owner
                .part
                .current(session)
                .map_err(|error| error.to_string())?;
            owner
                .checkpoints
                .adopt(session, completed)
                .map_err(|error| error.to_string())
        })
    }
    /// Publish a replayed result and its checkpoints as one operation.
    /// Example: `session.adopt_owned_replayed_checkpoints(policy, completed)?`.
    pub fn adopt_owned_replayed_checkpoints(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedCheckpoints,
    ) -> Result<(), String> {
        self.with_part_replay(policy, |session, owner| {
            owner
                .part
                .replay_change(session)
                .map_err(|error| error.to_string())?;
            session
                .adopt_replayed_checkpoints(completed, &mut owner.part, &mut owner.checkpoints)
                .map_err(|error| error.to_string())
        })
    }
}
