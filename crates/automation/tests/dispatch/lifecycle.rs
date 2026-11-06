// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn accepted_timeout_does_not_cancel_the_later_owner_mutation() {
    let (mut owner, caller, awake) = timed_fixture(1, Duration::from_millis(30));
    let worker = start_call(caller, "late", b"accepted");
    wait_awake(&awake);
    let error = worker.join().unwrap().unwrap_err().to_string();
    assert!(error.contains("accepted request may still execute"));
    let host = RecordingCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(host.requests.borrow()[0].1, b"accepted");
}
#[test]
fn zero_deadline_refuses_before_admission_and_wakeup() {
    let (mut owner, caller, awake) = timed_fixture(1, Duration::ZERO);
    let error = start_call(caller, "expired", b"")
        .join()
        .unwrap()
        .unwrap_err()
        .to_string();
    assert!(error.contains("request was not accepted"));
    assert_eq!(owner.pending().unwrap(), 0);
    assert!(awake.try_recv().is_err());
    assert_eq!(owner.drain(0, &RecordingCaller::default()).unwrap(), 0);
}
#[test]
fn full_queue_timeout_does_not_admit_or_wake_a_second_request() {
    for capacity in [0, 1] {
        verify_full_queue_timeout(capacity);
    }
}
fn verify_full_queue_timeout(capacity: usize) {
    let (mut owner, caller, awake) = timed_fixture(capacity, Duration::from_millis(30));
    let first = start_call(caller.clone(), "first", b"one");
    wait_awake(&awake);
    let second = start_call(caller, "blocked", b"two");
    assert!(
        second
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("request was not accepted")
    );
    assert!(first.join().unwrap().is_err());
    assert!(awake.try_recv().is_err());
    let host = RecordingCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(host.requests.borrow().len(), 1);
}
#[test]
fn close_is_idempotent_releases_queued_and_capacity_waiters_and_refuses_future_calls() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let first = start_call(caller.clone(), "first", b"one");
    wait_awake(&awake);
    let second = start_call(caller.clone(), "second", b"two");
    owner.close();
    owner.close();
    for worker in [first, second, start_call(caller, "future", b"")] {
        assert!(
            worker
                .join()
                .unwrap()
                .unwrap_err()
                .to_string()
                .contains("dispatcher closed")
        );
    }
    assert_eq!(owner.pending().unwrap(), 0);
    assert_eq!(owner.drain(0, &RecordingCaller::default()).unwrap(), 0);
}
#[test]
fn dropping_the_owner_releases_an_accepted_waiter() {
    let (owner, caller, awake) = queue_fixture(1);
    let worker = start_call(caller, "drop", b"one");
    wait_awake(&awake);
    drop(owner);
    assert!(
        worker
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("dispatcher closed")
    );
}
struct RefusingCaller;
impl Caller for RefusingCaller {
    fn call(&self, method: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        Err(Error(format!(
            "refused method {method}: expected fixture success"
        )))
    }
}
#[test]
fn domain_errors_return_to_their_waiter_without_stopping_admission() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let worker = start_call(caller.clone(), "decline", b"");
    wait_awake(&awake);
    assert_eq!(owner.drain(1, &RefusingCaller).unwrap(), 1);
    assert!(
        worker
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("refused method decline")
    );
    let next = start_call(caller, "next", b"healthy");
    wait_awake(&awake);
    owner.drain(0, &RecordingCaller::default()).unwrap();
    assert_eq!(next.join().unwrap().unwrap(), b"healthy");
}
struct PanickingCaller;
impl Caller for PanickingCaller {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        panic!("owned host panic fixture")
    }
}
#[test]
fn a_host_panic_returns_an_error_and_stops_admission_before_any_more_model_calls() {
    let (mut owner, caller, awake) = queue_fixture(2);
    let first = start_call(caller.clone(), "panic", b"");
    wait_awake(&awake);
    let second = start_call(caller.clone(), "abandoned", b"");
    wait_awake(&awake);
    assert_eq!(owner.drain(0, &PanickingCaller).unwrap(), 1);
    assert!(
        first
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("panicked")
    );
    assert_closed(second);
    assert!(start_call(caller, "future", b"").join().unwrap().is_err());
    assert_eq!(owner.pending().unwrap(), 0);
}
struct FailedWakeup;
impl DispatchWakeup for FailedWakeup {
    fn wake(&self) -> Result<(), String> {
        Err("window wake refused".into())
    }
}
#[test]
fn wake_failure_reports_acceptance_and_cannot_revoke_the_queued_request() {
    let (mut owner, caller) = owner_dispatch(1, Duration::from_secs(2), FailedWakeup);
    let error = start_call(caller, "wake", b"queued")
        .join()
        .unwrap()
        .unwrap_err()
        .to_string();
    assert!(error.contains("window wake refused"));
    assert!(error.contains("accepted request may still execute"));
    let host = RecordingCaller::default();
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(host.requests.borrow()[0].1, b"queued");
}

fn assert_closed(worker: JoinHandle<Result<Vec<u8>, Error>>) {
    assert!(
        worker
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("dispatcher closed")
    );
}

#[test]
fn unrepresentable_deadline_is_refused_before_admission() {
    let (owner, caller, awake) = timed_fixture(1, Duration::MAX);
    let error = start_call(caller, "deadline", b"")
        .join()
        .unwrap()
        .unwrap_err()
        .to_string();
    assert!(error.contains("expected a representable monotonic deadline"));
    assert_eq!(owner.pending().unwrap(), 0);
    assert!(awake.try_recv().is_err());
}
struct ReentrantOwnerCaller<W: DispatchWakeup> {
    caller: QueuedCaller<W>,
}
impl<W: DispatchWakeup> Caller for ReentrantOwnerCaller<W> {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        self.caller.call("nested", b"")
    }
}
#[test]
fn an_owner_callback_cannot_wait_on_the_queue_it_is_currently_draining() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let worker = start_call(caller.clone(), "callback", b"");
    wait_awake(&awake);
    let host = ReentrantOwnerCaller { caller };
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert!(
        worker
            .join()
            .unwrap()
            .unwrap_err()
            .to_string()
            .contains("owner reentry is refused")
    );
    assert_eq!(owner.pending().unwrap(), 0);
}

struct IncomingWakeCaller {
    trigger: Sender<()>,
    awake: Receiver<()>,
}
impl Caller for IncomingWakeCaller {
    fn call(&self, _: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        self.trigger
            .send(())
            .map_err(|error| Error(error.to_string()))?;
        self.awake
            .recv_timeout(Duration::from_secs(2))
            .map_err(|error| Error(error.to_string()))?;
        Ok(request.to_vec())
    }
}
#[test]
fn zero_frame_budget_drains_the_snapshot_and_defers_new_arrivals_until_next_frame() {
    let (mut owner, caller, awake) = queue_fixture(1);
    let first = start_call(caller.clone(), "first", b"one");
    wait_awake(&awake);
    let (trigger, ready) = mpsc::channel();
    let second = thread::spawn(move || {
        ready.recv_timeout(Duration::from_secs(2)).unwrap();
        caller.call("second", b"two")
    });
    let host = IncomingWakeCaller { trigger, awake };
    assert_eq!(owner.drain(0, &host).unwrap(), 1);
    assert_eq!(owner.pending().unwrap(), 1);
    assert_eq!(first.join().unwrap().unwrap(), b"one");
    assert_eq!(owner.drain(0, &RecordingCaller::default()).unwrap(), 1);
    assert_eq!(second.join().unwrap().unwrap(), b"two");
}
