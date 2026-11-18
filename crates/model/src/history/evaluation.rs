// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot};
use std::sync::{
    Arc, Weak,
    atomic::{AtomicU64, Ordering},
};
/// Read-only ownership and generation proof. A weak reference cannot keep a closed history alive.
#[derive(Clone, Debug)]
pub struct EvaluationScope {
    owner: Weak<AtomicU64>,
    generation: u64,
}
impl EvaluationScope {
    /// Inspect the captured generation; example: `scope.generation()` labels an exact query.
    pub fn generation(&self) -> u64 {
        self.generation
    }
    /// Poll cancellation without borrowing the owner; example: `scope.is_superseded()` between operations.
    pub fn is_superseded(&self) -> bool {
        self.owner
            .upgrade()
            .is_none_or(|owner| owner.load(Ordering::Acquire) != self.generation)
    }
    /// Prove both history ownership and current generation; example: `scope.is_current(&history)` before adoption.
    pub fn is_current(&self, history: &DocumentEditHistory) -> bool {
        self.belongs_to(history) && !self.is_superseded()
    }
    /// Prove history ownership independently of freshness; example: `scope.belongs_to(&history)` rejects a reopened document.
    pub fn belongs_to(&self, history: &DocumentEditHistory) -> bool {
        self.owner.ptr_eq(&Arc::downgrade(&history.generation))
    }
}
/// Proof of one accepted edit, without a claim that any geometry is reusable.
pub struct EvaluationTransition {
    source: EvaluationScope,
    target: EvaluationScope,
}
impl EvaluationTransition {
    pub(super) fn new(source: EvaluationScope, target: EvaluationScope) -> Self {
        Self { source, target }
    }
    /// Verify the exact source generation and current target owner.
    /// Example: `transition.matches(&source_scope, &history)` before validating result transfer.
    pub fn matches(&self, source: &EvaluationScope, history: &DocumentEditHistory) -> bool {
        self.source.owner.ptr_eq(&source.owner)
            && self.source.generation == source.generation
            && self.target.is_current(history)
            && self.source.belongs_to(history)
    }
    /// Read the accepted target generation; example: `transition.target_generation()`.
    pub fn target_generation(&self) -> u64 {
        self.target.generation
    }
}
/// Candidate inputs and their original history, retained across one accepted edit.
/// This guard checks freshness only; it does not authorize native name restoration.
pub struct PartCandidateGuard {
    source: EvaluationScope,
    candidate: DocumentSnapshot,
}
impl PartCandidateGuard {
    /// Capture the candidate while its source is current.
    /// Example: `PartCandidateGuard::capture(&scope, &candidate, &history)?`.
    pub fn capture(
        source: &EvaluationScope,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
    ) -> Result<Self, String> {
        if !source.is_current(history) {
            return Err("part candidate guard: expected its current source history".into());
        }
        Ok(Self {
            source: source.clone(),
            candidate: candidate.clone(),
        })
    }
    /// Check the accepted transition and all replay inputs.
    /// Example: `guard.matches(&transition, &history)` before restoring bodies.
    pub fn matches(
        &self,
        transition: &EvaluationTransition,
        history: &DocumentEditHistory,
    ) -> bool {
        transition.matches_part_candidate(&self.source, &self.candidate, history)
    }
}
impl EvaluationTransition {
    /// Check both source ownership and the committed candidate replay inputs.
    /// Example: `transition.matches_part_candidate(&scope, &candidate, &history)`.
    pub fn matches_part_candidate(
        &self,
        source: &EvaluationScope,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
    ) -> bool {
        self.matches(source, history) && candidate.part_replay_change(history.snapshot()).is_none()
    }
}
/// An owned immutable evaluation recipe, separate from live history and its reservation state.
pub struct HistoryEvaluationSnapshot {
    snapshot: DocumentSnapshot,
    scope: EvaluationScope,
}
impl HistoryEvaluationSnapshot {
    /// Borrow the captured inputs; example: `job.snapshot().parameters.ids()` cannot edit live history.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        &self.snapshot
    }
    /// Borrow the sealed ownership proof; example: `job.scope().is_superseded()` after an edit.
    pub fn scope(&self) -> &EvaluationScope {
        &self.scope
    }
}
impl DocumentEditHistory {
    /// Capture a sealed ownership proof without copying recipe inputs; example: `history.evaluation_scope()` for derived names.
    pub fn evaluation_scope(&self) -> EvaluationScope {
        EvaluationScope {
            owner: Arc::downgrade(&self.generation),
            generation: self.generation(),
        }
    }
    /// Capture immutable recipe inputs and one authoritative generation counter.
    /// Example: `let job = history.evaluation_snapshot();` before releasing the owner borrow.
    pub fn evaluation_snapshot(&self) -> HistoryEvaluationSnapshot {
        HistoryEvaluationSnapshot {
            snapshot: self.current.clone(),
            scope: self.evaluation_scope(),
        }
    }
}
