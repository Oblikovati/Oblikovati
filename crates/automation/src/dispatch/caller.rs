// SPDX-License-Identifier: GPL-2.0-only
use super::*;
/// Cloneable worker transport; only copied method names and payload bytes cross to the owner.
pub struct QueuedCaller<W: DispatchWakeup> {
    pub(super) shared: Arc<SharedQueue<W>>,
}
impl<W: DispatchWakeup> Clone for QueuedCaller<W> {
    fn clone(&self) -> Self {
        Self {
            shared: Arc::clone(&self.shared),
        }
    }
}
impl<W: DispatchWakeup> Caller for QueuedCaller<W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if thread::current().id() == self.shared.owner {
            return Err(Error(format!(
                "dispatch method {method:?}: expected a worker caller; synchronous owner reentry is refused"
            )));
        }
        let deadline = self.deadline(method)?;
        let (reply, response) = mpsc::channel();
        let queued = QueuedRequest {
            method: method.into(),
            arguments: request.to_vec(),
            reply,
        };
        self.enqueue(queued, deadline)?;
        self.wait_reply(method, response, deadline)
    }
}
impl<W: DispatchWakeup> QueuedCaller<W> {
    fn deadline(&self, method: &str) -> Result<Instant, Error> {
        Instant::now().checked_add(self.shared.timeout).ok_or_else(|| Error(format!(
            "dispatch method {method:?} timeout {:?}: expected a representable monotonic deadline", self.shared.timeout
        )))
    }
    fn enqueue(&self, request: QueuedRequest, deadline: Instant) -> Result<(), Error> {
        let method = request.method.clone();
        let mut status = self.shared.lock(&method)?;
        while !status.closed && status.requests.len() >= self.shared.capacity {
            status = self.shared.wait_capacity(status, deadline, &method)?;
        }
        if status.closed {
            return Err(self.shared.closed_error(&method));
        }
        if Instant::now() >= deadline {
            return Err(self.shared.timeout_error(&method, false));
        }
        status.requests.push_back(request);
        drop(status);
        // Admission is complete before wakeup. Failure cannot revoke the accepted mutation (#1493).
        self.shared.wakeup.wake().map_err(|cause| Error(format!(
            "dispatch method {method:?} wake failed: {cause}; expected an owner signal; accepted request may still execute"
        )))
    }
    fn wait_reply(
        &self,
        method: &str,
        response: Receiver<Result<Vec<u8>, Error>>,
        deadline: Instant,
    ) -> Result<Vec<u8>, Error> {
        match response.recv_timeout(deadline.saturating_duration_since(Instant::now())) {
            Ok(reply) => reply,
            Err(mpsc::RecvTimeoutError::Timeout) => Err(self.shared.timeout_error(method, true)),
            Err(mpsc::RecvTimeoutError::Disconnected) => Err(self.shared.closed_error(method)),
        }
    }
}
