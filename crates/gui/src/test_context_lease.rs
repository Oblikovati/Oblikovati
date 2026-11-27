// SPDX-License-Identifier: GPL-2.0-only
//! Test owners serialize access to Dear ImGui's process-global current context.
use std::cell::{Cell, RefCell};
use std::rc::Rc;
use std::sync::{Mutex, MutexGuard};

static CONTEXT_LOCK: Mutex<()> = Mutex::new(());
thread_local! {
    static DEPTH: Cell<usize> = const { Cell::new(0) };
    static LEASE: RefCell<Option<MutexGuard<'static, ()>>> = const { RefCell::new(None) };
}

/// Hold the test-only process-wide context lease, allowing nested checks on one thread.
pub(crate) struct TestContextLease {
    _thread_bound: std::marker::PhantomData<Rc<()>>,
}

/// Acquire the test context lease before observing or creating the native context.
/// Example: `let _lease = crate::test_context::acquire();`.
pub(crate) fn acquire() -> TestContextLease {
    DEPTH.with(|depth| {
        if depth.get() == 0 {
            let guard = CONTEXT_LOCK
                .lock()
                .unwrap_or_else(|error| error.into_inner());
            LEASE.with(|lease| *lease.borrow_mut() = Some(guard));
        }
        depth.set(depth.get() + 1);
    });
    TestContextLease {
        _thread_bound: std::marker::PhantomData,
    }
}

impl Drop for TestContextLease {
    fn drop(&mut self) {
        DEPTH.with(|depth| {
            let remaining = depth.get() - 1;
            depth.set(remaining);
            if remaining == 0 {
                LEASE.with(|lease| drop(lease.borrow_mut().take()));
            }
        });
    }
}
