// SPDX-License-Identifier: GPL-2.0-only
use super::{SessionCommands, WorkspaceObserver, WorkspaceSession};
use crate::{
    TransactionObserver,
    part_evaluation::{
        CompletedPartCheckpoints, CompletedPartEvaluation, CompletedPartSketchCheckpoints,
        CompletedPartSketchRecompute, PartCheckpointSlot, PartEvaluationError,
        PartEvaluationRequest, PartEvaluationSlot, PartSketchAdoptionError,
    },
};

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Inspect sketch datums from the live generation without publishing a part result.
    /// Example: `workspace.inspect_part_sketch_datums(document, policy)?`.
    pub fn inspect_part_sketch_datums(
        &self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        let guid = self
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        self.session(document)
            .map_err(|error| error.to_string())?
            .inspect_owned_sketch_datums(guid, policy)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    pub(crate) fn ensure_owned_checkpoints(
        &mut self,
        guid: oblikovati_model::identity::DocumentGuid,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        self.session
            .ensure_owned_checkpoints(guid, policy, attempts)
    }
    pub(crate) fn recompute_owned_part(
        &mut self,
        guid: oblikovati_model::identity::DocumentGuid,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        attempts: &mut u64,
    ) -> Result<(), String> {
        self.session.recompute_owned_part(guid, policy, attempts)
    }
    /// Publish an accepted checkpoint set with its part result.
    /// Example: `files.commands(id)?.adopt_replayed_checkpoints(done, &mut part, &mut checkpoints)?`.
    pub fn adopt_replayed_checkpoints<Body: Clone, Failure: Clone>(
        &mut self,
        completed: CompletedPartCheckpoints<Body, Failure>,
        part: &mut PartEvaluationSlot<Body, Failure>,
        checkpoints: &mut PartCheckpointSlot<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        self.session
            .adopt_replayed_checkpoints(completed, part, checkpoints)
    }
    /// Publish solved part checkpoints through their document owner.
    /// Example: `files.commands(id)?.adopt_checkpointed_sketch_replay(done, &mut part, &mut checkpoints)?`.
    pub fn adopt_checkpointed_sketch_replay<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchCheckpoints<Body, Failure>,
        part: &mut PartEvaluationSlot<Body, Failure>,
        checkpoints: &mut PartCheckpointSlot<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        self.session
            .adopt_checkpointed_sketch_replay(completed, part, checkpoints)
    }
    /// Publish current recomputed sketches without a new undo step.
    /// Example: `files.commands(id)?.adopt_recomputed_part_sketches(completed)?`.
    pub fn adopt_recomputed_part_sketches<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        self.session.adopt_recomputed_part_sketches(completed)
    }
    /// Adopt an unchanged replay, or refresh sketches for another replay.
    /// Example: `files.commands(id)?.adopt_sketch_replay(completed, &mut slot)?`.
    pub fn adopt_sketch_replay<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        self.session.adopt_sketch_replay(completed, slot)
    }
    /// Capture from a fixed file-owned document without keeping the borrowed capability.
    /// Example: `files.commands(id)?.prepare_part_evaluation()` before kernel work.
    pub fn prepare_part_evaluation(&self) -> PartEvaluationRequest {
        self.session.prepare_part_evaluation()
    }
    /// Publish a derived result through its existing owner without creating an undo command.
    /// Example: `files.commands(id)?.adopt_part_evaluation(completed, &mut slot)?`.
    pub fn adopt_part_evaluation<Body, Failure>(
        &mut self,
        completed: CompletedPartEvaluation<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        self.session.adopt_part_evaluation(completed, slot)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Publish a named part under its document policy; example: `commands.adopt_owned_part(policy, result)?`.
    pub fn adopt_owned_part(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<(), String> {
        self.session.adopt_owned_part(policy, completed)
    }
    /// Publish named sketch replay; example: `commands.adopt_owned_sketches(policy, result)?`.
    pub fn adopt_owned_sketches(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartSketchRecompute<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<bool, String> {
        self.session.adopt_owned_sketches(policy, completed)
    }
    /// Publish settled sketch checkpoints; example: `commands.adopt_owned_sketch_checkpoints(policy, result)?`.
    pub fn adopt_owned_sketch_checkpoints(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartSketchCheckpoints<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<bool, String> {
        self.session
            .adopt_owned_sketch_checkpoints(policy, completed)
    }
    /// Retain named checkpoints; example: `commands.adopt_owned_checkpoints(policy, result)?`.
    pub fn adopt_owned_checkpoints(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartCheckpoints<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<(), String> {
        self.session.adopt_owned_checkpoints(policy, completed)
    }
    /// Publish a full replay and its boundaries; example: `commands.adopt_owned_full_checkpoints(policy, result)?`.
    pub fn adopt_owned_full_checkpoints(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartCheckpoints<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<(), String> {
        self.session.adopt_owned_full_checkpoints(policy, completed)
    }
    /// Publish named replay checkpoints; example: `commands.adopt_owned_replayed_checkpoints(policy, result)?`.
    pub fn adopt_owned_replayed_checkpoints(
        &mut self,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        completed: CompletedPartCheckpoints<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
    ) -> Result<(), String> {
        self.session
            .adopt_owned_replayed_checkpoints(policy, completed)
    }
}
