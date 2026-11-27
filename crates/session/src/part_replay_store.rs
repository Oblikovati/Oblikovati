// SPDX-License-Identifier: GPL-2.0-only
//! Derived named part results belong to their document session.
use crate::{
    DocumentSession, TransactionObserver,
    part_evaluation::{
        CompletedPartCheckpoints, CompletedPartEvaluation, CompletedPartSketchCheckpoints,
        CompletedPartSketchRecompute, PartCheckpointSlot, PartEvaluationSlot,
    },
};
use oblikovati_martensite::{EvaluationFailure, NamedSolid, PrismResolutionPolicy};

type NamedPartSlot = PartEvaluationSlot<NamedSolid, EvaluationFailure>;
type NamedCheckpointSlot = PartCheckpointSlot<NamedSolid, EvaluationFailure>;
type NamedPartResult = CompletedPartEvaluation<NamedSolid, EvaluationFailure>;
type NamedCheckpoints = CompletedPartCheckpoints<NamedSolid, EvaluationFailure>;
type NamedSketchResult = CompletedPartSketchRecompute<NamedSolid, EvaluationFailure>;
type NamedSketchCheckpoints = CompletedPartSketchCheckpoints<NamedSolid, EvaluationFailure>;

// Bound retained kernel bodies when callers supply many resolution policies.
// Entries are ordered by publication; a successful update moves its policy to the end.
const MAX_RETAINED_POLICIES: usize = 4;

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
            .iter()
            .find(|owner| owner.policy == policy)
            .map(|owner| &owner.part)
    }
    /// Borrow checkpoints only under their recorded policy.
    /// Example: `session.owned_checkpoint_slot(policy)` before an edit.
    pub fn owned_checkpoint_slot(
        &self,
        policy: Option<PrismResolutionPolicy>,
    ) -> Option<&NamedCheckpointSlot> {
        self.part_replay
            .iter()
            .find(|owner| owner.policy == policy)
            .map(|owner| &owner.checkpoints)
    }
    /// Capture a current part prefix for an appended feature.
    /// Example: `session.owned_part_prefix(Some(policy))` before extrusion.
    pub fn owned_part_prefix(
        &self,
        policy: Option<PrismResolutionPolicy>,
    ) -> Option<crate::part_evaluation::PartCandidatePrefix<NamedSolid, EvaluationFailure>> {
        self.owned_part_slot(policy)?.candidate_prefix(self).ok()
    }
    /// Capture the most recently published current checkpoint and its policy.
    /// Example: `session.owned_checkpoint_prefix()` before an edit.
    pub fn owned_checkpoint_prefix(
        &self,
    ) -> Option<(
        Option<PrismResolutionPolicy>,
        crate::part_evaluation::PartCheckpointPrefix<NamedSolid, EvaluationFailure>,
    )> {
        self.part_replay
            .iter()
            .rev()
            .find_map(|owner| Some((owner.policy, owner.checkpoints.candidate_prefix(self).ok()?)))
    }
    fn with_part_replay<T, E>(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        operation: impl FnOnce(&mut Self, &mut DocumentPartReplay) -> Result<T, E>,
    ) -> Result<T, E> {
        let mut owners = std::mem::take(&mut self.part_replay);
        let previous_index = owners.iter().position(|owner| owner.policy == policy);
        let mut current = previous_index
            .map(|index| owners.remove(index))
            .unwrap_or_else(|| DocumentPartReplay::new(policy));
        let result = operation(self, &mut current);
        if result.is_ok() {
            owners.push(current);
            if owners.len() > MAX_RETAINED_POLICIES {
                owners.remove(0);
            }
        } else if let Some(index) = previous_index {
            owners.insert(index, current);
        }
        self.part_replay = owners;
        result
    }
    pub(crate) fn validate_prepared_named_destination(
        &self,
        policy: Option<PrismResolutionPolicy>,
    ) -> Result<(), String> {
        self.validate_prepared_replay_destination(
            self.owned_part_slot(policy),
            self.owned_checkpoint_slot(policy),
        )
        .map_err(|error| error.to_string())
    }

    pub(crate) fn publish_prepared_named_replay(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        prepared: oblikovati_model::feature::part::PartReplayCheckpoints<
            NamedSolid,
            EvaluationFailure,
        >,
    ) {
        install_prepared_named_replay(
            &mut self.part_replay,
            self.document,
            policy,
            self.history.evaluation_scope(),
            prepared,
        );
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
    /// Publish settled sketches, their part, and boundaries from one ordered replay.
    /// Example: `session.adopt_owned_sketch_checkpoints(policy, completed)?`.
    pub fn adopt_owned_sketch_checkpoints(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedSketchCheckpoints,
    ) -> Result<bool, String> {
        self.with_part_replay(policy, |session, owner| {
            session
                .adopt_checkpointed_sketch_replay(
                    completed,
                    &mut owner.part,
                    &mut owner.checkpoints,
                )
                .map_err(|error| error.to_string())
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
    /// Publish a current full replay and its boundaries together.
    /// Example: `session.adopt_owned_full_checkpoints(policy, completed)?`.
    pub fn adopt_owned_full_checkpoints(
        &mut self,
        policy: Option<PrismResolutionPolicy>,
        completed: NamedCheckpoints,
    ) -> Result<(), String> {
        self.with_part_replay(policy, |session, owner| {
            session
                .adopt_replayed_checkpoints(completed, &mut owner.part, &mut owner.checkpoints)
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

pub(crate) fn install_prepared_named_replay(
    owners: &mut Vec<DocumentPartReplay>,
    document: u64,
    policy: Option<PrismResolutionPolicy>,
    scope: oblikovati_model::history::EvaluationScope,
    prepared: oblikovati_model::feature::part::PartReplayCheckpoints<NamedSolid, EvaluationFailure>,
) {
    let index = owners.iter().position(|owner| owner.policy == policy);
    let mut owner = index
        .map(|index| owners.remove(index))
        .unwrap_or_else(|| DocumentPartReplay::new(policy));
    crate::part_evaluation::install_prepared_replay(
        document,
        scope,
        prepared,
        &mut owner.part,
        &mut owner.checkpoints,
    );
    owners.push(owner);
    if owners.len() > MAX_RETAINED_POLICIES {
        owners.remove(0);
    }
}
