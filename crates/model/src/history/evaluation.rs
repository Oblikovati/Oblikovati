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
    /// Capture immutable recipe inputs and one authoritative generation counter.
    /// Example: `let job = history.evaluation_snapshot();` before releasing the owner borrow.
    pub fn evaluation_snapshot(&self) -> HistoryEvaluationSnapshot {
        HistoryEvaluationSnapshot {
            snapshot: self.current.clone(),
            scope: EvaluationScope {
                owner: Arc::downgrade(&self.generation),
                generation: self.generation(),
            },
        }
    }
}
