// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, client::Caller};
use oblikovati_automation::dispatch::*;
use std::{
    cell::RefCell,
    sync::mpsc::{self, Receiver, Sender},
    thread::{self, JoinHandle, ThreadId},
    time::Duration,
};
#[path = "support/dispatch_wakeup.rs"]
mod dispatch_wakeup;
#[path = "dispatch/lifecycle.rs"]
mod lifecycle;
use dispatch_wakeup::WakeSignal;
type SignalOwner = OwnerDispatch<WakeSignal>;
type SignalCaller = QueuedCaller<WakeSignal>;
fn queue_fixture(capacity: usize) -> (SignalOwner, SignalCaller, Receiver<()>) {
    timed_fixture(capacity, Duration::from_secs(2))
}
fn timed_fixture(capacity: usize, timeout: Duration) -> (SignalOwner, SignalCaller, Receiver<()>) {
    let (signal, awake) = mpsc::channel();
    let (owner, caller) = owner_dispatch(capacity, timeout, WakeSignal(signal));
    (owner, caller, awake)
}
fn start_call<W: DispatchWakeup + 'static>(
    caller: QueuedCaller<W>,
    method: &'static str,
    bytes: &[u8],
) -> JoinHandle<Result<Vec<u8>, Error>> {
    let copied = bytes.to_vec();
    thread::spawn(move || caller.call(method, &copied))
}
fn wait_awake(awake: &Receiver<()>) {
    awake.recv_timeout(Duration::from_secs(2)).unwrap();
}
#[derive(Default)]
struct RecordingCaller {
    requests: RefCell<Vec<(String, Vec<u8>, ThreadId)>>,
}
impl Caller for RecordingCaller {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        self.requests
            .borrow_mut()
            .push((method.into(), request.to_vec(), thread::current().id()));
        Ok(request.to_vec())
    }
}
#[test]
fn copied_requests_execute_only_when_the_owner_drains_and_keep_fifo_replies() {
    let (mut owner, caller, awake) = queue_fixture(3);
    let first = start_call(caller.clone(), "first", b"one");
    wait_awake(&awake);
    let second = start_call(caller, "second", b"two");
    wait_awake(&awake);
    let host = RecordingCaller::default();
    assert!(host.requests.borrow().is_empty());
    assert_eq!(owner.pending().unwrap(), 2);
    assert_eq!(owner.drain(1, &host).unwrap(), 1);
    assert_eq!(first.join().unwrap().unwrap(), b"one");
    assert_eq!(owner.pending().unwrap(), 1);
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(second.join().unwrap().unwrap(), b"two");
    assert_owner_requests(&host);
}
fn assert_owner_requests(host: &RecordingCaller) {
    let requests = host.requests.borrow();
    assert_eq!(
        requests
            .iter()
            .map(|row| row.0.as_str())
            .collect::<Vec<_>>(),
        ["first", "second"]
    );
    assert!(requests.iter().all(|row| row.2 == thread::current().id()));
}
#[test]
fn a_full_queue_applies_backpressure_until_owner_drain_releases_capacity() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let first = start_call(caller.clone(), "first", b"one");
    wait_awake(&awake);
    let second = start_call(caller, "second", b"two");
    assert_eq!(owner.pending().unwrap(), 1);
    let host = RecordingCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    wait_awake(&awake);
    assert_eq!(owner.pending().unwrap(), 1);
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(first.join().unwrap().unwrap(), b"one");
    assert_eq!(second.join().unwrap().unwrap(), b"two");
    assert_owner_requests(&host);
}
#[test]
fn polling_wakeup_still_allows_a_cli_owner_to_drain_without_native_signals() {
    let (mut owner, caller) = owner_dispatch(1, Duration::from_secs(2), PollingDispatchWakeup);
    let worker = start_call(caller, "poll", b"reply");
    wait_pending(&owner);
    let host = RecordingCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(worker.join().unwrap().unwrap(), b"reply");
}
fn wait_pending<W: DispatchWakeup>(owner: &OwnerDispatch<W>) {
    let deadline = std::time::Instant::now() + Duration::from_secs(2);
    while owner.pending().unwrap() == 0 {
        assert!(
            std::time::Instant::now() < deadline,
            "expected one queued request before deadline"
        );
        thread::yield_now();
    }
}
#[test]
fn owner_thread_calls_refuse_before_enqueue_or_wakeup() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let error = caller.call("owner", b"payload").unwrap_err().to_string();
    assert!(error.contains("owner reentry is refused"));
    assert_eq!(owner.pending().unwrap(), 0);
    assert!(awake.try_recv().is_err());
    assert_eq!(owner.drain(0, &RecordingCaller::default()).unwrap(), 0);
}
