// SPDX-License-Identifier: GPL-2.0-only
//! Guard context creation in tests that instantiate Dear ImGui directly.
use std::ops::{Deref, DerefMut};

#[path = "test_context_lease.rs"]
mod context_lease;
pub(crate) use context_lease::{TestContextLease, acquire};

/// An ImGui context that keeps its test lease until after the context is dropped.
pub(crate) struct TestContext {
    context: imgui::Context,
    _lease: TestContextLease,
}

/// Create one guarded test context; example: `let context = test_context::create()`.
pub(crate) fn create() -> TestContext {
    let lease = acquire();
    TestContext {
        context: imgui::Context::create(),
        _lease: lease,
    }
}
impl Deref for TestContext {
    type Target = imgui::Context;
    fn deref(&self) -> &Self::Target {
        &self.context
    }
}
impl DerefMut for TestContext {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.context
    }
}
