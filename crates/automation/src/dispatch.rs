// SPDX-License-Identifier: GPL-2.0-only
//! Bounded copied requests; model state stays in the owner-thread Caller supplied to drain.
use oblikovati_api::{Error, client::Caller};
use std::{
    collections::VecDeque,
    marker::PhantomData,
    rc::Rc,
    sync::{
        Arc, Condvar, Mutex, MutexGuard,
        mpsc::{self, Receiver, Sender},
    },
    thread::{self, ThreadId},
    time::{Duration, Instant},
};
mod caller;
mod owner;
mod shared;
pub use caller::QueuedCaller;

/// Wake an idle owner after admission (#1493); implementations must not block or panic.
pub trait DispatchWakeup: Send + Sync {
    /// Signal the native loop or report failure; example: post one empty window event.
    fn wake(&self) -> Result<(), String>;
}
/// Explicit polling mode for a CLI owner that already drains without native window wakeups.
pub struct PollingDispatchWakeup;
impl DispatchWakeup for PollingDispatchWakeup {
    fn wake(&self) -> Result<(), String> {
        Ok(())
    }
}
struct QueuedRequest {
    method: String,
    arguments: Vec<u8>,
    reply: Sender<Result<Vec<u8>, Error>>,
}
struct QueueStatus {
    closed: bool,
    requests: VecDeque<QueuedRequest>,
}
struct SharedQueue<W> {
    status: Mutex<QueueStatus>,
    capacity_available: Condvar,
    capacity: usize,
    timeout: Duration,
    owner: ThreadId,
    wakeup: W,
}
/// A non-Send drain capability, created and used on the model owner thread.
/// Moving it to a worker is rejected at compile time:
/// ```compile_fail
/// use oblikovati_automation::dispatch::{owner_dispatch, PollingDispatchWakeup};
/// let (mut owner, _) = owner_dispatch(1, std::time::Duration::from_secs(1), PollingDispatchWakeup);
/// std::thread::spawn(move || owner.close());
/// ```
pub struct OwnerDispatch<W: DispatchWakeup> {
    shared: Arc<SharedQueue<W>>,
    owner_thread: PhantomData<Rc<()>>,
}
/// Create bounded admission and a typed-client transport; zero capacity follows Go's minimum of one.
/// Example: `let (mut owner, caller) = owner_dispatch(16, timeout, window_wakeup);`.
pub fn owner_dispatch<W: DispatchWakeup>(
    capacity: usize,
    timeout: Duration,
    wakeup: W,
) -> (OwnerDispatch<W>, QueuedCaller<W>) {
    let shared = Arc::new(SharedQueue {
        status: Mutex::new(QueueStatus {
            closed: false,
            requests: VecDeque::new(),
        }),
        capacity_available: Condvar::new(),
        capacity: capacity.max(1),
        timeout,
        owner: thread::current().id(),
        wakeup,
    });
    let owner = OwnerDispatch {
        shared: Arc::clone(&shared),
        owner_thread: PhantomData,
    };
    (owner, QueuedCaller { shared })
}
