// SPDX-License-Identifier: GPL-2.0-only
//! Retire shared prefixes when their document replaces or clears a result.
use std::ops::Deref;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};

pub(super) struct Retained<T> {
    value: T,
    lease: Arc<AtomicBool>,
}
impl<T> Retained<T> {
    pub(super) fn new(value: T) -> Arc<Self> {
        Arc::new(Self {
            value,
            lease: Arc::new(AtomicBool::new(true)),
        })
    }
    pub(super) fn lease(&self) -> Arc<AtomicBool> {
        Arc::clone(&self.lease)
    }
    pub(super) fn retire(&self) {
        self.lease.store(false, Ordering::Release);
    }
    pub(super) fn is_active(&self) -> bool {
        self.lease.load(Ordering::Acquire)
    }
}
impl<T> Deref for Retained<T> {
    type Target = T;
    fn deref(&self) -> &Self::Target {
        &self.value
    }
}
pub(super) fn lease_is_active(lease: &AtomicBool) -> bool {
    lease.load(Ordering::Acquire)
}

impl<Body, Failure> super::PartEvaluationSlot<Body, Failure> {
    /// Retire the retained output; example: `slot.clear()` before closing its owner.
    pub fn clear(&mut self) {
        if let Some(previous) = self.published.take() {
            previous.retire();
        }
    }
    pub(super) fn publish(&mut self, result: super::CompletedPartEvaluation<Body, Failure>) {
        self.clear();
        self.published = Some(Retained::new(result));
    }
}
impl<Body, Failure> Drop for super::PartEvaluationSlot<Body, Failure> {
    fn drop(&mut self) {
        self.clear();
    }
}
