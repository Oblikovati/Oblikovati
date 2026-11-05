// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::panic::{AssertUnwindSafe, catch_unwind};
impl<W: DispatchWakeup> OwnerDispatch<W> {
    /// Run at most the frame budget; zero drains the current admission snapshot, not future arrivals.
    /// Example: `owner.drain(32, &host)?;` keeps the borrowed model on its own thread.
    pub fn drain<C: Caller>(&mut self, limit: usize, caller: &C) -> Result<usize, Error> {
        let pending = self.pending()?;
        let budget = drain_budget(limit, pending);
        let mut ran = 0;
        for _ in 0..budget {
            let Some(request) = self.take_request()? else {
                break;
            };
            let healthy = execute_request(caller, request);
            ran += 1;
            if !healthy {
                self.close();
                break;
            }
        }
        Ok(ran)
    }
    /// Inspect queued requests without capturing model state; example: `owner.pending()?`.
    pub fn pending(&self) -> Result<usize, Error> {
        Ok(self.shared.lock("pending count")?.requests.len())
    }
    fn take_request(&self) -> Result<Option<QueuedRequest>, Error> {
        let mut status = self.shared.lock("owner drain")?;
        let request = status.requests.pop_front();
        if request.is_some() {
            self.shared.capacity_available.notify_one();
        }
        Ok(request)
    }
    /// Stop admission and release all undrained waiters; repeated close is harmless.
    /// Example: `owner.close()` after the add-in deactivation phase.
    pub fn close(&mut self) {
        self.shared.close();
    }
}
impl<W: DispatchWakeup> Drop for OwnerDispatch<W> {
    fn drop(&mut self) {
        self.shared.close();
    }
}
fn execute_request<C: Caller>(caller: &C, request: QueuedRequest) -> bool {
    let outcome = catch_unwind(AssertUnwindSafe(|| {
        caller.call(&request.method, &request.arguments)
    }));
    let (reply, healthy) = match outcome {
        Ok(reply) => (reply, true),
        Err(_) => (
            Err(Error(format!(
                "dispatch method {:?} panicked: expected a normal host result; admission stopped",
                request.method
            ))),
            false,
        ),
    };
    // An abandoned waiter cannot block the owner or revoke an already accepted request.
    let _ = request.reply.send(reply);
    healthy
}

fn drain_budget(limit: usize, pending: usize) -> usize {
    if limit == 0 {
        return pending;
    }
    limit.min(pending)
}
